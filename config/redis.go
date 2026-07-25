package config

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

func LoadRedisConfig() RedisConfig {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}

	password := os.Getenv("REDIS_PASSWORD")

	db := 0
	if v := os.Getenv("REDIS_DB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			db = n
		}
	}

	return RedisConfig{
		Host:     host,
		Port:     port,
		Password: password,
		DB:       db,
	}
}

func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func (c RedisConfig) GoRedisOptions() *redis.Options {
	return &redis.Options{
		Addr:     c.Addr(),
		Password: c.Password,
		DB:       c.DB,
	}
}

func InitRedis(ctx context.Context) error {
	cfg := LoadRedisConfig()

	rdb := redis.NewClient(cfg.GoRedisOptions())

	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		return err
	}

	fmt.Println("Conexión Redis OK:", pong)
	Rdb = rdb
	return nil
}
