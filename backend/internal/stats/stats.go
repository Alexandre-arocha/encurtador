// Package stats concentra o contrato de agregacoes usado pelo dashboard.
package stats

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	Range7D  Range = "7d"
	Range30D Range = "30d"
	RangeAll Range = "all"

	DefaultTopN = 10
	Unknown     = "unknown"
)

// Range representa o filtro aceito por GET /api/links/:id/stats.
type Range string

// ParseRange valida o parametro range.
func ParseRange(raw string) (Range, error) {
	switch Range(strings.TrimSpace(strings.ToLower(raw))) {
	case "", Range7D:
		return Range7D, nil
	case Range30D:
		return Range30D, nil
	case RangeAll:
		return RangeAll, nil
	default:
		return "", fmt.Errorf("range invalido: %q", raw)
	}
}

// Window descreve o intervalo fechado em dias e semiaberto em timestamps.
type Window struct {
	Range        Range
	StartDay     time.Time
	EndDay       time.Time
	Start        time.Time
	EndExclusive time.Time
}

// ResolveWindow calcula o intervalo em UTC. Para all, firstDay nil produz o dia
// atual com zero cliques.
func ResolveWindow(r Range, now time.Time, firstDay *time.Time) (Window, error) {
	if r == "" {
		r = Range7D
	}
	endDay := truncateDayUTC(now)
	startDay := endDay

	switch r {
	case Range7D:
		startDay = endDay.AddDate(0, 0, -6)
	case Range30D:
		startDay = endDay.AddDate(0, 0, -29)
	case RangeAll:
		if firstDay != nil {
			startDay = truncateDayUTC(*firstDay)
			if startDay.After(endDay) {
				startDay = endDay
			}
		}
	default:
		return Window{}, fmt.Errorf("range invalido: %q", r)
	}

	return Window{
		Range:        r,
		StartDay:     startDay,
		EndDay:       endDay,
		Start:        startDay,
		EndExclusive: endDay.AddDate(0, 0, 1),
	}, nil
}

// DailyPoint representa um ponto da serie diaria.
type DailyPoint struct {
	Day    time.Time `json:"day"`
	Clicks int64     `json:"clicks"`
}

// ReferrerPoint representa cliques agrupados por referrer.
type ReferrerPoint struct {
	Referrer *string `json:"referrer"`
	Clicks   int64   `json:"clicks"`
}

// BreakdownPoint representa cliques agrupados por device/country.
type BreakdownPoint struct {
	Key    string `json:"key"`
	Clicks int64  `json:"clicks"`
}

// Result e a resposta agregada esperada pelo endpoint de stats.
type Result struct {
	Range        Range            `json:"range"`
	StartDay     time.Time        `json:"start_day"`
	EndDay       time.Time        `json:"end_day"`
	TotalClicks  int64            `json:"total_clicks"`
	Daily        []DailyPoint     `json:"daily"`
	TopReferrers []ReferrerPoint  `json:"top_referrers"`
	Devices      []BreakdownPoint `json:"devices"`
	Countries    []BreakdownPoint `json:"countries"`
}

// FixtureClick permite validar as regras de agregacao sem subir Postgres.
type FixtureClick struct {
	CreatedAt  time.Time
	Referrer   *string
	DeviceType string
	Country    string
}

// BuildFromFixtures agrega fixtures deterministicas de clicks e link_daily.
func BuildFromFixtures(window Window, daily []DailyPoint, clicks []FixtureClick, topN int) Result {
	if topN <= 0 {
		topN = DefaultTopN
	}

	result := Result{
		Range:        window.Range,
		StartDay:     window.StartDay,
		EndDay:       window.EndDay,
		Daily:        normalizeDaily(window, daily),
		TopReferrers: topReferrersFromClicks(window, clicks, topN),
		Devices:      breakdownFromClicks(window, clicks, func(c FixtureClick) string { return c.DeviceType }),
		Countries:    breakdownFromClicks(window, clicks, func(c FixtureClick) string { return c.Country }),
	}
	for _, click := range clicks {
		if inWindow(click.CreatedAt, window) {
			result.TotalClicks++
		}
	}
	return result
}

