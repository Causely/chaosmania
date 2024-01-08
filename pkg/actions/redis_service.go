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
	Address string `json:"address"`
}

func (s *RedisService) Name() ServiceName {
	return s.name
}

func (s *RedisService) Type() ServiceType {
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
		rdb := redistrace.NewClient(opts)

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
func (s *RedisService) Set(ctx context.Context, key string, value string) error {
	if s.rdb8 != nil {
		return s.rdb8.Set(ctx, key, value, 0).Err()
	}

	return s.rdb9.Set(ctx, key, value, 0).Err()
}

// Redis Get
func (s *RedisService) Get(ctx context.Context, key string) (string, error) {
	if s.rdb8 != nil {
		return s.rdb8.Get(ctx, key).Result()
	}

	return s.rdb9.Get(ctx, key).Result()
}

func init() {
	SERVICE_TPES["redis"] = func(name ServiceName, m map[string]any) Service {
		s, err := NewRedisService(name, m)
		if err != nil {
			panic(err)
		}

		return s
	}
}
