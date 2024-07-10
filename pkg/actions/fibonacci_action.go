package actions

import (
	"context"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"go.uber.org/zap"
)

type Fibonacci struct{}

type FibonacciConfig struct {
	Value uint64 `json:"value"`
}

func fibonacci(n uint64) uint64 {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}

func (a *Fibonacci) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := pkg.ParseConfig[FibonacciConfig](cfg)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to parse config", zap.Error(err))
		return err
	}

	_ = fibonacci(config.Value)
	return nil
}

func (a *Fibonacci) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[FibonacciConfig](data)
}

func init() {
	ACTIONS["Fibonacci"] = &Fibonacci{}
}
