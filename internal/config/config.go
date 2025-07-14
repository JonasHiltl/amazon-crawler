package config

import (
	"log/slog"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type LogLevel string

func (l LogLevel) ToSlog() slog.Level {
	var sl slog.Level
	sl.UnmarshalText([]byte(l))
	return sl
}

type Config struct {
	OpensearchAddresses []string      `env:"OPENSEARCH_ADDRESSES"`
	OpensearchUsername  string        `env:"OPENSEARCH_USERNAME"`
	OpensearchPassword  string        `env:"OPENSEARCH_PASSWORD"`
	PostgresURL         string        `env:"POSTGRES_URL" env-required:"true"`
	PollInterval        time.Duration `env:"POLL_INTERVAL" env-default:"6s" env-required:"true"`
	Proxy               string        `env:"PROXY"`
	ProxyPW             string        `env:"PROXY_PASSWORD"`
	ProxyUser           string        `env:"PROXY_USERNAME"`
	SeedURLs            []string      `env:"SEED_URLS"`
	PlaywrightDriverDir string        `env:"PLAYWRIGHT_DRIVER_DIR"`
	LogLevel            LogLevel      `env:"LOG_LEVEL"`
}

func LoadConfig() (Config, error) {
	var cfg Config
	cleanenv.ReadConfig(".env", &cfg)
	err := cleanenv.ReadEnv(&cfg)
	return cfg, err
}
