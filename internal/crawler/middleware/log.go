package middleware

import (
	"context"
	"log/slog"

	playwright "github.com/playwright-community/playwright-go"
)

type logMiddleware struct{}

func NewLogMiddleware() ResponseMiddleware {
	return logMiddleware{}
}

func (j logMiddleware) Process(ctx context.Context, url string, page playwright.Page, res playwright.Response) error {
	slog.Info(url, slog.Int("status", res.Status()))
	return nil
}
