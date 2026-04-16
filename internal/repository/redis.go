package repository

import (
	"context"

	"github.com/go-redis/redis/v8"
)

type RedisRepository struct {
	Client *redis.Client
}

func NewRedisRepository(redisURL string) (*RedisRepository, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)
	err = client.Ping(context.Background()).Err()
	if err != nil {
		return nil, err
	}

	return &RedisRepository{Client: client}, nil
}
