package actions

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap"
)

type ElasticInsert struct{}

type ElasticInsertConfig struct {
	Document map[string]any `json:"document"`
	Index    string         `json:"index"`
	Address  string         `json:"address"`
	ApiKey   string         `json:"apikey"`
	Insecure bool           `json:"insecure"`
}

func (a *ElasticInsert) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := pkg.ParseConfig[ElasticInsertConfig](cfg)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to parse config", zap.Error(err))
		return err
	}
	esConfig := elasticsearch.Config{
		Addresses: []string{config.Address},
		APIKey:    config.ApiKey,
	}
	if config.Insecure {
		esConfig.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	es, err := elasticsearch.NewClient(esConfig)

	if err != nil {
		logger.FromContext(ctx).Warn("failed to create client", zap.Error(err))
		return err
	}

	// Convert the document to JSON format
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(config.Document); err != nil {
		logger.FromContext(ctx).Warn("failed to encode document", zap.Error(err))
		return err
	}

	// Insert the document
	res, err := es.Index(
		config.Index,                 // The index name
		&buf,                         // The document body
		es.Index.WithDocumentID("1"), // Optional: specify a document ID
		es.Index.WithContext(context.Background()),
		es.Index.WithPretty(),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		fmt.Printf("Error indexing document ID=1 to index %s: %s", config.Index, res.String())
	} else {
		fmt.Printf("Document indexed successfully in index %s\n", config.Index)
	}
	return nil
}

func (a *ElasticInsert) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[ElasticInsertConfig](data)
}

func init() {
	ACTIONS["ElasticInsert"] = &ElasticInsert{}
}
