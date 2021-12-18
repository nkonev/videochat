package redis

import (
	"fmt"
	"github.com/ehsaniara/gointerlock"
	redisV8 "github.com/go-redis/redis/v8"
	"time"
)

func myJob() {
	fmt.Println(time.Now(), " - called")
}

func CleanDeletedImagesFromMessageBody(redisConnector *redisV8.Client) *gointerlock.GoInterval {
	return &gointerlock.GoInterval{
		Name:           "MyTestJob",
		Interval:       2 * time.Second,
		Arg:            myJob,
		RedisConnector: redisConnector,
	}
}
