package service

import (
	"context"
	"fmt"
	"gamequest/types"
	"log"
	"strconv"
	"time"

	redis "github.com/go-redis/redis/v8"
	kafka "github.com/segmentio/kafka-go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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

func NewMySQLDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Println("Failed to connect to the database:", err)
		return nil, err
	}
	log.Println("Connected to the database successfully")
	return db, nil
}

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

func configureService() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "mysqldsn", "root:root123@tcp(localhost:9000)/gamequest?parseTime=true&loc=Local")
	ctx = context.WithValue(ctx, "redisaddr", "localhost:6379")
	ctx = context.WithValue(ctx, "redispassword", "")
	ctx = context.WithValue(ctx, "redisdb", 0)
	return ctx
}
func (s *Service) Producer() {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "match_requests",
	})
	for {
		time.Sleep(1 * time.Second)
		var events []types.OutboxEvent
		if err := s.db.Where("statue = ?", "pending").Find(&events).Error; err != nil {
			log.Printf("Failed to fetch pending outbox events: %v", err)
			continue
		}
		for _, event := range events {
			msg := kafka.Message{
				Value: []byte(fmt.Sprintf("Match ID %d is ready for processing", event.MatchId)),
			}
			if err := writer.WriteMessages(context.Background(), msg); err != nil {
				log.Printf("Failed to write message for match ID %d: %v", event.MatchId, err)
				continue
			}
			log.Printf("Processing outbox event for match ID: %d", event.MatchId)
			if err := s.db.Model(&types.OutboxEvent{}).Where("match_id = ?", event.MatchId).Update("statue", "processed").Error; err != nil {
				log.Printf("Failed to update outbox event status for match ID %d: %v", event.MatchId, err)
			}
		}
	}
}
func (s *Service) Consumer() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "match_requests",
		GroupID: "match_request_processors",
	})
	for {
		msg, err := reader.ReadMessage(context.Background())
		value := string(msg.Value)
		if err != nil {
			log.Printf("Failed to read message: %v", err)
			continue
		}
		log.Printf("Received message: %s", string(msg.Value))
		matchid := 0
		_, err = fmt.Sscanf(value, "Match ID %d is ready for processing", &matchid)
		if err != nil {
			log.Printf("Failed to parse match ID from message: %v", err)
			continue
		}
		log.Printf("Processing match request for match ID: %d", matchid)
		var matchrequest types.MatchRequest
		s.db.Where("match_id = ?", matchid).First(&matchrequest)
		s.redis.ZAdd(s.ctx, "matchbook", &redis.Z{Score: float64(matchrequest.PlayScore), Member: matchrequest.MatchId})
	}
}
func (s *Service) MatchMaker() {
	for {
		luaScript := `
			local result = redis.call('ZRANGE', KEYS[1], 0, 9, 'WITHSCORES')
        	redis.call('ZREMRANGEBYRANK', KEYS[1], 0, 9)
        	return result `
		time.Sleep(50 * time.Millisecond)
		// log.Printf("matchbook size %v\n", s.redis.ZCard(s.ctx, "matchbook").Val())
		for s.redis.ZCard(s.ctx, "matchbook").Val() >= 10 {
			result, err := s.redis.Eval(s.ctx, luaScript, []string{"matchbook"}).Result()
			if err != nil {
				log.Printf("Failed to execute Lua script: %v", err)
				continue
			}
			var scores []int
			var matchids []int
			for i := 0; i < len(result.([]interface{})); i += 2 {
				matchId, _ := strconv.Atoi(result.([]interface{})[i].(string))
				score, _ := strconv.Atoi(result.([]interface{})[i+1].(string))
				matchids = append(matchids, matchId)
				scores = append(scores, score)
			}
			log.Printf("Matched matchids: %v\n", matchids)
			log.Printf("Matched Scores %v\n", scores)
			err = s.db.Transaction(func(tx *gorm.DB) error {
				for i := 0; i < len(matchids); i += 1 {
					matchId := matchids[i]
					if err := tx.Model(&types.MatchRequest{}).Where("match_id = ?", matchId).Update("status", "matched").Error; err != nil {
						return fmt.Errorf("failed to update match request status for match ID %v: %w", matchId, err)
					}
				}
				return nil
			})
			if err != nil {
				log.Printf("Failed to update match request statuses: %v", err)
				for i := 0; i < len(matchids); i += 1 {
					matchId := matchids[i]
					score := scores[i]
					s.redis.ZAdd(s.ctx, "matchbook", &redis.Z{Score: float64(score), Member: matchId})
				}
			} else {
				maxScore := 0
				minScore := 3000
				for i := 0; i < len(scores); i += 1 {
					if scores[i] > maxScore {
						maxScore = scores[i]
					}
					if scores[i] < minScore {
						minScore = scores[i]
					}
				}
				gameinfo := types.GameInfo{MaxScore: maxScore, MinScore: minScore}
				if err := s.db.Create(&gameinfo).Error; err != nil {
					log.Printf("Failed to create game info: %v", err)
				}
				gameid := gameinfo.GameId
				for i := 0; i < len(matchids); i += 1 {
					matchId := matchids[i]
					game2match := types.Game2Match{GameId: gameid, MatchId: matchId}
					if err := s.db.Create(&game2match).Error; err != nil {
						log.Printf("Failed to create game2match record for match ID %d: %v", matchId, err)
					}
					log.Printf("Created game2match record for match ID %d and game ID %d", matchId, gameid)
				}
			}
		}
	}
}

func CreateService() (*Service, error) {

	ctx := configureService()
	db, err := NewMySQLDB(ctx.Value("mysqldsn").(string))
	redisClient := NewRedisClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create MySQLDB: %w", err)
	}
	if err := db.AutoMigrate(&types.User{}, &types.MatchRequest{}, &types.MatchHistory{}, &types.GameInfo{}, &types.Game2Match{}, &types.OutboxEvent{}); err != nil {
		return nil, fmt.Errorf("failed to migrate tables: %w", err)
	}
	log.Println("Service created successfully")
	service := &Service{db: db, ctx: ctx, redis: redisClient}
	go service.Producer()
	go service.Consumer()
	go service.MatchMaker()
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
