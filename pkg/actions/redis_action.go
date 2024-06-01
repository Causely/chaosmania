package actions

import (
	"context"
	"fmt"
	"strings"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	redis8 "github.com/go-redis/redis/v8"
	"github.com/redis/go-redis/extra/redisotel/v9"
	redis9 "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	redistrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/go-redis/redis.v8"
)

type RedisCommand struct{}

type RedisCommandConfig struct {
	Address       string `json:"address"`
	Command       string `json:"command"`
	Args          []any  `json:"args"`
	PeerService   string `json:"peer_service"`
	PeerNamespace string `json:"peer_namespace"`
}

func (redis *RedisCommand) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := pkg.ParseConfig[RedisCommandConfig](cfg)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to parse config", zap.Error(err))
		return err
	}

	if pkg.IsDatadogEnabled() {
		opts := &redis8.Options{Addr: config.Address, DB: 0}
		rdb := redistrace.NewClient(opts, redistrace.WithServiceName(config.PeerService))

		switch strings.ToLower(config.Command) {
		case "lpop":
			err = rdb.LPop(ctx, config.Args[0].(string)).Err()
		case "lpush":
			err = rdb.LPush(ctx, config.Args[0].(string), config.Args[:1]...).Err()
		case "get":
			err = rdb.Get(ctx, config.Args[0].(string)).Err()
		case "set":
			err = rdb.Set(ctx, config.Args[0].(string), config.Args[1].(string), 0).Err()
		default:
			return fmt.Errorf("redis command not supported: %s", config.Command)
		}

		if err != nil && err != redis8.Nil {
			return err
		}
	} else {
		opts := &redis9.Options{
			Addr: config.Address,
			DB:   0,
		}
		rdb := redis9.NewClient(opts)

		// Enable opentelemetry for redis
		err := redisotel.InstrumentMetrics(rdb)
		if err != nil {
			logger.FromContext(ctx).Error("failed to enable redis opentelemetry metrics", zap.Error(err))
		}
		err = redisotel.InstrumentTracing(rdb)
		if err != nil {
			logger.FromContext(ctx).Error("failed to enable redis opentelemetry tracing", zap.Error(err))
		}
		defer rdb.Close()

		switch strings.ToLower(config.Command) {
		case "lpop":
			err = rdb.LPop(ctx, config.Args[0].(string)).Err()
		case "lpush":
			err = rdb.LPush(ctx, config.Args[0].(string), config.Args[:1]...).Err()
		case "get":
			err = rdb.Get(ctx, config.Args[0].(string)).Err()
		case "set":
			err = rdb.Set(ctx, config.Args[0].(string), config.Args[1].(string), 0).Err()
		default:
			return fmt.Errorf("redis command not supported: %s", config.Command)
		}

		if err != nil && err != redis9.Nil {
			return err
		}
	}

	return nil
}

func (redis *RedisCommand) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[RedisCommandConfig](data)
}

func init() {
	ACTIONS["RedisCommand"] = &RedisCommand{}
}
