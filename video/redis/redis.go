package redis

import (
	"context"
	redisV8 "github.com/go-redis/redis/v8"
	"go.uber.org/fx"
	"nkonev.name/video/config"
	. "nkonev.name/video/logger"
)

func RedisV8(lc fx.Lifecycle, conf *config.ExtendedConfig) *redisV8.Client {
	rv8 := redisV8.NewClient(&redisV8.Options{
		Addr:       conf.RedisConfig.Address,
		Password:   conf.RedisConfig.Password,
		DB:         conf.RedisConfig.Db,
		MaxRetries: conf.RedisConfig.MaxRetries,
	})
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			Logger.Infof("Stopping redis scheduling connection")
			return rv8.Close()
		},
	})
	return rv8
}
