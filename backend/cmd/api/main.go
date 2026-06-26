// Command api é o entrypoint do servidor do encurtador.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"encurtador/internal/clicks"
	"encurtador/internal/config"
	"encurtador/internal/database"
	"encurtador/internal/enrich"
	"encurtador/internal/server"
	"encurtador/internal/stats"

	"github.com/joho/godotenv"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Em dev, carrega o .env da raiz do repo (best-effort). Em produção as
	// variáveis vêm do ambiente, então a ausência do arquivo é normal.
	loadDotEnv(logger)

	if err := run(logger); err != nil {
		logger.Error("falha ao iniciar o serviço", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Migrations rodam no boot (idempotente).
	if err := database.Migrate(cfg.DatabaseURL, logger); err != nil {
		return err
	}

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := database.MigrateRiver(ctx, pool, logger); err != nil {
		return err
	}

	rawIPCache := clicks.NewRawIPCache(5 * time.Minute)
	enricher, closeGeo := newEnricher(cfg, logger)
	defer closeGeo()

	workers := river.NewWorkers()
	river.AddWorker(workers, clicks.NewEnrichWorker(pool, enricher, rawIPCache, logger))

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger:  logger,
		Workers: workers,
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
	})
	if err != nil {
		return err
	}
	if err := riverClient.Start(ctx); err != nil {
		return err
	}

	clickService := clicks.NewService(pool, riverClient, rawIPCache, cfg.IPHashSalt, logger)
	statsRepo := stats.NewRepository(pool)
	srv := server.New(cfg, pool, logger, clickService, statsRepo)
	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Sobe o servidor em background e aguarda sinal de término.
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("servidor iniciado", "addr", cfg.HTTPAddr, "env", cfg.AppEnv)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case <-stop:
		logger.Info("sinal de término recebido, encerrando...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("erro no shutdown do servidor", "err", err)
	}
	if err := riverClient.Stop(shutdownCtx); err != nil {
		logger.Error("erro ao encerrar River", "err", err)
	}
	return nil
}

func newEnricher(cfg *config.Config, logger *slog.Logger) (*enrich.Service, func()) {
	if cfg.GeoLiteDBPath == "" {
		logger.Warn("GEOLITE_DB_PATH vazio; enriquecimento geo ficará nulo")
		return enrich.New(nil), func() {}
	}

	var lastErr error
	for _, path := range geoPathCandidates(cfg.GeoLiteDBPath) {
		geo, err := enrich.OpenMaxMind(path)
		if err == nil {
			return enrich.New(geo), func() {
				if err := geo.Close(); err != nil {
					logger.Warn("falha ao fechar GeoLite2", "err", err)
				}
			}
		}
		lastErr = err
	}

	logger.Warn("falha ao abrir GeoLite2; enriquecimento geo ficará nulo", "path", cfg.GeoLiteDBPath, "err", lastErr)
	return enrich.New(nil), func() {}
}

func geoPathCandidates(path string) []string {
	if filepath.IsAbs(path) {
		return []string{path}
	}
	return []string{path, filepath.Join("..", path)}
}

// loadDotEnv procura um .env na raiz do repo (um nível acima de backend/) e,
// como fallback, no diretório atual.
func loadDotEnv(logger *slog.Logger) {
	for _, path := range []string{"../.env", ".env"} {
		if _, err := os.Stat(path); err == nil {
			if err := godotenv.Load(path); err != nil {
				logger.Warn("falha ao carregar .env", "path", path, "err", err)
			} else {
				logger.Info(".env carregado", "path", path)
			}
			return
		}
	}
}
