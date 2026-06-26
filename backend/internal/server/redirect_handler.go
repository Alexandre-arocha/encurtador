package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"encurtador/internal/clicks"
	"encurtador/internal/links"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleRedirect(c *gin.Context) {
	if c.Request.Method != http.MethodGet {
		s.respondError(c, http.StatusNotFound, "not_found", "rota não encontrada")
		return
	}

	slug := strings.TrimPrefix(c.Request.URL.Path, "/")
	if slug == "" || strings.Contains(slug, "/") {
		s.respondError(c, http.StatusNotFound, "not_found", "rota não encontrada")
		return
	}

	link, err := s.links.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, links.ErrNotFound) {
			s.respondError(c, http.StatusNotFound, "not_found", "link não encontrado")
			return
		}
		s.logger.Error("erro ao resolver slug", "slug", slug, "err", err)
		s.respondError(c, http.StatusInternalServerError, "internal_error", "erro ao resolver link")
		return
	}

	if !link.IsActive || (link.ExpiresAt != nil && !link.ExpiresAt.After(time.Now())) {
		s.respondError(c, http.StatusGone, "link_unavailable", "link expirado ou inativo")
		return
	}

	rawIP := c.ClientIP()
	referrer := c.GetHeader("Referer")
	uaRaw := c.GetHeader("User-Agent")

	c.Redirect(http.StatusFound, link.TargetUrl)

	if s.clicks != nil {
		s.clicks.CaptureAsync(clicks.CaptureInput{
			LinkID:   link.ID,
			IP:       rawIP,
			Referrer: referrer,
			UARaw:    uaRaw,
		})
	}
}
