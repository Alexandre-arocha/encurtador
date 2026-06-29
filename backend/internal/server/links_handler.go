package server

import (
	"encoding/csv"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"encurtador/internal/database/db"
	"encurtador/internal/links"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	defaultListLimit = 50
	maxListLimit     = 100
	exportLimit      = 5000
)

// --- DTOs ---

type createLinkRequest struct {
	TargetURL   string     `json:"target_url"`
	Slug        string     `json:"slug"`
	Title       *string    `json:"title"`
	ExpiresAt   *time.Time `json:"expires_at"`
	Campaign    *string    `json:"campaign"`
	Tags        []string   `json:"tags"`
	UtmSource   *string    `json:"utm_source"`
	UtmMedium   *string    `json:"utm_medium"`
	UtmCampaign *string    `json:"utm_campaign"`
	UtmTerm     *string    `json:"utm_term"`
	UtmContent  *string    `json:"utm_content"`
	Notes       *string    `json:"notes"`
}

type updateLinkRequest struct {
	TargetURL   *string    `json:"target_url"`
	Title       *string    `json:"title"`
	ExpiresAt   *time.Time `json:"expires_at"`
	IsActive    *bool      `json:"is_active"`
	Campaign    *string    `json:"campaign"`
	Tags        *[]string  `json:"tags"`
	UtmSource   *string    `json:"utm_source"`
	UtmMedium   *string    `json:"utm_medium"`
	UtmCampaign *string    `json:"utm_campaign"`
	UtmTerm     *string    `json:"utm_term"`
	UtmContent  *string    `json:"utm_content"`
	Notes       *string    `json:"notes"`
}

type linkResponse struct {
	ID            string     `json:"id"`
	Slug          string     `json:"slug"`
	ShortURL      string     `json:"short_url"`
	TargetURL     string     `json:"target_url"`
	Title         *string    `json:"title"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	ExpiresAt     *time.Time `json:"expires_at"`
	IsActive      bool       `json:"is_active"`
	Campaign      *string    `json:"campaign"`
	Tags          []string   `json:"tags"`
	UtmSource     *string    `json:"utm_source"`
	UtmMedium     *string    `json:"utm_medium"`
	UtmCampaign   *string    `json:"utm_campaign"`
	UtmTerm       *string    `json:"utm_term"`
	UtmContent    *string    `json:"utm_content"`
	Notes         *string    `json:"notes"`
	TotalClicks   int64      `json:"total_clicks"`
	LastClickedAt *time.Time `json:"last_clicked_at"`
}

type listLinksResponse struct {
	Data   []linkResponse `json:"data"`
	Total  int64          `json:"total"`
	Limit  int32          `json:"limit"`
	Offset int32          `json:"offset"`
}

func (s *Server) toLinkResponse(l db.Link) linkResponse {
	return s.toLinkResponseWithStats(links.LinkWithStats{Link: l})
}

func (s *Server) toLinkResponseWithStats(item links.LinkWithStats) linkResponse {
	l := item.Link
	tags := l.Tags
	if tags == nil {
		tags = []string{}
	}
	return linkResponse{
		ID:            l.ID.String(),
		Slug:          l.Slug,
		ShortURL:      s.cfg.BaseURL + "/" + l.Slug,
		TargetURL:     l.TargetUrl,
		Title:         l.Title,
		CreatedAt:     l.CreatedAt,
		UpdatedAt:     l.UpdatedAt,
		ExpiresAt:     l.ExpiresAt,
		IsActive:      l.IsActive,
		Campaign:      l.Campaign,
		Tags:          tags,
		UtmSource:     l.UtmSource,
		UtmMedium:     l.UtmMedium,
		UtmCampaign:   l.UtmCampaign,
		UtmTerm:       l.UtmTerm,
		UtmContent:    l.UtmContent,
		Notes:         l.Notes,
		TotalClicks:   item.TotalClicks,
		LastClickedAt: item.LastClickedAt,
	}
}

// --- Handlers ---

func (s *Server) handleCreateLink(c *gin.Context) {
	var req createLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, "invalid_body", "corpo da requisição inválido")
		return
	}

	link, err := s.links.Create(c.Request.Context(), links.CreateInput{
		TargetURL:   req.TargetURL,
		CustomSlug:  req.Slug,
		Title:       req.Title,
		ExpiresAt:   req.ExpiresAt,
		Campaign:    req.Campaign,
		Tags:        req.Tags,
		UtmSource:   req.UtmSource,
		UtmMedium:   req.UtmMedium,
		UtmCampaign: req.UtmCampaign,
		UtmTerm:     req.UtmTerm,
		UtmContent:  req.UtmContent,
		Notes:       req.Notes,
	})
	if err != nil {
		s.respondLinkError(c, err)
		return
	}

	c.JSON(http.StatusCreated, s.toLinkResponse(link))
}

func (s *Server) handleListLinks(c *gin.Context) {
	limit := parseIntQuery(c, "limit", defaultListLimit)
	if limit <= 0 || limit > maxListLimit {
		limit = defaultListLimit
	}
	offset := parseIntQuery(c, "offset", 0)
	if offset < 0 {
		offset = 0
	}

	input := listInputFromQuery(c, int32(limit), int32(offset))
	items, total, err := s.links.List(c.Request.Context(), input)
	if err != nil {
		s.respondLinkError(c, err)
		return
	}

	data := make([]linkResponse, 0, len(items))
	for _, item := range items {
		data = append(data, s.toLinkResponseWithStats(item))
	}
	c.JSON(http.StatusOK, listLinksResponse{
		Data:   data,
		Total:  total,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
}

func (s *Server) handleExportLinksCSV(c *gin.Context) {
	items, _, err := s.links.List(c.Request.Context(), listInputFromQuery(c, exportLimit, 0))
	if err != nil {
		s.respondLinkError(c, err)
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="encurtador-links.csv"`)
	c.Status(http.StatusOK)

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	_ = writer.Write([]string{
		"id",
		"slug",
		"short_url",
		"target_url",
		"title",
		"campaign",
		"tags",
		"utm_source",
		"utm_medium",
		"utm_campaign",
		"utm_term",
		"utm_content",
		"notes",
		"is_active",
		"expires_at",
		"created_at",
		"updated_at",
		"total_clicks",
		"last_clicked_at",
	})
	for _, item := range items {
		l := item.Link
		lastClickedAt := ""
		if item.LastClickedAt != nil {
			lastClickedAt = item.LastClickedAt.Format(time.RFC3339)
		}
		_ = writer.Write([]string{
			l.ID.String(),
			l.Slug,
			s.cfg.BaseURL + "/" + l.Slug,
			l.TargetUrl,
			stringValue(l.Title),
			stringValue(l.Campaign),
			strings.Join(l.Tags, "|"),
			stringValue(l.UtmSource),
			stringValue(l.UtmMedium),
			stringValue(l.UtmCampaign),
			stringValue(l.UtmTerm),
			stringValue(l.UtmContent),
			stringValue(l.Notes),
			strconv.FormatBool(l.IsActive),
			timeValue(l.ExpiresAt),
			l.CreatedAt.Format(time.RFC3339),
			l.UpdatedAt.Format(time.RFC3339),
			strconv.FormatInt(item.TotalClicks, 10),
			lastClickedAt,
		})
	}
}

