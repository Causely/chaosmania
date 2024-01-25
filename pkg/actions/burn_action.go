package actions

import (
	"context"
	"time"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"go.uber.org/zap"
)

type Burn struct{}

type BurnConfig struct {
	Duration pkg.Duration `json:"duration"`
}

func (a *Burn) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := pkg.ParseConfig[BurnConfig](cfg)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to parse config", zap.Error(err))
		return err
	}

	end := time.Now().Add(config.Duration.Duration)
	for time.Now().Before(end) {
	}

	return nil
}

func (a *Burn) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[BurnConfig](data)
}

func init() {
	ACTIONS["Burn"] = &Burn{}
}
