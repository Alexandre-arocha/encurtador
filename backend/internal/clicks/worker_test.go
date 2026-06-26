package clicks

import (
	"encoding/json"
	"net/netip"
	"strings"
	"testing"
	"time"

	"encurtador/internal/enrich"

	"github.com/google/uuid"
)

func TestLinkDailyRollupUsesUTCDay(t *testing.T) {
	if !strings.Contains(upsertLinkDailySQL, "AT TIME ZONE 'UTC'") {
		t.Fatal("link_daily rollup deve converter created_at para dia UTC")
	}
}

func TestWorkerSQLKeepsRollupIdempotent(t *testing.T) {
	if !strings.Contains(updateClickEnrichedSQL, "AND enriched_at IS NULL") {
		t.Fatal("update do enrich deve ignorar clicks ja enriquecidos")
	}
	if !strings.Contains(upsertLinkDailySQL, "ON CONFLICT (link_id, day)") {
		t.Fatal("rollup diario deve usar upsert por link e dia")
	}
}

func TestEnrichClickArgsDoesNotSerializeRawIP(t *testing.T) {
	clickID := uuid.MustParse("018ff7d0-9c7a-7a9a-a85f-0d82b8c08c1d")
	payload, err := json.Marshal(EnrichClickArgs{ClickID: clickID})
	if err != nil {
		t.Fatal(err)
	}

	text := string(payload)
	if !strings.Contains(text, "click_id") {
		t.Fatalf("payload = %s, want click_id", text)
	}
	if strings.Contains(strings.ToLower(text), "ip") {
		t.Fatalf("payload do job nao deve conter IP cru: %s", text)
	}
}

func TestWorkerEnrichesWithVolatileRawIPCache(t *testing.T) {
	clickID := uuid.MustParse("018ff7d0-9c7a-7a9a-a85f-0d82b8c08c1d")
	cache := NewRawIPCache(time.Minute)
	cache.Put(clickID, "8.8.8.8")

	worker := &EnrichWorker{
		enricher: enrich.New(fakeGeo{country: "us", city: "Mountain View"}),
		rawIPs:   cache,
	}
	got := worker.enrichRawClick(clickID, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")

	if got.Country != "US" || got.City != "Mountain View" {
		t.Fatalf("geo = (%q, %q), want (US, Mountain View)", got.Country, got.City)
	}
	if got.DeviceType != enrich.DeviceDesktop {
		t.Fatalf("DeviceType = %q, want %q", got.DeviceType, enrich.DeviceDesktop)
	}
}

func TestWorkerEnrichesWithoutRawIPWhenCacheMisses(t *testing.T) {
	clickID := uuid.MustParse("018ff7d0-9c7a-7a9a-a85f-0d82b8c08c1d")
	geo := &countingGeo{}
	worker := &EnrichWorker{
		enricher: enrich.New(geo),
		rawIPs:   NewRawIPCache(time.Nanosecond),
	}

	got := worker.enrichRawClick(clickID, "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")

	if got.DeviceType != enrich.DeviceBot {
		t.Fatalf("DeviceType = %q, want %q", got.DeviceType, enrich.DeviceBot)
	}
	if geo.calls != 0 {
		t.Fatalf("geo calls = %d, want 0", geo.calls)
	}
}

type fakeGeo struct {
	country string
	city    string
}

func (f fakeGeo) LookupCity(ip netip.Addr) (enrich.GeoResult, bool) {
	return enrich.GeoResult{Country: f.country, City: f.city}, true
}

type countingGeo struct {
	calls int
}

func (g *countingGeo) LookupCity(ip netip.Addr) (enrich.GeoResult, bool) {
	g.calls++
	return enrich.GeoResult{}, false
}
