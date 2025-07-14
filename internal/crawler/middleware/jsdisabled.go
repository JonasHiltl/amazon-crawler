package middleware

import (
	"context"
	"errors"

	playwright "github.com/playwright-community/playwright-go"
)

type jsDisabledMiddleware struct{}

func NewJSDisabledMiddleware() ResponseMiddleware {
	return jsDisabledMiddleware{}
}

func (j jsDisabledMiddleware) Process(ctx context.Context, url string, page playwright.Page, res playwright.Response) error {
	visible, err := page.Locator("noscript:has-text(\"javascript is disabled\")").IsVisible()
	if visible && err == nil {
		return errors.New("js is disabled")
	}
	return nil
}
