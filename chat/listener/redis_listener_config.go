package listener

import (
	"context"
	"github.com/gomodule/redigo/redis"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	. "nkonev.name/chat/logger"
	"time"
)
/*
func newPool(s *shard, n *Node, conf RedisShardConfig) (redisConnPool, error) {
	host := conf.Host
	port := conf.Port
	password := conf.Password
	db := conf.DB

	useSentinel := conf.MasterName != "" && len(conf.SentinelAddrs) > 0
	usingPassword := password != ""

	poolFactory := makePoolFactory(s, n, conf)

	if !s.useCluster() {
		serverAddr := net.JoinHostPort(host, strconv.Itoa(port))
		if !useSentinel {
			n.Log(NewLogEntry(LogLevelInfo, fmt.Sprintf("Redis: %s/%d, using password: %v", serverAddr, db, usingPassword)))
		} else {
			n.Log(NewLogEntry(LogLevelInfo, fmt.Sprintf("Redis: Sentinel for name: %s, db: %d, using password: %v", conf.MasterName, db, usingPassword)))
		}
		pool, _ := poolFactory(serverAddr, getDialOpts(conf)...)
		return pool, nil
	}
	// OK, we should work with cluster.
	n.Log(NewLogEntry(LogLevelInfo, fmt.Sprintf("Redis: cluster addrs: %+v, using password: %v", conf.ClusterAddrs, usingPassword)))
	cluster := &redisc.Cluster{
		DialOptions:  getDialOpts(conf),
		StartupNodes: conf.ClusterAddrs,
		CreatePool:   poolFactory,
	}
	// Initialize cluster mapping.
	if err := cluster.Refresh(); err != nil {
		return nil, err
	}
	return cluster, nil
}

func makePoolFactory(s *shard, n *Node, conf RedisShardConfig) func(addr string, options ...redis.DialOption) (*redis.Pool, error) {
	password := conf.Password
	db := conf.DB

	useSentinel := conf.MasterName != "" && len(conf.SentinelAddrs) > 0

	var lastMu sync.Mutex
	var lastMaster string

	poolSize := defaultPoolSize
	maxIdle := poolSize

	var sntnl *sentinel.Sentinel
	if useSentinel {
		sntnl = &sentinel.Sentinel{
			Addrs:      conf.SentinelAddrs,
			MasterName: conf.MasterName,
			Dial: func(addr string) (redis.Conn, error) {
				timeout := 300 * time.Millisecond
				opts := []redis.DialOption{
					redis.DialConnectTimeout(timeout),
					redis.DialReadTimeout(timeout),
					redis.DialWriteTimeout(timeout),
				}
				c, err := redis.Dial("tcp", addr, opts...)
				if err != nil {
					n.Log(NewLogEntry(LogLevelError, "error dialing to Sentinel", map[string]interface{}{"error": err.Error()}))
					return nil, err
				}
				return c, nil
			},
		}

		// Periodically discover new Sentinels.
		go func() {
			if err := sntnl.Discover(); err != nil {
				n.Log(NewLogEntry(LogLevelError, "error discover Sentinel", map[string]interface{}{"error": err.Error()}))
			}
			for {
				<-time.After(30 * time.Second)
				if err := sntnl.Discover(); err != nil {
					n.Log(NewLogEntry(LogLevelError, "error discover Sentinel", map[string]interface{}{"error": err.Error()}))
				}
			}
		}()
	}

	return func(serverAddr string, dialOpts ...redis.DialOption) (*redis.Pool, error) {
		pool := &redis.Pool{
			MaxIdle:     maxIdle,
			MaxActive:   poolSize,
			Wait:        true,
			IdleTimeout: conf.IdleTimeout,
			Dial: func() (redis.Conn, error) {
				var err error
				if useSentinel {
					serverAddr, err = sntnl.MasterAddr()
					if err != nil {
						return nil, err
					}
					lastMu.Lock()
					if serverAddr != lastMaster {
						n.Log(NewLogEntry(LogLevelInfo, "Redis master discovered", map[string]interface{}{"addr": serverAddr}))
						lastMaster = serverAddr
					}
					lastMu.Unlock()
				}

				c, err := redis.Dial("tcp", serverAddr, dialOpts...)
				if err != nil {
					n.Log(NewLogEntry(LogLevelError, "error dialing to Redis", map[string]interface{}{"error": err.Error()}))
					return nil, err
				}

				if password != "" {
					if _, err := c.Do("AUTH", password); err != nil {
						_ = c.Close()
						n.Log(NewLogEntry(LogLevelError, "error auth in Redis", map[string]interface{}{"error": err.Error()}))
						return nil, err
					}
				}

				if db != 0 {
					if _, err := c.Do("SELECT", db); err != nil {
						_ = c.Close()
						n.Log(NewLogEntry(LogLevelError, "error selecting Redis db", map[string]interface{}{"error": err.Error()}))
						return nil, err
					}
				}
				return c, nil
			},
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				if useSentinel {
					if !sentinel.TestRole(c, "master") {
						return errors.New("failed master role check")
					}
					return nil
				}
				if s.useCluster() {
					// No need in this optimization outside cluster
					// use case due to utilization of pipelining.
					if time.Since(t) < time.Second {
						return nil
					}
				}
				_, err := c.Do("PING")
				return err
			},
		}
		return pool, nil
	}
}
*/












