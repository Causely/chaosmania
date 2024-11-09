package actions

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap"
)

type ElasticQuery struct{}

type ElasticQueryConfig struct {
	Query    string `json:"query"`
	Index    string `json:"index"`
	Address  string `json:"address"`
	ApiKey   string `json:"apikey"`
	Insecure bool   `json:"insecure"`
}

func (a *ElasticQuery) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := pkg.ParseConfig[ElasticQueryConfig](cfg)
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

	// Perform the search request
	res, err := es.Search(
		es.Search.WithContext(ctx),
		es.Search.WithIndex(config.Index),
		es.Search.WithBody(strings.NewReader(config.Query)),
		es.Search.WithPretty(),
	)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to execute search query", zap.Error(err))
		return err
	}

	defer res.Body.Close()

	// Decode and display the search results
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		logger.FromContext(ctx).Warn("failed to decode response body", zap.Error(err))
		return err
	}

	// Print the result in a readable format
	fmt.Printf("Search Results:\n")
	fmt.Println(result)

	return nil
}

func (a *ElasticQuery) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[ElasticQueryConfig](data)
}

func init() {
	ACTIONS["ElasticQuery"] = &ElasticQuery{}
}
