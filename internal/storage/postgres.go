package storage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jonashiltl/amazon-crawler/internal"
)

type PGOptions struct {
	DatabaseURL string
}

type pgStorage struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

func NewPGStorage(opts PGOptions) (Storage, error) {
	if opts.DatabaseURL == "" {
		return nil, errors.New("missing POSTGRES_URL config variable")
	}

	ctx := context.Background()
	dbpool, err := pgxpool.New(ctx, opts.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	s := &pgStorage{
		pool: dbpool,
		log:  internal.NewLogger("PGStorage"),
	}
	err = s.ensureSchema(ctx)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (p *pgStorage) Close() {
	p.pool.Close()
}

func (p *pgStorage) AddURLs(ctx context.Context, urls []string) error {
	if len(urls) == 0 {
		return nil
	}

	p.log.Debug("batch inserting urls", slog.Int("len", len(urls)))
	batch := &pgx.Batch{}
	for _, url := range urls {
		batch.Queue(`
            INSERT INTO url_queue (url, status)
            VALUES ($1, 'queued')
            ON CONFLICT (url) DO NOTHING
        `, url)
	}

	br := p.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range urls {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("batch insert error: %w", err)
		}
	}

	return nil
}

func (p *pgStorage) GetNextURL(ctx context.Context) (QueuedURL, error) {
	var q QueuedURL
	// selects the next url and marks it "processing" in a single query
	// FOR UPDATE SKIP LOCKED ensures only one process retrieves and locks urls
	row := p.pool.QueryRow(ctx, `
		WITH next_url AS (
			SELECT url
			FROM url_queue
			WHERE 
				status = 'queued'
				OR (status = 'processing' AND started_at < now() - INTERVAL '5 minute')
				OR (
        			status = 'failed'
        			AND NOW() >= failed_at + INTERVAL '5 minutes' * POWER(2, GREATEST(retry_count - 1, 0))
        			AND retry_count < 3
    			)
			ORDER BY id
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE url_queue
		SET status = 'processing', started_at = now()
		FROM next_url
		WHERE url_queue.url = next_url.url
		RETURNING url_queue.url, url_queue.status
	`)
	err := q.FromRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return QueuedURL{}, fmt.Errorf("no queued url available")
		}
		return QueuedURL{}, fmt.Errorf("failed to get next url: %w", err)
	}

	return q, nil
}

func (p *pgStorage) MarkDone(ctx context.Context, url string) error {
	_, err := p.pool.Exec(ctx, `
		UPDATE url_queue
		SET status = 'done', done_at = NOW()
		WHERE url = $1
	`, url)
	if err != nil {
		return fmt.Errorf("failed to mark %s as done: %w", url, err)
	}
	return nil
}

func (p *pgStorage) MarkFailed(ctx context.Context, url string, msg string) error {
	_, err := p.pool.Exec(ctx, `
		UPDATE url_queue
		SET status = 'failed', failed_at = NOW(), retry_count = retry_count + 1, reason = $1
		WHERE url = $2
	`, msg, url)
	if err != nil {
		return fmt.Errorf("failed to mark %s as failed: %w", url, err)
	}
	return nil
}

func (p *pgStorage) QueueSize(ctx context.Context) (int, error) {
	var count int
	err := p.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM url_queue WHERE status = 'queued'
	`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue size: %w", err)
	}
	return count, nil
}

func (p *pgStorage) ensureSchema(ctx context.Context) error {
	migration := `
    CREATE TABLE IF NOT EXISTS url_queue (
        id SERIAL PRIMARY KEY,
        url TEXT UNIQUE NOT NULL,
        status TEXT NOT NULL CHECK (status IN ('queued', 'processing', 'done', 'failed')),
		reason TEXT,
        queued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		started_at TIMESTAMPTZ,
		done_at TIMESTAMPTZ,
		failed_at TIMESTAMPTZ,
		retry_count INT NOT NULL DEFAULT 0
    );

    CREATE INDEX IF NOT EXISTS idx_url_queue_status ON url_queue (status, started_at);
    `
	if _, err := p.pool.Exec(ctx, migration); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	return nil
}