func RedisAaaConnection(lc fx.Lifecycle) (*redis.Pool, error) {
	Logger.Infof("Starting redis aaa connection")

	address := viper.GetString("aaa.redis.address")
	password := viper.GetString("aaa.redis.password")

	readDuration := viper.GetDuration("aaa.redis.readTimeout")
	writeDuration := viper.GetDuration("aaa.redis.writeTimeout")
	connectTimeout := viper.GetDuration("aaa.redis.connectTimeout")
	idleTimeout := viper.GetDuration("aaa.redis.idleTimeout")
	dbase := viper.GetInt("aaa.redis.db")
	maxIdle := viper.GetInt("aaa.redis.maxIdle")
	maxActive := viper.GetInt("aaa.redis.maxActive")

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



// https://pkg.go.dev/github.com/gomodule/redigo/redis#PubSubConn
func ListenPubSubChannels(
	pool *redis.Pool,
	onMessage AaaUserProfileUpdateListener,
	lc fx.Lifecycle) {

	go func() {
		lifecycleHookAppended := false

		var psc redis.PubSubConn
		for {
			conn := pool.Get()
			defer conn.Close()

			Logger.Infof("Starting redis aaa subscription")
			var channels []string = []string{"user.profile.update"}

			psc = redis.PubSubConn{Conn: conn}

			if err := psc.Subscribe(redis.Args{}.AddFlat(channels)...); err != nil {
				Logger.Errorf("Error on aaa subscription %v", err)
				sleepBetweenRetry()
				continue
			}

			if !lifecycleHookAppended {
				// Signal the receiving goroutine to exit by unsubscribing from all channels.
				lc.Append(fx.Hook{
					OnStop: func(ctx context.Context) error {
						Logger.Infof("Removing redis aaa subscription")
						return psc.Unsubscribe()
					},
				})
				lifecycleHookAppended = true
			}

			done := make(chan error, 1)

			// Start a goroutine to receive notifications from the server.
			go func() {
				for {
					switch n := psc.Receive().(type) {
					case error:
						done <- n
						return
					case redis.Message:
						if err := onMessage(n.Channel, n.Data); err != nil {
							done <- err
							return
						}
					case redis.Subscription:
						switch n.Count {
						case len(channels):
							// Notify application when all channels are subscribed.
							Logger.Infof("app subscribed to the all channels")
						case 0:
							// Return from the goroutine when all channels are unsubscribed.
							done <- nil
							return
						}
					}
				}
			}()

			// Wait for goroutine to complete.
			err := <-done
			Logger.Errorf("Error on redis aaa subscription %v", err)

			err = psc.Unsubscribe()
			Logger.Infof("Unsubscribing, error=%v", err)

			sleepBetweenRetry()
		}
	}()
}

func sleepBetweenRetry() {
	const sleepSec = 1
	Logger.Infof("Sleep %v sec", sleepSec)
	time.Sleep(sleepSec * time.Second)
}
