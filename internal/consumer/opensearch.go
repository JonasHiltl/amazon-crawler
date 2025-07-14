package consumer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jonashiltl/amazon-crawler/internal"
	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

type OpensearchConsumerOption func(*opensearch.Config)

func WithAddresses(addresses []string) OpensearchConsumerOption {
	return func(cfg *opensearch.Config) {
		cfg.Addresses = addresses
	}
}

func WithUsername(username string) OpensearchConsumerOption {
	return func(cfg *opensearch.Config) {
		cfg.Username = username
	}
}

func WithPassword(password string) OpensearchConsumerOption {
	return func(cfg *opensearch.Config) {
		cfg.Password = password
	}
}

func TrustTLS() OpensearchConsumerOption {
	return func(c *opensearch.Config) {
		c.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}
}

const (
	indexName         = "amzn-products"
	flushSize         = 10
	flushIntervalSecs = 60
)

type osconsumer struct {
	client *opensearchapi.Client
	log    *slog.Logger
	buffer []internal.Product
	mu     sync.Mutex
}

func NewOpensearchConsumer(opts ...OpensearchConsumerOption) (Consumer, error) {
	cfg := opensearch.Config{}
	for _, opt := range opts {
		opt(&cfg)
	}

	logger := internal.NewLogger("OpensearchConsumer")
	logger.Info("initializing", slog.Any("addresses", cfg.Addresses), slog.String("username", cfg.Username))

	client, err := opensearchapi.NewClient(opensearchapi.Config{
		Client: cfg,
	})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	_, err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	c := &osconsumer{
		client: client,
		log:    logger,
	}
	err = c.createIndex(ctx)
	if err != nil {
		return nil, err
	}

	c.startFlushThread()

	return c, nil
}

func (o *osconsumer) Consume(ctx context.Context, prd internal.Product) error {
	shouldFlush := false

	o.mu.Lock()
	o.buffer = append(o.buffer, prd)
	if len(o.buffer) >= flushSize {
		shouldFlush = true
	}
	o.mu.Unlock()

	if shouldFlush {
		o.flush(ctx)
	}

	return nil
}

func (o *osconsumer) Close() {
	o.log.Info("closing consumer, flushing buffered posts")
	o.flush(context.Background())
}

func (o *osconsumer) createIndex(ctx context.Context) error {
	_, err := o.client.Indices.Exists(ctx, opensearchapi.IndicesExistsReq{Indices: []string{indexName}})
	if err != nil {
		o.log.Info("creating opensearch index", slog.String("indexName", indexName))
		_, err = o.client.Indices.Create(ctx, opensearchapi.IndicesCreateReq{
			Index: indexName,
			Body: strings.NewReader(`{
            "mappings": {
                "properties": {
                    "asin":                     { "type": "keyword" },
                    "title":                    { "type": "text" },
                    "description":              { "type": "text" },
                    "aboutItem":                { "type": "text" },
                    "brand":                    { "type": "keyword" },
                    "manufacturer":             { "type": "keyword" },
                    "ageRange":                 { "type": "keyword" },
                    "weight":                   { "type": "text" },
                    "material":                 { "type": "text" },
                    "color":                    { "type": "text" },
                    "origin":                 	{ "type": "keyword" },
                    "dimensions":               { "type": "text" },
                    "sustainabilityFeatures":   { "type": "keyword" },
                    "averageRating": 			{ "type": "float" },
                    "ratings": 					{ "type": "integer" },
                    "isAmazonChoice":           { "type": "boolean" },
                    "images":                   { "type": "keyword" },
                    "boughtTogetherAsins":      { "type": "keyword" },
                    "categories":               { "type": "keyword" },
                    "listPrice":                { "type": "float" },
                    "discountedPrice":          { "type": "float" },
                    "currency":                 { "type": "keyword" },
                    "sellerId":                 { "type": "keyword" },
                    "firstAvailableAt":         { "type": "date" },
                    "boughtPastMonth":          { "type": "integer" },
                    "bestSellers": {
                        "properties" : {
                            "category":         { "type": "keyword" },
                            "rank":             { "type": "integer" }
                        }
                    }
                }
            }
        }`),
		})
		return err
	}

	o.log.Info("opensearch index exists", slog.String("indexName", indexName))
	return nil
}

func (o *osconsumer) flush(ctx context.Context) {
	o.mu.Lock()
	if len(o.buffer) == 0 {
		o.mu.Unlock()
		return
	}

	postsLen := len(o.buffer)
	actions := make([]string, 0, postsLen*2)
	for _, products := range o.buffer {
		meta := map[string]map[string]string{
			"index": {"_index": indexName, "_id": products.ASIN},
		}
		metaLine, err := json.Marshal(meta)
		if err != nil {
			o.log.Debug("failed to marshal meta line", internal.ErrAttr(err))
			continue
		}
		docLine, err := json.Marshal(products)
		if err != nil {
			o.log.Debug("failed to marshal doc line", internal.ErrAttr(err))
			continue
		}
		actions = append(actions, string(metaLine), string(docLine))
	}
	o.buffer = nil
	o.mu.Unlock()

	var buf bytes.Buffer
	for _, line := range actions {
		buf.WriteString(line + "\n")
	}
	o.log.Info("performing bulk request", slog.Int("posts", postsLen))
	bulkRes, err := o.client.Bulk(ctx, opensearchapi.BulkReq{
		Body: &buf,
	})
	if err != nil {
		o.log.Error("failed to perform bulk request", internal.ErrAttr(err))
	}
	if bulkRes.Errors {
		o.log.Warn("got error items on bulk request")
	}
}
func (o *osconsumer) startFlushThread() {
	interval := time.Second * flushIntervalSecs
	o.log.Info(fmt.Sprintf("flushing every %s", interval))
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			o.flush(context.Background())
		}
	}()
}
