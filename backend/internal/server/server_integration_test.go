package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"encurtador/internal/clicks"
	"encurtador/internal/config"
	"encurtador/internal/database"
	"encurtador/internal/stats"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const integrationAPIKey = "test-api-key"

func TestHTTPContractsIntegration(t *testing.T) {
	handler, pool := setupIntegrationServer(t)
	ctx := context.Background()

	unauthorized := performJSON(t, handler, http.MethodGet, "/api/links", "", nil)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want %d", unauthorized.Code, http.StatusUnauthorized)
	}

	invalid := performJSON(t, handler, http.MethodPost, "/api/links", integrationAPIKey, map[string]any{
		"target_url": "http://localhost:3000",
		"slug":       "local",
	})
	assertErrorCode(t, invalid, http.StatusBadRequest, "invalid_url")

	create := performJSON(t, handler, http.MethodPost, "/api/links", integrationAPIKey, map[string]any{
		"target_url": "https://example.com/path",
		"slug":       "go",
		"title":      "Exemplo",
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body=%s", create.Code, http.StatusCreated, create.Body.String())
	}
	var created linkResponse
	decodeBody(t, create, &created)
	if created.Slug != "go" || created.ShortURL != "http://short.test/go" {
		t.Fatalf("created = %#v, want slug go and short URL", created)
	}

	duplicate := performJSON(t, handler, http.MethodPost, "/api/links", integrationAPIKey, map[string]any{
		"target_url": "https://example.org",
		"slug":       "go",
	})
	assertErrorCode(t, duplicate, http.StatusConflict, "slug_taken")

	list := performJSON(t, handler, http.MethodGet, "/api/links", integrationAPIKey, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d; body=%s", list.Code, http.StatusOK, list.Body.String())
	}
	var listed listLinksResponse
	decodeBody(t, list, &listed)
	if listed.Total != 1 || len(listed.Data) != 1 {
		t.Fatalf("listed total=%d len=%d, want one link", listed.Total, len(listed.Data))
	}

	start := time.Now()
	redirect := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/go", nil)
	req.RemoteAddr = "203.0.113.10:12345"
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://ref.example")
	handler.ServeHTTP(redirect, req)
	if redirect.Code != http.StatusFound {
		t.Fatalf("redirect status = %d, want %d; body=%s", redirect.Code, http.StatusFound, redirect.Body.String())
	}
	if got := redirect.Header().Get("Location"); got != "https://example.com/path" {
		t.Fatalf("Location = %q, want target", got)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("redirect demorou %s; captura nao deve bloquear o caminho critico", elapsed)
	}

	linkID := uuid.MustParse(created.ID)
	waitForClickCount(t, pool, linkID, 1)
	var ipHash string
	if err := pool.QueryRow(ctx, `SELECT ip_hash FROM clicks WHERE link_id = $1 LIMIT 1`, linkID).Scan(&ipHash); err != nil {
		t.Fatal(err)
	}
	if ipHash == "" || ipHash == "203.0.113.10" {
		t.Fatalf("ip_hash = %q, IP cru nao pode ser persistido", ipHash)
	}

	patch := performJSON(t, handler, http.MethodPatch, "/api/links/"+created.ID, integrationAPIKey, map[string]any{
		"is_active": false,
	})
	if patch.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want %d; body=%s", patch.Code, http.StatusOK, patch.Body.String())
	}

	gone := performJSON(t, handler, http.MethodGet, "/go", "", nil)
	assertErrorCode(t, gone, http.StatusGone, "link_unavailable")

	missing := performJSON(t, handler, http.MethodGet, "/missing", "", nil)
	assertErrorCode(t, missing, http.StatusNotFound, "not_found")

	remove := performJSON(t, handler, http.MethodDelete, "/api/links/"+created.ID, integrationAPIKey, nil)
	if remove.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d; body=%s", remove.Code, http.StatusNoContent, remove.Body.String())
	}
}

