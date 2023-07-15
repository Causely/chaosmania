package actions

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Causely/chaosmania/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisCommand struct{}

type RedisCommandConfig struct {
	Command string `json:"command"`
	Args    []any  `json:"args"`
}

func InitRedis(logger *zap.Logger, application string) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
		DB:   0,
	})

	return rdb
}

func (a *RedisCommand) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := ParseConfig[RedisCommandConfig](cfg)
	if err != nil {
		logger.NewLogger().Warn("failed to parse config", zap.Error(err))
		return err
	}

	rdb := InitRedis(logger.NewLogger(), os.Getenv("DEPLOYMENT_NAME"))
	defer rdb.Close()

	switch strings.ToLower(config.Command) {
	case "lpop":
		err = rdb.LPop(ctx, config.Args[0].(string)).Err()
	case "lpush":
		err = rdb.LPush(ctx, config.Args[0].(string), config.Args[:1]...).Err()
	default:
		return fmt.Errorf("redis command not supported: %s", config.Command)
	}

	if err != nil {
		logger.NewLogger().Warn("failed to execute command", zap.Error(err))
	}

	return err
}

func (a *RedisCommand) ParseConfig(data map[string]any) (any, error) {
	return ParseConfig[RedisCommandConfig](data)
}

func init() {
	ACTIONS["RedisCommand"] = &RedisCommand{}
}
