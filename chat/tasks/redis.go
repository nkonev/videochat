package tasks

import (
	"context"

	"github.com/nkonev/dcron"
	redisLock "github.com/nkonev/dcron/plugin/lock/redis"
	redisV9 "github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"nkonev.name/chat/config"
	"nkonev.name/chat/logger"
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

func Scheduler(redisClient *redisV9.Client, lgr *logger.LoggerWrapper) (*dcron.Cron, error) {
	return dcron.NewCron(
		redisLock.WithLock(redisClient, redisLock.WithSLog(lgr)),
		dcron.WithSLog(lgr),
	), nil
}
