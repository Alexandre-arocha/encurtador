package server

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// apiKeyAuth protege os endpoints /api/* com a API key single-tenant.
// Aceita o header "X-API-Key: <key>" ou "Authorization: Bearer <key>".
func (s *Server) apiKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.cfg.APIKey == "" {
			s.respondError(c, http.StatusInternalServerError, "config_error", "API key não configurada no servidor")
			c.Abort()
			return
		}

		provided := extractAPIKey(c)
		if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(s.cfg.APIKey)) != 1 {
			s.respondError(c, http.StatusUnauthorized, "unauthorized", "API key inválida ou ausente")
			c.Abort()
			return
		}

		c.Next()
	}
}

// cors libera o front local em desenvolvimento. Em produção, a origem deve ser
// a mesma do deploy ou tratada por proxy.
func (s *Server) cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if isAllowedOrigin(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func isAllowedOrigin(origin string) bool {
	return strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:")
}

func extractAPIKey(c *gin.Context) string {
	if key := strings.TrimSpace(c.GetHeader("X-API-Key")); key != "" {
		return key
	}
	const prefix = "Bearer "
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(auth, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(auth, prefix))
	}
	return ""
}
