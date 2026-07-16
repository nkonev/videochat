package tasks

import (
	"context"

	"github.com/nkonev/dcron"
	redisLock "github.com/nkonev/dcron/plugin/lock/redis"
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

func Scheduler(redisClient *redisV9.Client, lgr *logger.Logger) (*dcron.Cron, error) {
	ad := &TracingLoggerAdapter{lgr}
	return dcron.NewCron(
		redisLock.WithLock(redisClient, redisLock.WithSLog(ad)),
		dcron.WithSLog(ad),
	), nil
}

type TracingLoggerAdapter struct {
	lgr *logger.Logger
}

func (la *TracingLoggerAdapter) ErrorContext(ctx context.Context, msg string, args ...any) {
	la.lgr.WithTracing(ctx).With(args...).Error(msg)
}

func (la *TracingLoggerAdapter) InfoContext(ctx context.Context, msg string, args ...any) {
	la.lgr.WithTracing(ctx).With(args...).Info(msg)
}
