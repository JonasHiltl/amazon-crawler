package middleware

import (
	"context"

	"github.com/jonashiltl/amazon-crawler/internal/polite"
	"github.com/playwright-community/playwright-go"
)

type robotsMiddleware struct {
	polite.Options
	robots *polite.RobotsChecker
}

// Detects whether the request is forbidden by the pages robots.txt
func NewRobotsMiddleware(opts polite.Options) RequestMiddleware {
	robots := polite.NewRobotsChecker(polite.Options{
		Proxy:     opts.Proxy,
		ProxyPW:   opts.ProxyPW,
		ProxyUser: opts.ProxyUser,
	})

	return &robotsMiddleware{
		Options: opts,
		robots:  robots,
	}
}

func (r *robotsMiddleware) Process(ctx context.Context, url string, page playwright.Page) error {
	uaResult, err := page.Evaluate("navigator.userAgent")
	if err != nil {
		// if userAgent can't be found let request pass
		return nil
	}
	ua, ok := uaResult.(string)
	if !ok {
		return nil // Do nothing
	}

	err = r.robots.Check(url, ua)
	if err != nil {
		return err
	}
	return nil
}
