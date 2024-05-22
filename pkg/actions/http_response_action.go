package actions

import (
	"context"
	"net/http"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"go.uber.org/zap"
)

type HTTPResponse struct{}

type HTTPResponseConfig struct {
	StatusCode int `json:"statusCode"`
}

type ContextKey string

const ResponseWriterKey ContextKey = "http.ResponseWriter"

func (a *HTTPResponse) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := pkg.ParseConfig[HTTPResponseConfig](cfg)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to parse config", zap.Error(err))
		return err
	}

	val := ctx.Value(ResponseWriterKey)
	if val == nil {
		panic("http.ResponseWriter not found in context")
	}

	w := val.(http.ResponseWriter)
	w.WriteHeader(config.StatusCode)
	return nil
}

func (a *HTTPResponse) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[HTTPResponseConfig](data)
}

func init() {
	ACTIONS["HTTPResponse"] = &HTTPResponse{}
}
