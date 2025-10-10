package tasks

import (
	"context"
	"github.com/nkonev/dcron"
	redisV9 "github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"nkonev.name/chat/config"
	"nkonev.name/chat/logger"
	"time"
)

func RedisV9(lc fx.Lifecycle, lgr *logger.LoggerWrapper, cfg *config.AppConfig) *redisV9.Client {
	rv8 := redisV9.NewClient(&redisV9.Options{
		Addr:       cfg.Redis.Address,
		Password:   cfg.Redis.Password,
		DB:         cfg.Redis.Db,
		MaxRetries: cfg.Redis.MaxRetries,
	})
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			lgr.Info("Stopping redis scheduling connection")
			return rv8.Close()
		},
	})
	return rv8
}

type RedisLock struct {
	client *redisV9.Client
	lgr    *logger.LoggerWrapper
	cfg    *config.TaskConfig
}

func (m *RedisLock) Lock(ctx context.Context, jobSettings any, key, value string) bool {
	exp := jobSettings.(time.Duration)
	if exp == 0 {
		m.lgr.ErrorContext(ctx, "bad zero expiration", dcron.SlogKeyTaskName, key)
		return false
	}

	locked, err := m.client.SetNX(ctx, key, value, exp).Result()
	if err != nil {
		m.lgr.ErrorContext(ctx, "unable to invoke redis", logger.AttributeError, err)
		return false
	}

	return locked
}

func (m *RedisLock) Unlock(ctx context.Context, jobSetting any, key, value string) {
	m.client.Del(ctx, key)
}

func RedisLocker(redisClient *redisV9.Client, lgr *logger.LoggerWrapper, cfg *config.AppConfig) (*RedisLock, error) {
	return &RedisLock{client: redisClient, lgr: lgr, cfg: &cfg.Schedulers}, nil
}

func Scheduler(locker *RedisLock, lgr *logger.LoggerWrapper) (*dcron.Cron, error) {
	return dcron.NewCron(dcron.WithLock(locker), dcron.WithSLog(lgr)), nil
}
