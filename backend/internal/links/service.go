// Package links concentra o domínio de links: validação, geração de slug,
// detecção de colisão e o CRUD sobre a tabela links.
package links

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"encurtador/internal/database/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	StatusActive   = "active"
	StatusInactive = "inactive"
	StatusExpired  = "expired"

	MaxTags      = 10
	MaxTagLength = 32
)

var tagPattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

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
	TargetURL   string
	CustomSlug  string // opcional; vazio = gerar base62
	Title       *string
	ExpiresAt   *time.Time
	Campaign    *string
	Tags        []string
	UtmSource   *string
	UtmMedium   *string
	UtmCampaign *string
	UtmTerm     *string
	UtmContent  *string
	Notes       *string
}

// LinkWithStats representa um link com métricas derivadas para listagem/export.
type LinkWithStats struct {
	Link          db.Link
	TotalClicks   int64
	LastClickedAt *time.Time
}

// Create valida e persiste um novo link. Com slug customizado, colisão retorna
// ErrSlugTaken; sem slug, gera base62 com retry em caso de colisão.
func (s *Service) Create(ctx context.Context, in CreateInput) (db.Link, error) {
	if err := ValidateTargetURL(in.TargetURL); err != nil {
		return db.Link{}, err
	}
	tags, err := NormalizeTags(in.Tags)
	if err != nil {
		return db.Link{}, err
	}
	in.Tags = tags
	in.Campaign = cleanOptional(in.Campaign)
	in.UtmSource = cleanOptional(in.UtmSource)
	in.UtmMedium = cleanOptional(in.UtmMedium)
	in.UtmCampaign = cleanOptional(in.UtmCampaign)
	in.UtmTerm = cleanOptional(in.UtmTerm)
	in.UtmContent = cleanOptional(in.UtmContent)
	in.Notes = cleanOptional(in.Notes)

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
		ID:          id,
		Slug:        slug,
		TargetUrl:   in.TargetURL,
		Title:       cleanOptional(in.Title),
		ExpiresAt:   in.ExpiresAt,
		IsActive:    true,
		Campaign:    in.Campaign,
		Tags:        in.Tags,
		UtmSource:   in.UtmSource,
		UtmMedium:   in.UtmMedium,
		UtmCampaign: in.UtmCampaign,
		UtmTerm:     in.UtmTerm,
		UtmContent:  in.UtmContent,
		Notes:       in.Notes,
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

// ListInput controla a paginação e os filtros da listagem.
type ListInput struct {
	Limit    int32
	Offset   int32
	Q        string
	Status   string
	Tag      string
	Campaign string
}

// List retorna uma página de links, métricas derivadas e o total filtrado.
func (s *Service) List(ctx context.Context, in ListInput) ([]LinkWithStats, int64, error) {
	params, err := listParams(in)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.q.ListLinks(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.q.CountLinks(ctx, db.CountLinksParams{
		Q:        params.Q,
		Status:   params.Status,
		Tag:      params.Tag,
		Campaign: params.Campaign,
	})
	if err != nil {
		return nil, 0, err
	}

	items := make([]LinkWithStats, 0, len(rows))
	for _, row := range rows {
		items = append(items, rowToLinkWithStats(row))
	}
	return items, total, nil
}

func listParams(in ListInput) (db.ListLinksParams, error) {
	status := strings.TrimSpace(strings.ToLower(in.Status))
	switch status {
	case "", StatusActive, StatusInactive, StatusExpired:
	default:
		return db.ListLinksParams{}, ErrInvalidStatus
	}

	tag := strings.TrimSpace(in.Tag)
	if tag != "" {
		normalized, err := NormalizeTags([]string{tag})
		if err != nil || len(normalized) != 1 {
			return db.ListLinksParams{}, ErrInvalidTags
		}
		tag = normalized[0]
	}

	return db.ListLinksParams{
		Q:        strings.TrimSpace(in.Q),
		Status:   status,
		Tag:      tag,
		Campaign: strings.TrimSpace(in.Campaign),
		Limit:    in.Limit,
		Offset:   in.Offset,
	}, nil
}

func rowToLinkWithStats(row db.ListLinksRow) LinkWithStats {
	link := db.Link{
		ID:          row.ID,
		Slug:        row.Slug,
		TargetUrl:   row.TargetUrl,
		Title:       row.Title,
		CreatedAt:   row.CreatedAt,
		ExpiresAt:   row.ExpiresAt,
		IsActive:    row.IsActive,
		Campaign:    row.Campaign,
		Tags:        row.Tags,
		UtmSource:   row.UtmSource,
		UtmMedium:   row.UtmMedium,
		UtmCampaign: row.UtmCampaign,
		UtmTerm:     row.UtmTerm,
		UtmContent:  row.UtmContent,
		Notes:       row.Notes,
		UpdatedAt:   row.UpdatedAt,
	}
	var lastClickedAt *time.Time
	if row.HasLastClickedAt {
		value := row.LastClickedAt
		lastClickedAt = &value
	}
	return LinkWithStats{Link: link, TotalClicks: row.TotalClicks, LastClickedAt: lastClickedAt}
}

// UpdateInput descreve uma atualização parcial (PATCH). Campos nil ficam
// inalterados.
type UpdateInput struct {
	TargetURL   *string
	Title       *string
	ExpiresAt   *time.Time
	IsActive    *bool
	Campaign    *string
	Tags        *[]string
	UtmSource   *string
	UtmMedium   *string
	UtmCampaign *string
	UtmTerm     *string
	UtmContent  *string
	Notes       *string
}

// Update aplica uma atualização parcial sobre um link existente.
func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (db.Link, error) {
	current, err := s.GetByID(ctx, id)
	if err != nil {
		return db.Link{}, err
	}

	params := db.UpdateLinkParams{
		ID:          id,
		TargetUrl:   current.TargetUrl,
		Title:       current.Title,
		ExpiresAt:   current.ExpiresAt,
		IsActive:    current.IsActive,
		Campaign:    current.Campaign,
		Tags:        current.Tags,
		UtmSource:   current.UtmSource,
		UtmMedium:   current.UtmMedium,
		UtmCampaign: current.UtmCampaign,
		UtmTerm:     current.UtmTerm,
		UtmContent:  current.UtmContent,
		Notes:       current.Notes,
	}

	if params.Tags == nil {
		params.Tags = []string{}
	}
	if in.TargetURL != nil {
		if err := ValidateTargetURL(*in.TargetURL); err != nil {
			return db.Link{}, err
		}
		params.TargetUrl = *in.TargetURL
	}
	if in.Title != nil {
		params.Title = cleanOptional(in.Title)
	}
	if in.ExpiresAt != nil {
		params.ExpiresAt = in.ExpiresAt
	}
	if in.IsActive != nil {
		params.IsActive = *in.IsActive
	}
	if in.Campaign != nil {
		params.Campaign = cleanOptional(in.Campaign)
	}
	if in.Tags != nil {
		tags, err := NormalizeTags(*in.Tags)
		if err != nil {
			return db.Link{}, err
		}
		params.Tags = tags
	}
	if in.UtmSource != nil {
		params.UtmSource = cleanOptional(in.UtmSource)
	}
	if in.UtmMedium != nil {
		params.UtmMedium = cleanOptional(in.UtmMedium)
	}
	if in.UtmCampaign != nil {
		params.UtmCampaign = cleanOptional(in.UtmCampaign)
	}
	if in.UtmTerm != nil {
		params.UtmTerm = cleanOptional(in.UtmTerm)
	}
	if in.UtmContent != nil {
		params.UtmContent = cleanOptional(in.UtmContent)
	}
	if in.Notes != nil {
		params.Notes = cleanOptional(in.Notes)
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

// NormalizeTags valida, remove duplicatas e preserva a ordem das tags.
func NormalizeTags(tags []string) ([]string, error) {
	out := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, raw := range tags {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			continue
		}
		if len(tag) > MaxTagLength || !tagPattern.MatchString(tag) {
			return nil, ErrInvalidTags
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
		if len(out) > MaxTags {
			return nil, ErrInvalidTags
		}
	}
	return out, nil
}

func cleanOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// isUniqueViolation detecta violação de unicidade do Postgres (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
