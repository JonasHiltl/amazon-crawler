package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jonashiltl/amazon-crawler/internal"
)

type stdoutConsumer struct {
}

func NewStdoutConsumer() Consumer {
	return &stdoutConsumer{}
}

func (s *stdoutConsumer) Consume(ctx context.Context, prd internal.Product) error {
	marshalled, _ := json.Marshal(prd)
	fmt.Println(string(marshalled))
	return nil
}

func (s *stdoutConsumer) Close() {}
