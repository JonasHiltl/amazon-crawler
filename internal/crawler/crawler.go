package crawler

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/jonashiltl/amazon-crawler/internal"
	"github.com/jonashiltl/amazon-crawler/internal/consumer"
	"github.com/jonashiltl/amazon-crawler/internal/crawler/middleware"
	"github.com/jonashiltl/amazon-crawler/internal/polite"
	"github.com/jonashiltl/amazon-crawler/internal/storage"
	"github.com/playwright-community/playwright-go"
	"github.com/subsan/uafaker"
)

type crawler struct {
	Options
	browser             playwright.Browser
	pw                  *playwright.Playwright
	ctx                 context.Context
	log                 *slog.Logger
	jobs                chan string // channel holding the urls to process
	numWorkers          int         // number of workers to process the polled url
	errorCount          int32
	errorThreshold      int32                           // max number of errors before the crawler shuts down
	newURLS             chan []string                   // extracted urls to queue in storage
	requestMiddlewares  []middleware.RequestMiddleware  // exectued in order of their definition
	responseMiddlewares []middleware.ResponseMiddleware // executed in order of their definition
}

type Options struct {
	Consumer            consumer.Consumer
	Storage             storage.Storage
	SeedURLs            []string
	PollInterval        time.Duration
	Proxy               string
	ProxyPW             string
	ProxyUser           string
	PlaywrightDriverDir string
	Cancel              context.CancelFunc
}

func NewCrawler(ctx context.Context, opts Options) (*crawler, error) {
	log := internal.NewLogger("Crawler")

	if opts.PlaywrightDriverDir != "" {
		slog.Info("using custom playwright driver", slog.String("path", opts.PlaywrightDriverDir))
	}

	pw, err := playwright.Run(&playwright.RunOptions{
		DriverDirectory: opts.PlaywrightDriverDir,
	})
	if err != nil {
		return nil, fmt.Errorf("could not start playwright: %w", err)
	}

	log.Info(fmt.Sprintf("polling every %s for queued urls", opts.PollInterval))
	log.Info(fmt.Sprintf("using %d seed url", len(opts.SeedURLs)))

	numWorkers := 10
	c := &crawler{
		Options:        opts,
		pw:             pw,
		ctx:            ctx,
		log:            log,
		numWorkers:     numWorkers,
		errorThreshold: 5,
		jobs:           make(chan string, numWorkers*2),   // *2 gives buffer when workers can't keep up with poll volume
		newURLS:        make(chan []string, numWorkers*2), // each worker produces one []string, of newly found, relevant urls
		requestMiddlewares: []middleware.RequestMiddleware{
			middleware.NewRobotsMiddleware(polite.Options{
				Proxy:     opts.Proxy,
				ProxyPW:   opts.ProxyPW,
				ProxyUser: opts.ProxyUser,
			}),
		},
		responseMiddlewares: []middleware.ResponseMiddleware{
			middleware.NewLogMiddleware(),
			middleware.NewCaptchaMiddleware(),
			middleware.NewJSDisabledMiddleware(),
		},
	}

	return c, nil
}

func (c *crawler) Start() error {
	wsURL, err := c.startCamoufox()
	if err != nil {
		return err
	}

	err = c.connectBrowser(wsURL)
	if err != nil {
		return err
	}

	// start the specififed number of workers
	for i := range c.numWorkers {
		go c.worker(i)
	}
	c.startNewURLConsumer()

	// process seed urls
	for _, url := range c.SeedURLs {
		c.get(url)
		sleepWithJitter(c.PollInterval)
	}

	// poll for new urls
	c.poll()

	return nil
}

func (c *crawler) Close() {
	c.browser.Close()
}

