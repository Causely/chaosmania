package actions

import (
	"context"
	"math/rand"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"go.uber.org/zap"
)

type Panic struct {
}

type PanicConfig struct {
	Probability float64 `json:"probability"`
}

func (a *Panic) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := pkg.ParseConfig[PanicConfig](cfg)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to parse config", zap.Error(err))
		return err
	}

	if config.Probability > 0 {
		if rand.Float64() < config.Probability {
			go func() {
				panic("Failed to execute action")
			}()
		}
	}

	return nil
}

func (a *Panic) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[PanicConfig](data)
}

func init() {
	ACTIONS["Panic"] = &Panic{}
}
