package actions

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Causely/chaosmania/pkg/logger"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/semconv/v1.21.0"
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

	// Enable opentelemetry for redis
	err := redisotel.InstrumentMetrics(rdb)
	if err != nil {
		logger.Error("failed to enable redis opentelemetry metrics", zap.Error(err))
	}
	err = redisotel.InstrumentTracing(rdb,
		redisotel.WithAttributes(
			semconv.NetPeerName("redis"),
			semconv.NetPeerPort(6379),
		))
	if err != nil {
		logger.Error("failed to enable redis opentelemetry tracing", zap.Error(err))
	}
	return rdb
}

func (action *RedisCommand) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := ParseConfig[RedisCommandConfig](cfg)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to parse config", zap.Error(err))
		return err
	}

	rdb := InitRedis(logger.FromContext(ctx), os.Getenv("DEPLOYMENT_NAME"))
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
		logger.FromContext(ctx).Warn("failed to execute command", zap.Error(err))
	}

	return err
}

func (action *RedisCommand) ParseConfig(data map[string]any) (any, error) {
	return ParseConfig[RedisCommandConfig](data)
}

func init() {
	ACTIONS["RedisCommand"] = &RedisCommand{}
}
