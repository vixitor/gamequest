package myredis

import (
	"context"
	redis "github.com/go-redis/redis/v8"
)

func NewRedisClient(ctx context.Context) *redis.Client {
	addr := ctx.Value("redisaddr").(string)
	password := ctx.Value("redispassword").(string)
	db := ctx.Value("redisdb").(int)

	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

func (c *redis.Client) GetMatchId(ctx context.Context) (int, error) {
	return c.Get(ctx, "match_id").Int()
}
