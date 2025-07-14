package storage

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// A storage layer manages the set of URLs to be scraped.
type Storage interface {
	// Add the URLs to the queue.
	// The implementation must handle deduplication of already queued urls
	AddURLs(ctx context.Context, url []string) error

	// Retrieves the next URL and marks it as "Processing".
	GetNextURL(ctx context.Context) (QueuedURL, error)

	// Marks the URL as done.
	MarkDone(ctx context.Context, url string) error

	// Marks the URL as failed.
	MarkFailed(ctx context.Context, url string, msg string) error

	// Returns the number of URLs waiting in the queue.
	QueueSize(ctx context.Context) (int, error)

	Close()
}

type Status int

const (
	Queued Status = iota
	Processing
	Done
	Failed
)

type QueuedURL struct {
	URL    string
	Status Status
}

func (q *QueuedURL) FromRow(row pgx.Row) error {
	var statusStr string
	err := row.Scan(&q.URL, &statusStr)
	if err != nil {
		return err
	}
	q.Status = statusFromString(statusStr)
	return nil
}

func statusFromString(s string) Status {
	switch s {
	case "queued":
		return Queued
	case "processing":
		return Processing
	case "done":
		return Done
	case "failed":
		return Failed
	default:
		return Queued
	}
}
