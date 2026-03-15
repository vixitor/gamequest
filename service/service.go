package service

import (
	"context"
	"fmt"
	"gamequest/types"
	"log"
	"time"

	"gamequest/util"
	redis "github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

func (s *Service) GetMatchId() (int, error) {
	matchId, err := s.redis.Incr(s.ctx, "match_id").Result()
	if err != nil {
		return -1, fmt.Errorf("failed to get match ID: %w", err)
	}
	return int(matchId), nil
}

func (s *Service) GetUserById(userId int) (*types.User, error) {
	var user types.User
	result := s.db.First(&user, userId)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", result.Error)
	}
	return &user, nil
}

type Service struct {
	db    *gorm.DB
	redis *redis.Client
	ctx   context.Context
}

func CreateService() (*Service, error) {

	ctx := util.ConfigureService()
	db, err := util.NewMySQLDB(ctx.Value("mysqldsn").(string))
	redisClient := util.NewRedisClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create MySQLDB: %w", err)
	}
	if err := db.AutoMigrate(&types.User{}, &types.MatchRequest{}, &types.MatchHistory{}, &types.GameInfo{}, &types.Game2Match{}, &types.OutboxEvent{}); err != nil {
		return nil, fmt.Errorf("failed to migrate tables: %w", err)
	}
	log.Println("Service created successfully")
	service := &Service{db: db, ctx: ctx, redis: redisClient}
	return service, nil
}

func (s *Service) Register(username string, password string) error {
	user := types.User{Username: username, Password: password, Score: 1500}
	result := s.db.Create(&user)
	if result.Error != nil {
		return fmt.Errorf("failed to register user: %w", result.Error)
	}
	log.Printf("User %s registered successfully", username)
	return nil
}

func (s *Service) Login(username string, password string) (int, error) {
	var user types.User
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

func (s *Service) CreateMatchRequest(playerID int, matchId int) (int, error) {
	user, err := s.GetUserById(playerID)
	if err != nil {
		return -1, fmt.Errorf("failed to get user by ID: %w", err)
	}
	score := user.Score
	matchRequest := types.MatchRequest{
		PlayId:    playerID,
		MatchId:   matchId,
		PlayScore: score,
		Status:    "pending",
	}
	outboxEvent := types.OutboxEvent{
		MatchId: matchId,
		Statue:  "pending",
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&matchRequest).Error; err != nil {
			return fmt.Errorf("failed to create match request: %w", err)
		}
		if err := tx.Create(&outboxEvent).Error; err != nil {
			return fmt.Errorf("failed to create outbox event: %w", err)
		}
		return nil
	})
	if err != nil {
		return -1, fmt.Errorf("failed to create match request: %w", err)
	}
	return matchId, nil
}

func (s *Service) GetMatchRequestStatus(matchId int) string {
	var matchRequest types.MatchRequest
	key := fmt.Sprintf("match_request_status:%d", matchId)
	err := s.redis.Exists(s.ctx, key).Err()
	if err == nil {
		status, err := s.redis.Get(s.ctx, key).Result()
		if err == nil {
			return status
		}
	}
	result := s.db.Where("match_id = ?", matchId).First(&matchRequest)
	if result.Error != nil {
		log.Printf("Failed to get match request status: %v", result.Error)
		return "pending"
	}
	s.redis.Set(s.ctx, key, matchRequest.Status, 3*time.Second)
	return matchRequest.Status
}

func (s *Service) GetGameId(matchId int) int {
	var game2match types.Game2Match
	result := s.db.Where("match_id = ?", matchId).First(&game2match)
	if result.Error != nil {
		log.Printf("Failed to get game ID for match ID %d: %v", matchId, result.Error)
		return -1
	}
	return game2match.GameId
}
