package tasks

import (
	"context"
	"github.com/nkonev/dcron"
	redisV9 "github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"nkonev.name/storage/logger"
)

func RedisV9(lgr *logger.Logger, lc fx.Lifecycle) *redisV9.Client {
	rv8 := redisV9.NewClient(&redisV9.Options{
		Addr:       viper.GetString("redis.address"),
		Password:   viper.GetString("redis.password"),
		DB:         viper.GetInt("redis.db"),
		MaxRetries: viper.GetInt("redis.maxRetries"),
	})
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			lgr.Infof("Stopping redis scheduling connection")
			return rv8.Close()
		},
	})
	return rv8
}

type RedisLock struct {
	client *redisV9.Client
	lgr    *logger.Logger
}

func (m *RedisLock) Lock(ctx context.Context, key, value string) bool {
	exp := viper.GetDuration("schedulers." + key + ".expiration")
	if exp == 0 {
		m.lgr.WithTracing(ctx).Errorf("not set expiring duration")
		return false
	}

	locked, err := m.client.SetNX(ctx, key, value, exp).Result()
	if err != nil {
		m.lgr.WithTracing(ctx).Errorf("unable to invoke redis: %v", err)
		return false
	}

	return locked
}

func (m *RedisLock) Unlock(ctx context.Context, key, value string) {
	m.client.Del(ctx, key)
}

func RedisLocker(redisClient *redisV9.Client, lgr *logger.Logger) (*RedisLock, error) {
	return &RedisLock{client: redisClient, lgr: lgr}, nil
}

func Scheduler(locker *RedisLock) (*dcron.Cron, error) {
	return dcron.NewCron(dcron.WithLock(locker)), nil
}
