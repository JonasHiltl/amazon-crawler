package consumer

import (
	"context"

	"github.com/jonashiltl/amazon-crawler/internal"
)

type Consumer interface {
	Consume(ctx context.Context, prd internal.Product) error
	Close()
}
