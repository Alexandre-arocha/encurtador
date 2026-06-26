package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"encurtador/internal/config"

	"github.com/gin-gonic/gin"
)

func TestAPIKeyAuth(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
		header     string
		wantStatus int
		wantCode   string
	}{
		{name: "missing", wantStatus: http.StatusUnauthorized, wantCode: "unauthorized"},
		{name: "wrong", headerName: "X-API-Key", header: "wrong", wantStatus: http.StatusUnauthorized, wantCode: "unauthorized"},
		{name: "x api key", headerName: "X-API-Key", header: "test-key", wantStatus: http.StatusOK},
		{name: "bearer", headerName: "Authorization", header: "Bearer test-key", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := newMiddlewareTestEngine("test-key")
			req := httptest.NewRequest(http.MethodGet, "/api/probe", nil)
			if tt.headerName != "" {
				req.Header.Set(tt.headerName, tt.header)
			}
			rec := httptest.NewRecorder()

			engine.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantCode != "" {
				var body errorBody
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Error.Code != tt.wantCode {
					t.Fatalf("code = %q, want %q", body.Error.Code, tt.wantCode)
				}
			}
		})
	}
}

func TestCORSPreflightAllowsLocalOriginWithoutAPIKey(t *testing.T) {
	engine := newMiddlewareTestEngine("test-key")
	req := httptest.NewRequest(http.MethodOptions, "/api/probe", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want local origin", got)
	}
}

func newMiddlewareTestEngine(apiKey string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	s := &Server{cfg: &config.Config{APIKey: apiKey}}
	engine := gin.New()
	engine.Use(s.cors())
	api := engine.Group("/api", s.apiKeyAuth())
	api.GET("/probe", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return engine
}