func normalizeDaily(window Window, rows []DailyPoint) []DailyPoint {
	byDay := make(map[time.Time]int64, len(rows))
	for _, row := range rows {
		byDay[truncateDayUTC(row.Day)] += row.Clicks
	}

	totalDays := int(window.EndDay.Sub(window.StartDay).Hours()/24) + 1
	if totalDays < 1 {
		totalDays = 1
	}
	out := make([]DailyPoint, 0, totalDays)
	for day := window.StartDay; !day.After(window.EndDay); day = day.AddDate(0, 0, 1) {
		out = append(out, DailyPoint{Day: day, Clicks: byDay[day]})
	}
	return out
}

func topReferrersFromClicks(window Window, clicks []FixtureClick, topN int) []ReferrerPoint {
	counts := map[string]int64{}
	labels := map[string]*string{}
	for _, click := range clicks {
		if !inWindow(click.CreatedAt, window) {
			continue
		}
		key := ""
		if click.Referrer != nil {
			key = *click.Referrer
			labels[key] = click.Referrer
		}
		counts[key]++
	}

	keys := sortedCountKeys(counts)
	if len(keys) > topN {
		keys = keys[:topN]
	}
	out := make([]ReferrerPoint, 0, len(keys))
	for _, key := range keys {
		out = append(out, ReferrerPoint{Referrer: labels[key], Clicks: counts[key]})
	}
	return out
}

func breakdownFromClicks(window Window, clicks []FixtureClick, keyFn func(FixtureClick) string) []BreakdownPoint {
	counts := map[string]int64{}
	for _, click := range clicks {
		if !inWindow(click.CreatedAt, window) {
			continue
		}
		key := strings.TrimSpace(keyFn(click))
		if key == "" {
			key = Unknown
		}
		counts[key]++
	}

	keys := sortedCountKeys(counts)
	out := make([]BreakdownPoint, 0, len(keys))
	for _, key := range keys {
		out = append(out, BreakdownPoint{Key: key, Clicks: counts[key]})
	}
	return out
}

func sortedCountKeys(counts map[string]int64) []string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]] == counts[keys[j]] {
			return keys[i] < keys[j]
		}
		return counts[keys[i]] > counts[keys[j]]
	})
	return keys
}

func inWindow(ts time.Time, window Window) bool {
	ts = ts.UTC()
	return !ts.Before(window.Start) && ts.Before(window.EndExclusive)
}

func truncateDayUTC(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// Queryer e o subconjunto de pgxpool.Pool/pgx.Tx usado pelo repositorio.
type Queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Repository executa as queries de agregacao em Postgres.
type Repository struct {
	db Queryer
}

// NewRepository cria um repositorio de stats.
func NewRepository(db Queryer) *Repository {
	return &Repository{db: db}
}

// Get monta todas as agregacoes para um link no range informado.
func (r *Repository) Get(ctx context.Context, linkID uuid.UUID, rangeValue Range, now time.Time, topN int) (Result, error) {
	if r == nil || r.db == nil {
		return Result{}, errors.New("stats repository sem db")
	}
	if topN <= 0 {
		topN = DefaultTopN
	}

	firstDay, err := r.firstStatsDay(ctx, linkID)
	if err != nil {
		return Result{}, err
	}
	window, err := ResolveWindow(rangeValue, now, firstDay)
	if err != nil {
		return Result{}, err
	}

	daily, err := r.daily(ctx, linkID, window)
	if err != nil {
		return Result{}, err
	}
	topReferrers, err := r.topReferrers(ctx, linkID, window, topN)
	if err != nil {
		return Result{}, err
	}
	devices, err := r.breakdown(ctx, deviceBreakdownSQL, linkID, window)
	if err != nil {
		return Result{}, err
	}
	countries, err := r.breakdown(ctx, countryBreakdownSQL, linkID, window)
	if err != nil {
		return Result{}, err
	}
	totalClicks, err := r.totalClicks(ctx, linkID, window)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Range:        window.Range,
		StartDay:     window.StartDay,
		EndDay:       window.EndDay,
		TotalClicks:  totalClicks,
		Daily:        daily,
		TopReferrers: topReferrers,
		Devices:      devices,
		Countries:    countries,
	}, nil
}

