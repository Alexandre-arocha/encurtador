package stats

import (
	"strings"
	"testing"
	"time"
)

func TestResolveWindowRanges(t *testing.T) {
	now := time.Date(2026, 6, 25, 15, 20, 0, 0, time.FixedZone("BRT", -3*60*60))

	w7, err := ResolveWindow(Range7D, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := w7.StartDay.Format(time.DateOnly), "2026-06-19"; got != want {
		t.Fatalf("7d start = %s, want %s", got, want)
	}
	if got, want := w7.EndDay.Format(time.DateOnly), "2026-06-25"; got != want {
		t.Fatalf("7d end = %s, want %s", got, want)
	}

	w30, err := ResolveWindow(Range30D, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := w30.StartDay.Format(time.DateOnly), "2026-05-27"; got != want {
		t.Fatalf("30d start = %s, want %s", got, want)
	}

	first := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	wAll, err := ResolveWindow(RangeAll, now, &first)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := wAll.StartDay.Format(time.DateOnly), "2026-06-02"; got != want {
		t.Fatalf("all start = %s, want %s", got, want)
	}
}

func TestBuildFromFixturesAggregatesAndPreservesDailyGaps(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	first := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)
	window, err := ResolveWindow(RangeAll, now, &first)
	if err != nil {
		t.Fatal(err)
	}

	refGoogle := "https://google.com"
	refDocs := "https://docs.example"
	clicks := []FixtureClick{
		{CreatedAt: time.Date(2026, 6, 23, 9, 0, 0, 0, time.UTC), Referrer: &refGoogle, DeviceType: "desktop", Browser: "Chrome", OS: "Windows", Country: "BR", City: "Sao Paulo"},
		{CreatedAt: time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC), Referrer: &refGoogle, DeviceType: "mobile", Browser: "Safari", OS: "iOS", Country: "BR", City: "Rio"},
		{CreatedAt: time.Date(2026, 6, 24, 11, 0, 0, 0, time.UTC), Referrer: &refDocs, DeviceType: "desktop", Browser: "Chrome", OS: "Windows", Country: "US", City: "Austin"},
		{CreatedAt: time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC), DeviceType: "", Browser: "", OS: "", Country: "", City: ""},
		{CreatedAt: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC), Referrer: &refGoogle, DeviceType: "bot", Browser: "Googlebot", OS: "unknown", Country: "BR", City: "Sao Paulo"},
	}
	daily := []DailyPoint{
		{Day: time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC), Clicks: 2},
		{Day: time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC), Clicks: 2},
	}

	got := BuildFromFixtures(window, daily, clicks, 2)

	if got.TotalClicks != 4 {
		t.Fatalf("TotalClicks = %d, want 4", got.TotalClicks)
	}
	if len(got.Daily) != 3 {
		t.Fatalf("daily len = %d, want 3", len(got.Daily))
	}
	if got.Daily[0].Clicks != 2 || got.Daily[1].Clicks != 2 || got.Daily[2].Clicks != 0 {
		t.Fatalf("daily clicks = [%d %d %d], want [2 2 0]", got.Daily[0].Clicks, got.Daily[1].Clicks, got.Daily[2].Clicks)
	}
	if got.TopReferrers[0].Referrer == nil || *got.TopReferrers[0].Referrer != refGoogle || got.TopReferrers[0].Clicks != 2 {
		t.Fatalf("top referrer[0] = %#v, want google with 2", got.TopReferrers[0])
	}
	if got.Devices[0] != (BreakdownPoint{Key: "desktop", Clicks: 2}) {
		t.Fatalf("devices[0] = %#v, want desktop 2", got.Devices[0])
	}
	if got.Countries[0] != (BreakdownPoint{Key: "BR", Clicks: 2}) {
		t.Fatalf("countries[0] = %#v, want BR 2", got.Countries[0])
	}
	if got.Browsers[0] != (BreakdownPoint{Key: "Chrome", Clicks: 2}) {
		t.Fatalf("browsers[0] = %#v, want Chrome 2", got.Browsers[0])
	}
	if got.OperatingSystems[0] != (BreakdownPoint{Key: "Windows", Clicks: 2}) {
		t.Fatalf("operating systems[0] = %#v, want Windows 2", got.OperatingSystems[0])
	}
	if got.Cities[0] != (BreakdownPoint{Key: "Austin", Clicks: 1}) {
		t.Fatalf("cities[0] = %#v, want Austin 1 por desempate alfabetico", got.Cities[0])
	}
}

func TestDailyRollupMatchesClickCountFixture(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	window, err := ResolveWindow(Range7D, now, nil)
	if err != nil {
		t.Fatal(err)
	}

	clicks := []FixtureClick{
		{CreatedAt: time.Date(2026, 6, 19, 1, 0, 0, 0, time.UTC)},
		{CreatedAt: time.Date(2026, 6, 20, 1, 0, 0, 0, time.UTC)},
		{CreatedAt: time.Date(2026, 6, 20, 2, 0, 0, 0, time.UTC)},
		{CreatedAt: time.Date(2026, 6, 25, 23, 59, 59, 0, time.UTC)},
	}
	daily := []DailyPoint{
		{Day: time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC), Clicks: 1},
		{Day: time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC), Clicks: 2},
		{Day: time.Date(2026, 6, 25, 0, 0, 0, 0, time.UTC), Clicks: 1},
	}

	got := BuildFromFixtures(window, daily, clicks, DefaultTopN)
	var rollupTotal int64
	for _, point := range got.Daily {
		rollupTotal += point.Clicks
	}
	if rollupTotal != got.TotalClicks {
		t.Fatalf("rollupTotal = %d, TotalClicks = %d", rollupTotal, got.TotalClicks)
	}
}

func TestBuildFromFixturesUsesUTCBoundaries(t *testing.T) {
	now := time.Date(2026, 6, 25, 23, 30, 0, 0, time.UTC)
	window, err := ResolveWindow(Range7D, now, nil)
	if err != nil {
		t.Fatal(err)
	}

	plusTwo := time.FixedZone("UTC+2", 2*60*60)
	minusThree := time.FixedZone("UTC-3", -3*60*60)
	clicks := []FixtureClick{
		// 2026-06-25 22:30 UTC, inside the window's last UTC day.
		{CreatedAt: time.Date(2026, 6, 26, 0, 30, 0, 0, plusTwo)},
		// 2026-06-26 01:30 UTC, outside the half-open UTC window.
		{CreatedAt: time.Date(2026, 6, 25, 22, 30, 0, 0, minusThree)},
	}
	daily := []DailyPoint{
		{Day: time.Date(2026, 6, 25, 0, 0, 0, 0, time.UTC), Clicks: 1},
	}

	got := BuildFromFixtures(window, daily, clicks, DefaultTopN)
	if got.TotalClicks != 1 {
		t.Fatalf("TotalClicks = %d, want 1", got.TotalClicks)
	}
	var rollupTotal int64
	for _, point := range got.Daily {
		rollupTotal += point.Clicks
	}
	if rollupTotal != got.TotalClicks {
		t.Fatalf("rollupTotal = %d, TotalClicks = %d", rollupTotal, got.TotalClicks)
	}
}

func TestFirstStatsDayQueryUsesUTCDay(t *testing.T) {
	if !strings.Contains(firstStatsDaySQL, "created_at AT TIME ZONE 'UTC'") {
		t.Fatal("first stats day deve converter created_at para dia UTC")
	}
}
