package actions

import (
	"context"
	"net/http"

	"github.com/Causely/chaosmania/pkg/logger"
	"go.uber.org/zap"
)

type HTTPResponse struct{}

type HTTPResponseConfig struct {
	StatusCode int `json:"statusCode"`
}

func (a *HTTPResponse) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := ParseConfig[HTTPResponseConfig](cfg)
	if err != nil {
		logger.NewLogger().Warn("failed to parse config", zap.Error(err))
		return err
	}

	val := ctx.Value("http.ResponseWriter")
	if val == nil {
		return nil
	}

	w := val.(http.ResponseWriter)
	w.WriteHeader(config.StatusCode)
	return nil
}

func (a *HTTPResponse) ParseConfig(data map[string]any) (any, error) {
	return ParseConfig[HTTPResponseConfig](data)
}

func init() {
	ACTIONS["HTTPResponse"] = &HTTPResponse{}
}
