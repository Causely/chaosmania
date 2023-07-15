package actions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Causely/chaosmania/pkg/logger"
	"go.uber.org/zap"
)

type HTTPRequest struct{}

type HTTPRequestConfig struct {
	Url  string         `json:"url"`
	Body map[string]any `json:"body"`
}

func (a *HTTPRequest) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := ParseConfig[HTTPRequestConfig](cfg)
	if err != nil {
		logger.NewLogger().Warn("failed to parse config", zap.Error(err))
		return err
	}

	payloadBytes, err := json.Marshal(Convert(config.Body))
	if err != nil {
		logger.NewLogger().Warn("failed marshal json", zap.Error(err))
		return err
	}

	// Create a new HTTP POST request with the payload
	req, err := http.NewRequestWithContext(ctx, "POST", config.Url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		logger.NewLogger().Warn("failed to create new request", zap.Error(err))
		return err
	}

	// Set the content type header to indicate a JSON payload
	req.Header.Set("Content-Type", "application/json")

	// Send the request to the server
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.NewLogger().Warn("failed to send request", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			logger.NewLogger().Warn("failed to read body", zap.Error(err))
			return err
		}
		return fmt.Errorf("request failed (%v): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (a *HTTPRequest) ParseConfig(data map[string]any) (any, error) {
	return ParseConfig[HTTPRequestConfig](data)
}

func init() {
	ACTIONS["HTTPRequest"] = &HTTPRequest{}
}
