package proxy

import (
	"errors"
	"sync"

	"github.com/playwright-community/playwright-go"
)

type proxyManager struct {
	Options
	mu    sync.Mutex
	index int
}

type Options struct {
	Proxies  []string
	Username string
	Password string
}

func NewProxyManager(opts Options) *proxyManager {
	return &proxyManager{
		Options: opts,
	}
}

func (pm *proxyManager) RoundRobin() (playwright.Proxy, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.Proxies) == 0 {
		return playwright.Proxy{}, errors.New("no proxies available")
	}

	proxyURL := pm.Proxies[pm.index]
	pm.index = (pm.index + 1) % len(pm.Proxies)

	return playwright.Proxy{
		Server:   proxyURL,
		Username: playwright.String(pm.Username),
		Password: playwright.String(pm.Password),
	}, nil
}
