package actions

import (
	"context"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"go.uber.org/zap"
)

type GlobalMutexUnlock struct{}

type GlobalMutexUnlockConfig struct {
	Id       string   `json:"id"`
	Workload Workload `json:"workload"`
}

func (a *GlobalMutexUnlock) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := pkg.ParseConfig[GlobalMutexUnlockConfig](cfg)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to parse config", zap.Error(err))
		return err
	}

	GLOBAL_MUTEX.Lock()
	defer GLOBAL_MUTEX.Unlock()

	_, ok := GLOBAL_MUTEX_LOCKS[config.Id]
	if ok {
		GLOBAL_MUTEX_LOCKS[config.Id].Unlock()
	}

	return nil
}

func (a *GlobalMutexUnlock) ParseConfig(data map[string]any) (any, error) {
	c, err := pkg.ParseConfig[GlobalMutexUnlockConfig](data)

	if err != nil {
		return nil, err
	}

	return c, c.Workload.Verify()
}

func init() {
	ACTIONS["GlobalMutexUnlock"] = &GlobalMutexUnlock{}
}