func (c *crawler) startCamoufox() (string, error) {
	slog.Info("starting Camoufox browser")
	var proxyLine string
	if c.Proxy != "" {
		slog.Info(fmt.Sprintf("using proxy %s", c.Proxy))
		proxyLine = `
		geoip=True,
		proxy={
			'server':` + strconv.Quote(c.Proxy) + `,
			'username':` + strconv.Quote(c.ProxyUser) + `,
			'password':` + strconv.Quote(c.ProxyPW) + `
		}
		`
	}

	userAgent := uafaker.Windows().Firefox().Random()
	slog.Info(fmt.Sprintf("using User-Agent %s", userAgent))

	code := `
from camoufox.server import launch_server
from browserforge.fingerprints import Screen

launch_server(
    screen=Screen(max_width=1920, max_height=1080),
    headless="virtual",
	os="windows",
	config={
		"mediaDevices:enabled": True,
		"navigator.userAgent": ` + strconv.Quote(userAgent) + `
	},
    block_images=True,
	locale="en-US",
    port=9222,
    ws_path="play",
	i_know_what_im_doing=True,
	` + proxyLine + `
)`

	// use xvfb so that webgl is supported in docker container
	cmd := exec.Command("xvfb-run", "-a", "-e", "/dev/stdout", "python3", "-c", code)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start Camoufox: %w", err)
	}

	// monitor camoufox health and stop crawler if camoufox exists
	go func() {
		err := cmd.Wait()
		if err != nil {
			slog.Error("Camoufox exited with error", internal.ErrAttr(err))
		} else {
			slog.Warn("Camoufox exited cleanly")
		}
		c.Cancel()
	}()

	return "ws://localhost:9222/play", nil
}

func (c *crawler) connectBrowser(wsURL string) error {
	attempts := 5
	sleep := time.Second * 2
	for i := range attempts {
		if i > 0 {
			time.Sleep(sleep)
			sleep *= 2
		}
		browser, err := c.pw.Firefox.Connect(wsURL)
		if err != nil {
			continue
		}
		c.browser = browser
		c.log.Info(fmt.Sprintf("connected to browser at %s", wsURL))
		return nil
	}

	return fmt.Errorf("failed to connect to %s", wsURL)
}

func (c *crawler) poll() {
	for {
		select {
		case <-c.ctx.Done():
			c.log.Info("polling stopped")
			return
		default:
		}

		queuedURL, err := c.Storage.GetNextURL(c.ctx)
		if err != nil {
			c.log.Error(err.Error())
			sleepWithJitter(c.PollInterval)
			continue
		}

		if queuedURL.URL == "" {
			sleepWithJitter(c.PollInterval)
			continue
		}

		c.get(queuedURL.URL)
		sleepWithJitter(c.PollInterval)
	}
}

// Consumes the newly found urls and queues them in storage.
func (c *crawler) startNewURLConsumer() {
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				return
			case links, ok := <-c.newURLS:
				if !ok {
					return
				}
				err := c.Storage.AddURLs(c.ctx, links)
				if err != nil {
					c.log.Error(err.Error())
				}
			}
		}
	}()
}

// Adds the url to the internal job queue.
// Blocks if the channel (buffered) is full.
func (c *crawler) get(url string) {
	select {
	case c.jobs <- url:
	case <-c.ctx.Done():
		return
	}

}

func (c *crawler) worker(id int) {
	c.log.Info(fmt.Sprintf("created worker %d, waiting on urls...", id))
	for {
		select {
		case <-c.ctx.Done():
			c.log.Info(fmt.Sprintf("worker %d shutting down", id))
			return
		case url, ok := <-c.jobs:
			if !ok {
				return
			}
			c.processJob(url)
		}
	}
}

func (c *crawler) processJob(url string) {
	jobCtx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	links, err := c.processURL(jobCtx, url)
	if err != nil {
		c.onError(c.ctx, url, err)
		return
	}

	select {
	case c.newURLS <- links:
	case <-c.ctx.Done():
		return
	}

	if err := c.Storage.MarkDone(c.ctx, url); err != nil {
		c.log.Error("mark done error: " + err.Error())
	}
}

var blocked_resources = []string{"stylesheet", "font", "media", "image", "other", "xhr"}

