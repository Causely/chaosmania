package actions

import (
	"context"
	"sync"

	"github.com/Causely/chaosmania/pkg/logger"
	"go.uber.org/zap"
)

var GLOBAL_MUTEX sync.Mutex
var GLOBAL_MUTEX_LOCKS map[string]*sync.Mutex = make(map[string]*sync.Mutex)

type GlobalMutexLock struct{}

type GlobalMutexLockConfig struct {
	Id string `json:"id"`
}

func (a *GlobalMutexLock) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := ParseConfig[GlobalMutexLockConfig](cfg)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to parse config", zap.Error(err))
		return err
	}

	var lock *sync.Mutex

	GLOBAL_MUTEX.Lock()
	_, ok := GLOBAL_MUTEX_LOCKS[config.Id]
	if !ok {
		GLOBAL_MUTEX_LOCKS[config.Id] = &sync.Mutex{}
	}

	lock = GLOBAL_MUTEX_LOCKS[config.Id]
	GLOBAL_MUTEX.Unlock()

	lock.Lock()

	return nil
}

func (a *GlobalMutexLock) ParseConfig(data map[string]any) (any, error) {
	return ParseConfig[GlobalMutexLockConfig](data)
}

func init() {
	ACTIONS["GlobalMutexLock"] = &GlobalMutexLock{}
}
