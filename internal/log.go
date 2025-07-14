package internal

import (
	"log/slog"
)

func NewLogger(service string) *slog.Logger {
	return slog.Default().With(
		"service", service,
	)
}

func ErrAttr(err error) slog.Attr {
	return slog.Any("error", err)
}
