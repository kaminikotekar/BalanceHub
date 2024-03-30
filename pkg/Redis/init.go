package Redis

import (
	"context"
	"github.com/kaminikotekar/BalanceHub/pkg/Config"
	"github.com/redis/go-redis/v9"
	"os"
)

var (
	Initialized  = false
	RedisEnabled bool
	ctx          context.Context
	client       *redis.Client
	redisConfig  Config.RedisServer
)

func InitServer() {
	RedisEnabled = false
	redisConfig = Config.Configuration.GetRedisConfig()
	if redisConfig.Ip != "" && redisConfig.Port != "" {
		RedisEnabled = true
	}

	ctx = GetContext()
	client = GetRDClient()
	Initialized = true
}

func GetContext() context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return ctx
}

func GetRDClient() *redis.Client {

	if client == nil {
		client = redis.NewClient(&redis.Options{
			Addr:     redisConfig.Ip + ":" + redisConfig.Port,
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       redisConfig.Dbindex,
		})
	}
	return client
}

func IsCacheAllowed() bool {
	return redisConfig.Caching
}

func CacheDuration() int {
	return redisConfig.CacheDuration
}