func (s *Server) handleGetLink(c *gin.Context) {
	id, ok := s.parseLinkID(c)
	if !ok {
		return
	}

	link, err := s.links.GetByID(c.Request.Context(), id)
	if err != nil {
		s.respondLinkError(c, err)
		return
	}
	c.JSON(http.StatusOK, s.toLinkResponse(link))
}

func (s *Server) handlePatchLink(c *gin.Context) {
	id, ok := s.parseLinkID(c)
	if !ok {
		return
	}

	var req updateLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, "invalid_body", "corpo da requisição inválido")
		return
	}

	link, err := s.links.Update(c.Request.Context(), id, links.UpdateInput{
		TargetURL:   req.TargetURL,
		Title:       req.Title,
		ExpiresAt:   req.ExpiresAt,
		IsActive:    req.IsActive,
		Campaign:    req.Campaign,
		Tags:        req.Tags,
		UtmSource:   req.UtmSource,
		UtmMedium:   req.UtmMedium,
		UtmCampaign: req.UtmCampaign,
		UtmTerm:     req.UtmTerm,
		UtmContent:  req.UtmContent,
		Notes:       req.Notes,
	})
	if err != nil {
		s.respondLinkError(c, err)
		return
	}
	c.JSON(http.StatusOK, s.toLinkResponse(link))
}

func (s *Server) handleDeleteLink(c *gin.Context) {
	id, ok := s.parseLinkID(c)
	if !ok {
		return
	}

	if err := s.links.Delete(c.Request.Context(), id); err != nil {
		s.respondLinkError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Helpers ---

func listInputFromQuery(c *gin.Context, limit, offset int32) links.ListInput {
	return links.ListInput{
		Limit:    limit,
		Offset:   offset,
		Q:        c.Query("q"),
		Status:   c.Query("status"),
		Tag:      c.Query("tag"),
		Campaign: c.Query("campaign"),
	}
}

func (s *Server) parseLinkID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		s.respondError(c, http.StatusBadRequest, "invalid_id", "id inválido (esperado UUID)")
		return uuid.Nil, false
	}
	return id, true
}

func parseIntQuery(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func timeValue(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

// respondLinkError mapeia erros de domínio de links para status HTTP.
func (s *Server) respondLinkError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, links.ErrNotFound):
		s.respondError(c, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, links.ErrSlugTaken):
		s.respondError(c, http.StatusConflict, "slug_taken", err.Error())
	case errors.Is(err, links.ErrReservedSlug):
		s.respondError(c, http.StatusBadRequest, "reserved_slug", err.Error())
	case errors.Is(err, links.ErrInvalidSlug):
		s.respondError(c, http.StatusBadRequest, "invalid_slug", err.Error())
	case errors.Is(err, links.ErrInvalidURL):
		s.respondError(c, http.StatusBadRequest, "invalid_url", err.Error())
	case errors.Is(err, links.ErrInvalidTags):
		s.respondError(c, http.StatusBadRequest, "invalid_tags", err.Error())
	case errors.Is(err, links.ErrInvalidStatus):
		s.respondError(c, http.StatusBadRequest, "invalid_status", err.Error())
	case errors.Is(err, links.ErrSlugGenFailed):
		s.logger.Error("falha ao gerar slug único", "err", err)
		s.respondError(c, http.StatusInternalServerError, "slug_generation_failed", err.Error())
	default:
		s.logger.Error("erro inesperado no domínio de links", "err", err)
		s.respondError(c, http.StatusInternalServerError, "internal_error", "erro interno")
	}
}
