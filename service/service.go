package service

import (
	"context"
	"fmt"
	mysql "gamequest/mysql"
	redis "gamequest/redis"
	"gamequest/types"

	"log"

	"gorm.io/gorm"
)

type User = types.User

type Service struct {
	db    *gorm.DB
	redis *redis.Client
	ctx   context.Context
}

func configureService() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "mysqldsn", "root:root123@tcp(localhost:9000)/gamequest?parseTime=true&loc=Local")
	ctx = context.WithValue(ctx, "redisaddr", "localhost:6379")
	ctx = context.WithValue(ctx, "redispassword", "")
	ctx = context.WithValue(ctx, "redisdb", 0)
	return ctx
}
func CreateService() (*Service, error) {
	ctx := configureService()
	db, err := mysql.NewMySQLDB(ctx.Value("mysqldsn").(string))

	if err != nil {
		return nil, fmt.Errorf("failed to create MySQLDB: %w", err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		return nil, fmt.Errorf("failed to migrate user table: %w", err)
	}
	log.Println("Service created successfully")
	return &Service{db: db, ctx: ctx}, nil
}

func (s *Service) Register(username string, password string) error {
	user := User{Username: username, Password: password, Score: 1500}
	result := s.db.Create(&user)
	if result.Error != nil {
		return fmt.Errorf("failed to register user: %w", result.Error)
	}
	log.Printf("User %s registered successfully", username)
	return nil
}

func (s *Service) Login(username string, password string) (int, error) {
	var user User
	result := s.db.Where("username = ?", username).First(&user)
	if result.Error != nil {
		return -1, fmt.Errorf("failed to login: %w", result.Error)
	}
	if user.Password != password {
		return -1, fmt.Errorf("invalid password for user %s", username)
	}
	log.Printf("User %s logged in successfully", username)
	return user.Id, nil
}

func (s *Service) CreateMatchRequest(playerID int) error {
	return nil
}