func (r *Repository) firstStatsDay(ctx context.Context, linkID uuid.UUID) (*time.Time, error) {
	var day pgtype.Date
	if err := r.db.QueryRow(ctx, firstStatsDaySQL, linkID).Scan(&day); err != nil {
		return nil, err
	}
	if !day.Valid {
		return nil, nil
	}
	t := truncateDayUTC(day.Time)
	return &t, nil
}

func (r *Repository) daily(ctx context.Context, linkID uuid.UUID, window Window) ([]DailyPoint, error) {
	rows, err := r.db.Query(ctx, dailySeriesSQL, linkID, window.StartDay, window.EndDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []DailyPoint
	for rows.Next() {
		var day pgtype.Date
		var clicks int64
		if err := rows.Scan(&day, &clicks); err != nil {
			return nil, err
		}
		points = append(points, DailyPoint{Day: truncateDayUTC(day.Time), Clicks: clicks})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizeDaily(window, points), nil
}

func (r *Repository) topReferrers(ctx context.Context, linkID uuid.UUID, window Window, topN int) ([]ReferrerPoint, error) {
	rows, err := r.db.Query(ctx, topReferrersSQL, linkID, window.Start, window.EndExclusive, topN)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ReferrerPoint
	for rows.Next() {
		var ref pgtype.Text
		var clicks int64
		if err := rows.Scan(&ref, &clicks); err != nil {
			return nil, err
		}
		var referrer *string
		if ref.Valid {
			value := ref.String
			referrer = &value
		}
		out = append(out, ReferrerPoint{Referrer: referrer, Clicks: clicks})
	}
	return out, rows.Err()
}

func (r *Repository) breakdown(ctx context.Context, query string, linkID uuid.UUID, window Window) ([]BreakdownPoint, error) {
	rows, err := r.db.Query(ctx, query, linkID, window.Start, window.EndExclusive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []BreakdownPoint
	for rows.Next() {
		var item BreakdownPoint
		if err := rows.Scan(&item.Key, &item.Clicks); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Repository) totalClicks(ctx context.Context, linkID uuid.UUID, window Window) (int64, error) {
	var total int64
	err := r.db.QueryRow(ctx, totalClicksSQL, linkID, window.Start, window.EndExclusive).Scan(&total)
	return total, err
}

const firstStatsDaySQL = `
SELECT min(day)::date
FROM (
    SELECT min(day)::date AS day
    FROM link_daily
    WHERE link_id = $1
    UNION ALL
    SELECT min((created_at AT TIME ZONE 'UTC')::date)::date AS day
    FROM clicks
    WHERE link_id = $1
) bounds;
`

const dailySeriesSQL = `
WITH days AS (
    SELECT generate_series($2::date, $3::date, interval '1 day')::date AS day
)
SELECT days.day, COALESCE(link_daily.clicks, 0)::bigint AS clicks
FROM days
LEFT JOIN link_daily
    ON link_daily.link_id = $1
   AND link_daily.day = days.day
ORDER BY days.day;
`

const topReferrersSQL = `
SELECT referrer, count(*)::bigint AS clicks
FROM clicks
WHERE link_id = $1
  AND created_at >= $2
  AND created_at < $3
GROUP BY referrer
ORDER BY clicks DESC, referrer ASC NULLS LAST
LIMIT $4;
`

const deviceBreakdownSQL = `
SELECT COALESCE(NULLIF(device_type, ''), 'unknown') AS key, count(*)::bigint AS clicks
FROM clicks
WHERE link_id = $1
  AND created_at >= $2
  AND created_at < $3
GROUP BY key
ORDER BY clicks DESC, key ASC;
`

const countryBreakdownSQL = `
SELECT COALESCE(NULLIF(country, ''), 'unknown') AS key, count(*)::bigint AS clicks
FROM clicks
WHERE link_id = $1
  AND created_at >= $2
  AND created_at < $3
GROUP BY key
ORDER BY clicks DESC, key ASC;
`

const totalClicksSQL = `
SELECT count(*)::bigint
FROM clicks
WHERE link_id = $1
  AND created_at >= $2
  AND created_at < $3;
`
