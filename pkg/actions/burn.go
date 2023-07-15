package actions

import (
	"context"
	"time"

	"github.com/Causely/chaosmania/pkg/logger"
	"go.uber.org/zap"
)

type Burn struct{}

type BurnConfig struct {
	Duration Duration `json:"duration"`
}

func (a *Burn) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := ParseConfig[BurnConfig](cfg)
	if err != nil {
		logger.NewLogger().Warn("failed to parse config", zap.Error(err))
		return err
	}

	end := time.Now().Add(config.Duration.Duration)
	for time.Now().Before(end) {
	}

	return nil
}

func (a *Burn) ParseConfig(data map[string]any) (any, error) {
	return ParseConfig[BurnConfig](data)
}

func init() {
	ACTIONS["Burn"] = &Burn{}
}
