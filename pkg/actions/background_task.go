package actions

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Causely/chaosmania/pkg/logger"
	"go.uber.org/zap"
)

type task struct{}

type BackgroundTask struct {
	tasks map[string]task
	mu    sync.Mutex
}

type BackgroundTaskConfig struct {
	Id       string   `json:"id"`
	Duration Duration `json:"duration"`
	Workload Workload `json:"workload"`
}

func (a *BackgroundTask) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := ParseConfig[BackgroundTaskConfig](cfg)
	if err != nil {
		logger.NewLogger().Warn("failed to parse config", zap.Error(err))
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.tasks[config.Id]; ok {
		msg := fmt.Sprintf("background task with id %s already exists", config.Id)
		logger.NewLogger().Warn(msg)
		return fmt.Errorf(msg)
	}

	go func() {
		ctx := context.Background()
		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(config.Duration.Duration))
		defer cancel()

		err := config.Workload.Execute(ctx)
		if err != nil {
			logger.NewLogger().Error(err.Error())
		}

		a.mu.Lock()
		defer a.mu.Unlock()

		delete(a.tasks, config.Id)
	}()

	return nil
}

func (a *BackgroundTask) ParseConfig(data map[string]any) (any, error) {
	c, err := ParseConfig[BackgroundTaskConfig](data)

	if err != nil {
		return nil, err
	}

	return c, c.Workload.Verify()
}

func init() {
	ACTIONS["BackgroundTask"] = &BackgroundTask{}
}
