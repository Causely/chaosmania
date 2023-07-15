package actions

import (
	"context"
	"github.com/Causely/chaosmania/pkg/logger"
	"go.uber.org/zap"
)

type Print struct{}

type PrintConfig struct {
	Message string `json:"message"`
}

func (a *Print) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := ParseConfig[PrintConfig](cfg)
	if err != nil {
		logger.NewLogger().Warn("failed to parse config", zap.Error(err))
		return err
	}

	if logger.NewLogger() != nil {
		logger.NewLogger().Info(config.Message)
	}

	return nil
}

func (a *Print) ParseConfig(data map[string]any) (any, error) {
	return ParseConfig[PrintConfig](data)
}

func init() {
	ACTIONS["Print"] = &Print{}
}