func TestStatsIntegration(t *testing.T) {
	handler, pool := setupIntegrationServer(t)
	ctx := context.Background()
	linkID := uuid.MustParse("018ff7d0-9c7a-7a9a-a85f-0d82b8c08c1d")
	baseDay := truncateTestDay(time.Now().UTC())

	if _, err := pool.Exec(ctx, `
INSERT INTO links (id, slug, target_url, title, is_active)
VALUES ($1, 'stats', 'https://example.com/stats', 'Stats', true)
`, linkID); err != nil {
		t.Fatal(err)
	}

	seedStatsClick(t, pool, linkID, baseDay.AddDate(0, 0, -35).Add(9*time.Hour), stringPtr("https://old.example"), "bot", "BR")
	seedStatsClick(t, pool, linkID, baseDay.AddDate(0, 0, -8).Add(10*time.Hour), stringPtr("https://docs.example"), "mobile", "US")
	seedStatsClick(t, pool, linkID, baseDay.AddDate(0, 0, -6).Add(11*time.Hour), stringPtr("https://google.com"), "desktop", "BR")
	seedStatsClick(t, pool, linkID, baseDay.AddDate(0, 0, -1).Add(12*time.Hour), nil, "", "")
	seedStatsClick(t, pool, linkID, baseDay.Add(13*time.Hour), stringPtr("https://google.com"), "desktop", "BR")

	tests := []struct {
		rangeValue string
		wantTotal  int64
		wantStart  time.Time
	}{
		{rangeValue: "7d", wantTotal: 3, wantStart: baseDay.AddDate(0, 0, -6)},
		{rangeValue: "30d", wantTotal: 4, wantStart: baseDay.AddDate(0, 0, -29)},
		{rangeValue: "all", wantTotal: 5, wantStart: baseDay.AddDate(0, 0, -35)},
	}

	for _, tt := range tests {
		t.Run(tt.rangeValue, func(t *testing.T) {
			rec := performJSON(t, handler, http.MethodGet, "/api/links/"+linkID.String()+"/stats?range="+tt.rangeValue, integrationAPIKey, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("stats status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
			}
			var got statsResponse
			decodeBody(t, rec, &got)
			if got.TotalClicks != tt.wantTotal {
				t.Fatalf("TotalClicks = %d, want %d", got.TotalClicks, tt.wantTotal)
			}
			if got.StartDay != tt.wantStart.Format(time.DateOnly) {
				t.Fatalf("StartDay = %s, want %s", got.StartDay, tt.wantStart.Format(time.DateOnly))
			}
			var dailyTotal int64
			for _, point := range got.Daily {
				dailyTotal += point.Clicks
			}
			if dailyTotal != got.TotalClicks {
				t.Fatalf("daily total = %d, total_clicks = %d", dailyTotal, got.TotalClicks)
			}
		})
	}

	rec := performJSON(t, handler, http.MethodGet, "/api/links/"+linkID.String()+"/stats?range=7d", integrationAPIKey, nil)
	var seven statsResponse
	decodeBody(t, rec, &seven)
	assertBreakdownContains(t, seven.Devices, "unknown", 1)
	assertBreakdownContains(t, seven.Countries, "unknown", 1)
	if len(seven.TopReferrers) == 0 || seven.TopReferrers[0].Referrer == nil || *seven.TopReferrers[0].Referrer != "https://google.com" || seven.TopReferrers[0].Clicks != 2 {
		t.Fatalf("top referrer = %#v, want google with 2 clicks", seven.TopReferrers)
	}

	invalid := performJSON(t, handler, http.MethodGet, "/api/links/"+linkID.String()+"/stats?range=90d", integrationAPIKey, nil)
	assertErrorCode(t, invalid, http.StatusBadRequest, "invalid_range")
}

func setupIntegrationServer(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL nao configurado; pulando integracao Postgres")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := database.Migrate(databaseURL, logger); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	ctx := context.Background()
	pool, err := database.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `TRUNCATE link_daily, clicks, links RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	cfg := &config.Config{
		AppEnv:     "test",
		HTTPAddr:   ":0",
		BaseURL:    "http://short.test",
		APIKey:     integrationAPIKey,
		IPHashSalt: "integration-salt",
	}
	clickService := clicks.NewService(pool, nil, clicks.NewRawIPCache(time.Minute), cfg.IPHashSalt, logger)
	return New(cfg, pool, logger, clickService, stats.NewRepository(pool)).Handler(), pool
}

func performJSON(t *testing.T, handler http.Handler, method, path, apiKey string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(payload)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder, out any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
}

func assertErrorCode(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, wantStatus, rec.Body.String())
	}
	var body errorBody
	decodeBody(t, rec, &body)
	if body.Error.Code != wantCode {
		t.Fatalf("error code = %q, want %q; body=%s", body.Error.Code, wantCode, rec.Body.String())
	}
	if body.Error.Message == "" {
		t.Fatalf("mensagem de erro vazia para code %q", wantCode)
	}
}

func waitForClickCount(t *testing.T, pool *pgxpool.Pool, linkID uuid.UUID, want int64) {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(2 * time.Second)
	for {
		var got int64
		if err := pool.QueryRow(ctx, `SELECT count(*)::bigint FROM clicks WHERE link_id = $1`, linkID).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got >= want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("click count nao chegou a %d", want)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func seedStatsClick(t *testing.T, pool *pgxpool.Pool, linkID uuid.UUID, createdAt time.Time, referrer *string, deviceType, country string) {
	t.Helper()
	ctx := context.Background()
	clickID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO clicks (id, link_id, created_at, ip_hash, referrer, ua_raw, device_type, country, enriched_at)
VALUES ($1, $2, $3, $4, $5, 'ua', $6, $7, now())
`, clickID, linkID, createdAt, "hash-"+strconv.FormatInt(createdAt.UnixNano(), 10), referrer, emptyStringToNil(deviceType), emptyStringToNil(country)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO link_daily (link_id, day, clicks)
VALUES ($1, ($2 AT TIME ZONE 'UTC')::date, 1)
ON CONFLICT (link_id, day)
DO UPDATE SET clicks = link_daily.clicks + 1
`, linkID, createdAt); err != nil {
		t.Fatal(err)
	}
}

func assertBreakdownContains(t *testing.T, points []breakdownResponse, key string, clicks int64) {
	t.Helper()
	for _, point := range points {
		if point.Key == key && point.Clicks == clicks {
			return
		}
	}
	t.Fatalf("breakdown %#v nao contem %q com %d cliques", points, key, clicks)
}

func truncateTestDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func stringPtr(value string) *string {
	return &value
}

func emptyStringToNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
