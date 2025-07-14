package middleware

import (
	"context"

	"github.com/playwright-community/playwright-go"
)

// Called after the page is created but before the url is requested.
type RequestMiddleware interface {
	Process(ctx context.Context, url string, page playwright.Page) error
}

// Called after the url is requested
type ResponseMiddleware interface {
	Process(ctx context.Context, url string, page playwright.Page, res playwright.Response) error
}
