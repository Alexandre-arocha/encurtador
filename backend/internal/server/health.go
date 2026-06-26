package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleHealthz responde o estado do serviço e a conectividade com o banco.
func (s *Server) handleHealthz(c *gin.Context) {
	if err := s.pool.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "db": "down"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "db": "up"})
}
