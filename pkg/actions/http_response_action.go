package actions

import (
	"context"
	"math/rand"
	"net/http"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"go.uber.org/zap"
)

type HTTPResponse struct{}

type HTTPResponseConfig struct {
	StatusCode                   int             `json:"statusCode,omitempty"`
	StatusCodesWithProbabilities map[int]float64 `json:"statusCodesWithProbabilities,omitempty"`
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

	var statusCode int
	if config.StatusCode != 0 {
		// Use the single status code with probability 1 if defined
		statusCode = config.StatusCode
	} else {
		// Otherwise, select a status code based on the provided probabilities
		statusCode = selectStatusCode(config.StatusCodesWithProbabilities)
	}

	w.WriteHeader(statusCode)
	return nil
}

func selectStatusCode(statusCodes map[int]float64) int {
	var totalProb float64
	for _, prob := range statusCodes {
		totalProb += prob
	}

	randValue := rand.Float64() * totalProb
	var cumulativeProb float64

	for statusCode, prob := range statusCodes {
		cumulativeProb += prob
		if randValue < cumulativeProb {
			return statusCode
		}
	}

	// Default to 200 if no status code is selected
	return http.StatusOK
}

func (a *HTTPResponse) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[HTTPResponseConfig](data)
}

func init() {
	ACTIONS["HTTPResponse"] = &HTTPResponse{}
}
