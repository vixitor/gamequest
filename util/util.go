package util

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func ConfigureService() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "mysqldsn", "root:root123@tcp(localhost:9000)/gamequest?parseTime=true&loc=Local")
	ctx = context.WithValue(ctx, "redisaddr", "localhost:6379")
	ctx = context.WithValue(ctx, "redispassword", "")
	ctx = context.WithValue(ctx, "redisdb", 0)
	return ctx
}

func NewMySQLDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Println("Failed to connect to the database:", err)
		return nil, err
	}
	log.Println("Connected to the database successfully")
	return db, nil
}

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
