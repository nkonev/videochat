package redis

import (
	"context"
	"github.com/gomodule/redigo/redis"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	. "nkonev.name/chat/logger"
	"time"
)

func RedisPooledConnection(lc fx.Lifecycle) (*redis.Pool, error) {
	Logger.Infof("Starting redis pooled connection")

	address := viper.GetString("redis.address")
	password := viper.GetString("redis.password")

	readDuration := viper.GetDuration("redis.readTimeout")
	writeDuration := viper.GetDuration("redis.writeTimeout")
	connectTimeout := viper.GetDuration("redis.connectTimeout")
	idleTimeout := viper.GetDuration("redis.idleTimeout")
	dbase := viper.GetInt("redis.db")
	maxIdle := viper.GetInt("redis.maxIdle")
	maxActive := viper.GetInt("redis.maxActive")

	pool := &redis.Pool{
		MaxIdle:     maxIdle,
		MaxActive:   maxActive,
		Wait:        true,
		IdleTimeout: idleTimeout,
		Dial: func() (redis.Conn, error) {
			var err error

			c, err := redis.Dial("tcp", address,
				redis.DialReadTimeout(readDuration), // default 0 means infinity read
				redis.DialWriteTimeout(writeDuration),
				redis.DialConnectTimeout(connectTimeout),
				redis.DialDatabase(dbase),
				redis.DialPassword(password),
			)
			if err != nil {
				Logger.Errorf("error dialing to Redis %v", err.Error())
				return nil, err
			}
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			Logger.Infof("Stopping redis aaa connection")
			return pool.Close()
		},
	})
	return pool, nil
}
