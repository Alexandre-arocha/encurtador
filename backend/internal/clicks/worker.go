package clicks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"encurtador/internal/enrich"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

// EnrichClickArgs identifica o job River de enriquecimento. Nao inclui IP cru:
// o unico dado persistido no job e o id do click.
type EnrichClickArgs struct {
	ClickID uuid.UUID `json:"click_id"`
}

func (EnrichClickArgs) Kind() string { return "enrich_click" }

// EnrichWorker processa clicks crus, preenche os campos derivados e incrementa
// o rollup diario de forma idempotente.
type EnrichWorker struct {
	river.WorkerDefaults[EnrichClickArgs]

	pool     *pgxpool.Pool
	enricher *enrich.Service
	rawIPs   *RawIPCache
	logger   *slog.Logger
}

// NewEnrichWorker cria o worker River de enriquecimento.
func NewEnrichWorker(pool *pgxpool.Pool, enricher *enrich.Service, rawIPs *RawIPCache, logger *slog.Logger) *EnrichWorker {
	return &EnrichWorker{
		pool:     pool,
		enricher: enricher,
		rawIPs:   rawIPs,
		logger:   logger,
	}
}

func (w *EnrichWorker) Work(ctx context.Context, job *river.Job[EnrichClickArgs]) error {
	if w == nil || w.pool == nil {
		return errors.New("enrich worker sem pool")
	}

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()

	row, err := fetchClickForEnrich(ctx, tx, job.Args.ClickID)
	if errors.Is(err, pgx.ErrNoRows) {
		w.rawIPs.Delete(job.Args.ClickID)
		return nil
	}
	if err != nil {
		return err
	}
	if row.Enriched {
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		committed = true
		w.rawIPs.Delete(job.Args.ClickID)
		return nil
	}

	result := w.enrichRawClick(job.Args.ClickID, row.UARaw)

	updated, err := updateClickEnriched(ctx, tx, job.Args.ClickID, result)
	if err != nil {
		return err
	}
	if updated {
		if _, err := tx.Exec(ctx, upsertLinkDailySQL, row.LinkID, row.CreatedAt); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	w.rawIPs.Delete(job.Args.ClickID)
	return nil
}

type clickForEnrich struct {
	LinkID    uuid.UUID
	CreatedAt time.Time
	UARaw     string
	Enriched  bool
}

func fetchClickForEnrich(ctx context.Context, tx pgx.Tx, clickID uuid.UUID) (clickForEnrich, error) {
	var row clickForEnrich
	err := tx.QueryRow(ctx, fetchClickForEnrichSQL, clickID).Scan(
		&row.LinkID,
		&row.CreatedAt,
		&row.UARaw,
		&row.Enriched,
	)
	return row, err
}

func updateClickEnriched(ctx context.Context, tx pgx.Tx, clickID uuid.UUID, result enrich.Enriched) (bool, error) {
	tag, err := tx.Exec(ctx, updateClickEnrichedSQL,
		clickID,
		emptyToNil(result.DeviceType),
		emptyToNil(result.Browser),
		emptyToNil(result.OS),
		emptyToNil(result.Country),
		emptyToNil(result.City),
	)
	if err != nil {
		return false, fmt.Errorf("erro ao atualizar enrich do click: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (w *EnrichWorker) enrichRawClick(clickID uuid.UUID, uaRaw string) enrich.Enriched {
	if w == nil || w.enricher == nil {
		return enrich.Enriched{}
	}
	rawIP := ""
	if w.rawIPs != nil {
		rawIP, _ = w.rawIPs.Get(clickID)
	}
	return w.enricher.Enrich(enrich.Raw{UARaw: uaRaw, IP: rawIP})
}

func emptyToNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

const fetchClickForEnrichSQL = `
SELECT link_id, created_at, ua_raw, enriched_at IS NOT NULL AS enriched
FROM clicks
WHERE id = $1
FOR UPDATE;
`

const updateClickEnrichedSQL = `
UPDATE clicks
SET device_type = $2,
    browser = $3,
    os = $4,
    country = $5,
    city = $6,
    enriched_at = now()
WHERE id = $1
  AND enriched_at IS NULL;
`

const upsertLinkDailySQL = `
INSERT INTO link_daily (link_id, day, clicks)
VALUES ($1, ($2 AT TIME ZONE 'UTC')::date, 1)
ON CONFLICT (link_id, day)
DO UPDATE SET clicks = link_daily.clicks + 1;
`
