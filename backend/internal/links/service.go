// Package links concentra o domínio de links: validação, geração de slug,
// detecção de colisão e o CRUD sobre a tabela links.
package links

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"encurtador/internal/database/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service implementa as operações de domínio sobre links.
type Service struct {
	q      *db.Queries
	logger *slog.Logger
}

// NewService cria o serviço de links a partir do pool de conexões.
func NewService(pool *pgxpool.Pool, logger *slog.Logger) *Service {
	return &Service{q: db.New(pool), logger: logger}
}

// CreateInput reúne os dados para criação de um link.
type CreateInput struct {
	TargetURL  string
	CustomSlug string // opcional; vazio = gerar base62
	Title      *string
	ExpiresAt  *time.Time
}

// Create valida e persiste um novo link. Com slug customizado, colisão retorna
// ErrSlugTaken; sem slug, gera base62 com retry em caso de colisão.
func (s *Service) Create(ctx context.Context, in CreateInput) (db.Link, error) {
	if err := ValidateTargetURL(in.TargetURL); err != nil {
		return db.Link{}, err
	}

	if in.CustomSlug != "" {
		if err := ValidateSlugFormat(in.CustomSlug); err != nil {
			return db.Link{}, err
		}
		link, err := s.createWithSlug(ctx, in.CustomSlug, in)
		if isUniqueViolation(err) {
			return db.Link{}, ErrSlugTaken
		}
		return link, err
	}

	for attempt := 0; attempt < maxSlugAttempts; attempt++ {
		slug, err := generateSlug()
		if err != nil {
			return db.Link{}, err
		}
		link, err := s.createWithSlug(ctx, slug, in)
		if err == nil {
			return link, nil
		}
		if isUniqueViolation(err) {
			s.logger.Warn("colisão de slug gerado, tentando novamente", "slug", slug, "attempt", attempt+1)
			continue
		}
		return db.Link{}, err
	}
	return db.Link{}, ErrSlugGenFailed
}

func (s *Service) createWithSlug(ctx context.Context, slug string, in CreateInput) (db.Link, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return db.Link{}, err
	}
	return s.q.CreateLink(ctx, db.CreateLinkParams{
		ID:        id,
		Slug:      slug,
		TargetUrl: in.TargetURL,
		Title:     in.Title,
		ExpiresAt: in.ExpiresAt,
		IsActive:  true,
	})
}

// GetByID retorna um link pelo id, ou ErrNotFound.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (db.Link, error) {
	link, err := s.q.GetLinkByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Link{}, ErrNotFound
	}
	return link, err
}

// GetBySlug retorna um link pelo slug, ou ErrNotFound.
func (s *Service) GetBySlug(ctx context.Context, slug string) (db.Link, error) {
	link, err := s.q.GetLinkBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Link{}, ErrNotFound
	}
	return link, err
}

// ListInput controla a paginação da listagem.
type ListInput struct {
	Limit  int32
	Offset int32
}

// List retorna uma página de links e o total geral.
func (s *Service) List(ctx context.Context, in ListInput) ([]db.Link, int64, error) {
	links, err := s.q.ListLinks(ctx, db.ListLinksParams{Limit: in.Limit, Offset: in.Offset})
	if err != nil {
		return nil, 0, err
	}
	total, err := s.q.CountLinks(ctx)
	if err != nil {
		return nil, 0, err
	}
	return links, total, nil
}

// UpdateInput descreve uma atualização parcial (PATCH). Campos nil ficam
// inalterados.
type UpdateInput struct {
	TargetURL *string
	Title     *string
	ExpiresAt *time.Time
	IsActive  *bool
}

// Update aplica uma atualização parcial sobre um link existente.
func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (db.Link, error) {
	current, err := s.GetByID(ctx, id)
	if err != nil {
		return db.Link{}, err
	}

	params := db.UpdateLinkParams{
		ID:        id,
		TargetUrl: current.TargetUrl,
		Title:     current.Title,
		ExpiresAt: current.ExpiresAt,
		IsActive:  current.IsActive,
	}

	if in.TargetURL != nil {
		if err := ValidateTargetURL(*in.TargetURL); err != nil {
			return db.Link{}, err
		}
		params.TargetUrl = *in.TargetURL
	}
	if in.Title != nil {
		params.Title = in.Title
	}
	if in.ExpiresAt != nil {
		params.ExpiresAt = in.ExpiresAt
	}
	if in.IsActive != nil {
		params.IsActive = *in.IsActive
	}

	return s.q.UpdateLink(ctx, params)
}

// Delete remove um link, ou retorna ErrNotFound.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	n, err := s.q.DeleteLink(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// isUniqueViolation detecta violação de unicidade do Postgres (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
