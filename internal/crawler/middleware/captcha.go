package middleware

import (
	"context"
	"errors"

	playwright "github.com/playwright-community/playwright-go"
)

type captchaMiddleware struct{}

func NewCaptchaMiddleware() ResponseMiddleware {
	return captchaMiddleware{}
}

func (j captchaMiddleware) Process(ctx context.Context, url string, page playwright.Page, res playwright.Response) error {
	visible, err := page.Locator("input#captchacharacters,div#challenge-container").IsVisible()
	if visible && err == nil {
		return errors.New("blocked with captcha")
	}
	return nil
}
