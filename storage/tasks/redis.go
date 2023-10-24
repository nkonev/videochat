package tasks

import (
	"context"
	redisV8 "github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	. "nkonev.name/storage/logger"
)

func RedisV8(lc fx.Lifecycle) *redisV8.Client {
	rv8 := redisV8.NewClient(&redisV8.Options{
		Addr:       viper.GetString("redis.address"),
		Password:   viper.GetString("redis.password"),
		DB:         viper.GetInt("redis.db"),
		MaxRetries: viper.GetInt("redis.maxRetries"),
	})
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			Logger.Infof("Stopping redis scheduling connection")
			return rv8.Close()
		},
	})
	return rv8
}