// Fetches the url, parses the page and returns new relevant links.
func (c *crawler) processURL(ctx context.Context, url string) ([]string, error) {
	context, err := c.browser.NewContext()
	if err != nil {
		return nil, err
	}

	// use sync.Once to make sure Close is only called once
	var once sync.Once
	closeCtx := func() {
		once.Do(func() {
			context.Close()
		})
	}
	// Ensure it's closed at the end
	defer closeCtx()

	// Also close if context is cancelled
	go func() {
		<-ctx.Done()
		closeCtx()
	}()

	page, err := context.NewPage()
	if err != nil {
		return nil, err
	}
	page.Route("**/*", func(r playwright.Route) {
		if slices.Contains(blocked_resources, r.Request().ResourceType()) {
			r.Abort()
		} else {
			r.Continue()
		}
	})

	for _, mw := range c.requestMiddlewares {
		if err := mw.Process(ctx, url, page); err != nil {
			return nil, err
		}
	}

	res, err := page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if err != nil {
		return nil, err
	}
	if !res.Ok() {
		return nil, fmt.Errorf("response status %d", res.Status())
	}

	for _, mw := range c.responseMiddlewares {
		if err := mw.Process(ctx, url, page, res); err != nil {
			return nil, err
		}
	}

	c.resetErrorCount()

	if strings.Contains(url, "/dp/") {
		err := c.parseProductDetails(ctx, page)
		if err != nil {
			return nil, err
		}
	}

	return c.getRelevantLinks(page)
}

func (c *crawler) onError(ctx context.Context, url string, err error) {
	msg := err.Error()
	c.log.Error(msg, slog.String("url", url))
	err = c.Storage.MarkFailed(ctx, url, msg)
	if err != nil {
		c.log.Error(err.Error())
	}

	newCount := c.incrementErrorCount()
	if newCount > c.errorThreshold {
		c.log.Error("too many errors, shutting down", slog.Int("count", int(newCount)))
		c.Cancel()
	}
}

func (c *crawler) incrementErrorCount() int32 {
	return atomic.AddInt32(&c.errorCount, 1)
}

func (c *crawler) resetErrorCount() {
	atomic.StoreInt32(&c.errorCount, 0)
}

func (c *crawler) takeScreenshot(p playwright.Page) {
	fileName := strings.ReplaceAll(p.URL(), "/", "-") + ".png"
	if _, err := p.Screenshot(playwright.PageScreenshotOptions{
		Path:     playwright.String(fileName),
		FullPage: playwright.Bool(true),
	}); err != nil {
		c.log.Error("could not create screenshot", internal.ErrAttr(err))
	} else {
		c.log.Info("saved screenshot", slog.String("path", fileName))
	}
}

func (c *crawler) parseProductDetails(ctx context.Context, page playwright.Page) error {
	product, err := internal.ProductFromPage(page)
	if err != nil {
		return fmt.Errorf("failed to parse product: %w", err)
	}
	c.log.Debug("product parsed", slog.String("url", page.URL()))

	err = c.Consumer.Consume(ctx, product)
	if err != nil {
		return fmt.Errorf("failed to consume product: %w", err)
	}
	return nil
}

// Finds all relevant links, e.g. product details or search pages and adds them to the queue
func (c *crawler) getRelevantLinks(page playwright.Page) ([]string, error) {
	links := mapset.NewThreadUnsafeSet[string]()

	a, err := page.Locator("a[href]").All()
	if err == nil {
		for _, link := range a {
			href, err := link.GetAttribute("href")
			if err != nil {
				continue
			}

			if asin, err := internal.AsinFromURL(href); err == nil {
				links.Add(createProductURL(asin))
			}

			if isRelevantURL(href) {
				links.Add(withBaseURL(href))
			}
		}
	}

	a, err = page.Locator("a.s-pagination-next, a#apb-desktop-browse-search-see-all").All()
	if err == nil {
		for _, link := range a {
			href, err := link.GetAttribute("href")
			if err != nil {
				continue
			}
			links.Add(withBaseURL(href))
		}
	}

	slice := links.ToSlice()
	c.log.Debug(fmt.Sprintf("found %d relevant links", len(slice)))
	return slice, nil
}

func sleepWithJitter(base time.Duration) {
	factor := 0.5 + rand.Float64()
	delay := time.Duration(float64(base) * factor)
	time.Sleep(delay)
}
