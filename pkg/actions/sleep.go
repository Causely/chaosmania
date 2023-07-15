package actions

import (
	"context"
	"time"

	"github.com/Causely/chaosmania/pkg/logger"
	"go.uber.org/zap"
)

type Sleep struct{}

type SleepConfig struct {
	Duration Duration `json:"duration"`
}

func (a *Sleep) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := ParseConfig[SleepConfig](cfg)
	if err != nil {
		logger.NewLogger().Warn("failed to parse config", zap.Error(err))
		return err
	}

	select {
	case <-ctx.Done():
	case <-time.After(config.Duration.Duration):
	}

	return nil
}

func (a *Sleep) ParseConfig(data map[string]any) (any, error) {
	return ParseConfig[SleepConfig](data)
}

func init() {
	ACTIONS["Sleep"] = &Sleep{}
}
