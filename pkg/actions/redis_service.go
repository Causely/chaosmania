package actions

import (
	"context"

	"github.com/Causely/chaosmania/pkg"
	redis8 "github.com/go-redis/redis/v8"
	"github.com/redis/go-redis/extra/redisotel/v9"
	redis9 "github.com/redis/go-redis/v9"
	redistrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/go-redis/redis.v8"
)

type RedisService struct {
	name   ServiceName
	config RedisServiceConfig
	rdb8   redis8.UniversalClient
	rdb9   *redis9.Client
}

type RedisServiceConfig struct {
	Address       string `json:"address"`
	PeerService   string `json:"peer_service"`
	PeerNamespace string `json:"peer_namespace"`
}

func (redis *RedisService) Name() ServiceName {
	return redis.name
}

func (redis *RedisService) Type() ServiceType {
	return "redis"
}

func NewRedisService(name ServiceName, config map[string]any) (Service, error) {
	cfg, err := pkg.ParseConfig[RedisServiceConfig](config)
	if err != nil {
		return nil, err
	}

	redisService := RedisService{
		config: *cfg,
		name:   name,
	}

	if pkg.IsDatadogEnabled() {
		opts := &redis8.Options{Addr: cfg.Address, DB: 0}
		rdb := redistrace.NewClient(opts, redistrace.WithServiceName(cfg.PeerService))

		if err != nil && err != redis8.Nil {
			return nil, err
		}

		redisService.rdb8 = rdb
	} else {
		opts := &redis9.Options{
			Addr: cfg.Address,
			DB:   0,
		}
		rdb := redis9.NewClient(opts)

		// Enable opentelemetry traces for redis
		err := redisotel.InstrumentMetrics(rdb)
		if err != nil {
			return nil, err
		}

		err = redisotel.InstrumentTracing(rdb)
		if err != nil {
			return nil, err
		}

		if err != nil && err != redis9.Nil {
			return nil, err
		}

		redisService.rdb9 = rdb
	}

	return &redisService, nil
}

// Redis Set
func (redis *RedisService) Set(ctx context.Context, key string, value string) error {
	if redis.rdb8 != nil {
		return redis.rdb8.Set(ctx, key, value, 0).Err()
	}

	return redis.rdb9.Set(ctx, key, value, 0).Err()
}

// Redis Get
func (redis *RedisService) Get(ctx context.Context, key string) (string, error) {
	if redis.rdb8 != nil {
		return redis.rdb8.Get(ctx, key).Result()
	}

	return redis.rdb9.Get(ctx, key).Result()
}

func init() {
	SERVICE_TYPES["redis"] = func(name ServiceName, m map[string]any) Service {
		s, err := NewRedisService(name, m)
		if err != nil {
			panic(err)
		}

		return s
	}
}
