package polite

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"

	"github.com/jonashiltl/amazon-crawler/internal"
	"github.com/temoto/robotstxt"
)

type Options struct {
	Proxy     string
	ProxyPW   string
	ProxyUser string
}

func NewRobotsChecker(opts Options) *RobotsChecker {
	log := internal.NewLogger("RobotsChecker")
	transport := &http.Transport{}

	if opts.Proxy != "" {
		log.Debug("using proxy", slog.String("host", opts.Proxy))
		proxyURL, err := url.Parse(opts.Proxy)
		if err == nil {
			if opts.ProxyUser != "" && opts.ProxyPW != "" {
				log.Debug("using proxy username and password")
				proxyURL.User = url.UserPassword(opts.ProxyUser, opts.ProxyPW)
			}
			transport.Proxy = http.ProxyURL(proxyURL)
		} else {
			slog.Warn("failed to parse proxy", slog.String("host", opts.Proxy))
		}
	}

	return &RobotsChecker{
		Options: opts,
		log:     log,
		client: &http.Client{
			Transport: transport,
		},
		robotsMap: make(map[string]*robotstxt.RobotsData),
	}
}

type RobotsChecker struct {
	Options
	log       *slog.Logger
	client    *http.Client
	mut       sync.RWMutex
	robotsMap map[string]*robotstxt.RobotsData
}

// Checks if the robots.txt forbids access of the User-Agent.
// It returns an error if access is explicitly forbidden by robots.txt,
func (r *RobotsChecker) Check(rawURL string, ua string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil // error but allow access
	}

	r.mut.RLock()
	robotsData, exists := r.robotsMap[parsed.Host]
	r.mut.RUnlock()

	if !exists {
		robotsData, err = r.getRobotsData(parsed)
		if err != nil {
			r.log.Error("reading robots.txt", internal.ErrAttr(err))
			return nil // error but allow access
		}

		r.mut.Lock()
		r.robotsMap[parsed.Host] = robotsData
		r.mut.Unlock()
	}

	if !robotsData.TestAgent(parsed.Path, ua) {
		return errors.New("forbidden by robots.txt")
	}
	return nil
}

func (r *RobotsChecker) getRobotsData(url *url.URL) (*robotstxt.RobotsData, error) {
	requestURL := url.Scheme + "://" + url.Host + "/robots.txt"
	resp, err := r.client.Get(requestURL)
	if err != nil {
		return nil, err
	}

	r.log.Info(requestURL, slog.Int("status", resp.StatusCode))

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	return robotstxt.FromStatusAndBytes(resp.StatusCode, body)
}
