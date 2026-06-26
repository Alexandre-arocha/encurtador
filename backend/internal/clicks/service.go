// Package clicks cuida da captura crua e do enriquecimento assíncrono dos
// cliques.
package clicks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

const captureTimeout = 3 * time.Second

// Service grava clicks e enfileira o job de enrich fora do caminho critico do
// redirect.
type Service struct {
	pool        *pgxpool.Pool
	riverClient *river.Client[pgx.Tx]
	rawIPs      *RawIPCache
	ipHashSalt  string
	logger      *slog.Logger
}

// NewService cria o servico de captura de clicks.
func NewService(pool *pgxpool.Pool, riverClient *river.Client[pgx.Tx], rawIPs *RawIPCache, ipHashSalt string, logger *slog.Logger) *Service {
	return &Service{
		pool:        pool,
		riverClient: riverClient,
		rawIPs:      rawIPs,
		ipHashSalt:  ipHashSalt,
		logger:      logger,
	}
}

// CaptureInput contem os campos coletados pelo handler de redirect.
type CaptureInput struct {
	LinkID   uuid.UUID
	IP       string
	Referrer string
	UARaw    string
}

// CaptureAsync agenda a captura em background e retorna imediatamente.
func (s *Service) CaptureAsync(in CaptureInput) {
	if s == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), captureTimeout)
		defer cancel()

		if _, err := s.RecordAndEnqueue(ctx, in); err != nil && s.logger != nil {
			s.logger.Warn("falha ao capturar clique", "link_id", in.LinkID.String(), "err", err)
		}
	}()
}

// RecordAndEnqueue persiste o click e insere um job River na mesma transacao.
func (s *Service) RecordAndEnqueue(ctx context.Context, in CaptureInput) (uuid.UUID, error) {
	if s == nil || s.pool == nil {
		return uuid.Nil, errors.New("click service sem pool")
	}
	if s.ipHashSalt == "" {
		return uuid.Nil, errors.New("IP_HASH_SALT nao configurado")
	}

	clickID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("erro ao gerar id do clique: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
			s.rawIPs.Delete(clickID)
		}
	}()

	referrer := nullableTrimmed(in.Referrer)
	uaRaw := strings.TrimSpace(in.UARaw)
	ipHash := HashIP(in.IP, s.ipHashSalt)

	if _, err := tx.Exec(ctx, createClickSQL, clickID, in.LinkID, ipHash, referrer, uaRaw); err != nil {
		return uuid.Nil, err
	}

	s.rawIPs.Put(clickID, in.IP)
	if s.riverClient != nil {
		if _, err := s.riverClient.InsertTx(ctx, tx, EnrichClickArgs{ClickID: clickID}, &river.InsertOpts{
			MaxAttempts: 5,
			Queue:       river.QueueDefault,
		}); err != nil {
			return uuid.Nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	committed = true
	return clickID, nil
}

func nullableTrimmed(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

const createClickSQL = `
INSERT INTO clicks (id, link_id, ip_hash, referrer, ua_raw)
VALUES ($1, $2, $3, $4, $5);
`
