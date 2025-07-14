package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jonashiltl/amazon-crawler/internal"
	"github.com/jonashiltl/amazon-crawler/internal/config"
	"github.com/jonashiltl/amazon-crawler/internal/consumer"
	"github.com/jonashiltl/amazon-crawler/internal/crawler"
	"github.com/jonashiltl/amazon-crawler/internal/storage"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", internal.ErrAttr(err))
		os.Exit(1)
	}

	setDefaultLogger(&cfg)

	consumer, err := createConsumer(&cfg)
	if err != nil {
		slog.Error("failed to create consumer", internal.ErrAttr(err))
		os.Exit(1)
	}

	storage, err := storage.NewPGStorage(storage.PGOptions{
		DatabaseURL: cfg.PostgresURL,
	})
	if err != nil {
		slog.Error("failed to create postgres storage", internal.ErrAttr(err))
		os.Exit(1)
	}

	crawl, err := crawler.NewCrawler(ctx, crawler.Options{
		Consumer:            consumer,
		Storage:             storage,
		SeedURLs:            cfg.SeedURLs,
		PollInterval:        cfg.PollInterval,
		Proxy:               cfg.Proxy,
		ProxyPW:             cfg.ProxyPW,
		ProxyUser:           cfg.ProxyUser,
		PlaywrightDriverDir: cfg.PlaywrightDriverDir,
		Cancel:              cancel,
	})
	if err != nil {
		slog.Error("failed to create crawler", internal.ErrAttr(err))
		return
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- crawl.Start()
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("exited with error", internal.ErrAttr(err))
		}
	}

	crawl.Close()
	consumer.Close()
	storage.Close()
}

func createConsumer(cfg *config.Config) (consumer.Consumer, error) {
	if len(cfg.OpensearchAddresses) > 0 {
		return consumer.NewOpensearchConsumer(
			consumer.WithAddresses(cfg.OpensearchAddresses),
			consumer.WithUsername(cfg.OpensearchUsername),
			consumer.WithPassword(cfg.OpensearchPassword),
			consumer.TrustTLS(),
		)
	}

	slog.Info("printing crawled products to stdout")
	return consumer.NewStdoutConsumer(), nil
}

func setDefaultLogger(cfg *config.Config) {
	level := cfg.LogLevel.ToSlog()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	logger.Info(fmt.Sprintf("using log level %s", level))
}
