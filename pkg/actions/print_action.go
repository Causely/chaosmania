package actions

import (
	"context"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"go.uber.org/zap"
)

type Print struct{}

type PrintConfig struct {
	Message string `json:"message"`
}

func (a *Print) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := pkg.ParseConfig[PrintConfig](cfg)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to parse config", zap.Error(err))
		return err
	}

	logger, _ := zap.NewProduction()
	logger.Info(config.Message)

	return nil
}

func (a *Print) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[PrintConfig](data)
}

func init() {
	ACTIONS["Print"] = &Print{}
}
