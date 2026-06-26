package server

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"encurtador/internal/database/db"
	"encurtador/internal/links"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	defaultListLimit = 50
	maxListLimit     = 100
)

// --- DTOs ---

type createLinkRequest struct {
	TargetURL string     `json:"target_url"`
	Slug      string     `json:"slug"`
	Title     *string    `json:"title"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type updateLinkRequest struct {
	TargetURL *string    `json:"target_url"`
	Title     *string    `json:"title"`
	ExpiresAt *time.Time `json:"expires_at"`
	IsActive  *bool      `json:"is_active"`
}

type linkResponse struct {
	ID        string     `json:"id"`
	Slug      string     `json:"slug"`
	ShortURL  string     `json:"short_url"`
	TargetURL string     `json:"target_url"`
	Title     *string    `json:"title"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`
	IsActive  bool       `json:"is_active"`
}

type listLinksResponse struct {
	Data   []linkResponse `json:"data"`
	Total  int64          `json:"total"`
	Limit  int32          `json:"limit"`
	Offset int32          `json:"offset"`
}

func (s *Server) toLinkResponse(l db.Link) linkResponse {
	return linkResponse{
		ID:        l.ID.String(),
		Slug:      l.Slug,
		ShortURL:  s.cfg.BaseURL + "/" + l.Slug,
		TargetURL: l.TargetUrl,
		Title:     l.Title,
		CreatedAt: l.CreatedAt,
		ExpiresAt: l.ExpiresAt,
		IsActive:  l.IsActive,
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
		TargetURL:  req.TargetURL,
		CustomSlug: req.Slug,
		Title:      req.Title,
		ExpiresAt:  req.ExpiresAt,
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

	items, total, err := s.links.List(c.Request.Context(), links.ListInput{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		s.logger.Error("erro ao listar links", "err", err)
		s.respondError(c, http.StatusInternalServerError, "internal_error", "erro ao listar links")
		return
	}

	data := make([]linkResponse, 0, len(items))
	for _, l := range items {
		data = append(data, s.toLinkResponse(l))
	}
	c.JSON(http.StatusOK, listLinksResponse{
		Data:   data,
		Total:  total,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
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
		TargetURL: req.TargetURL,
		Title:     req.Title,
		ExpiresAt: req.ExpiresAt,
		IsActive:  req.IsActive,
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
	case errors.Is(err, links.ErrSlugGenFailed):
		s.logger.Error("falha ao gerar slug único", "err", err)
		s.respondError(c, http.StatusInternalServerError, "slug_generation_failed", err.Error())
	default:
		s.logger.Error("erro inesperado no domínio de links", "err", err)
		s.respondError(c, http.StatusInternalServerError, "internal_error", "erro interno")
	}
}
