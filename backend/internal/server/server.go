// Package server monta o router Gin e os handlers HTTP.
package server

import (
	"log/slog"
	"net/http"

	"encurtador/internal/clicks"
	"encurtador/internal/config"
	"encurtador/internal/links"
	"encurtador/internal/stats"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server agrupa as dependências dos handlers.
type Server struct {
	cfg    *config.Config
	pool   *pgxpool.Pool
	logger *slog.Logger
	engine *gin.Engine
	links  *links.Service
	clicks *clicks.Service
	stats  *stats.Repository
}

// New cria o servidor com o router configurado.
func New(cfg *config.Config, pool *pgxpool.Pool, logger *slog.Logger, clickService *clicks.Service, statsRepo *stats.Repository) *Server {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	s := &Server{
		cfg:    cfg,
		pool:   pool,
		logger: logger,
		links:  links.NewService(pool, logger),
		clicks: clickService,
		stats:  statsRepo,
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(s.cors())
	s.engine = engine
	s.registerRoutes()

	return s
}

// Handler expõe o http.Handler do servidor.
func (s *Server) Handler() http.Handler {
	return s.engine
}

func (s *Server) registerRoutes() {
	s.engine.GET("/healthz", s.handleHealthz)

	api := s.engine.Group("/api", s.apiKeyAuth())
	{
		api.POST("/links", s.handleCreateLink)
		api.GET("/links", s.handleListLinks)
		api.GET("/links/export.csv", s.handleExportLinksCSV)
		api.GET("/links/:id/stats", s.handleGetLinkStats)
		api.GET("/links/:id", s.handleGetLink)
		api.PATCH("/links/:id", s.handlePatchLink)
		api.DELETE("/links/:id", s.handleDeleteLink)
	}

	s.engine.NoRoute(s.handleRedirect)
}
