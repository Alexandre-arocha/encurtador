// Package database cuida da conexão com o Postgres e da aplicação das
// migrations embarcadas (golang-migrate + embed.FS).
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"embed"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // registra o driver sql "pgx"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

// all: inclui também eventuais arquivos ocultos (ex.: .gitkeep) para que o
// embed funcione mesmo quando ainda não há nenhuma migration .sql.
//
//go:embed all:migrations
var migrationsFS embed.FS

// Migrate aplica todas as migrations pendentes. É seguro chamar no boot:
// se não houver migrations, vira no-op.
func Migrate(databaseURL string, logger *slog.Logger) error {
	has, err := hasMigrations()
	if err != nil {
		return fmt.Errorf("erro ao ler migrations embarcadas: %w", err)
	}
	if !has {
		logger.Info("nenhuma migration encontrada — nada a aplicar")
		return nil
	}

	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("erro ao abrir migrations: %w", err)
	}
	defer src.Close()

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("erro ao abrir conexão para migrate: %w", err)
	}
	defer db.Close()

	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{})
	if err != nil {
		return fmt.Errorf("erro ao inicializar driver de migrate: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "pgx5", driver)
	if err != nil {
		return fmt.Errorf("erro ao inicializar migrate: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("erro ao aplicar migrations: %w", err)
	}

	logger.Info("migrations aplicadas com sucesso")
	return nil
}

// MigrateRiver aplica as migrations internas que o River precisa para filas.
func MigrateRiver(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), &rivermigrate.Config{Logger: logger})
	if err != nil {
		return fmt.Errorf("erro ao inicializar migrations do River: %w", err)
	}

	result, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{})
	if err != nil {
		return fmt.Errorf("erro ao aplicar migrations do River: %w", err)
	}
	logger.Info("migrations do River aplicadas", "versions", len(result.Versions))
	return nil
}

// hasMigrations diz se existe pelo menos um arquivo .sql em migrations/.
func hasMigrations() (bool, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			return true, nil
		}
	}
	return false, nil
}
