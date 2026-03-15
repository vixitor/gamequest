package worker

import (
	"context"
	"fmt"
	"gamequest/types"
	"log"

	"gamequest/util"
	redis "github.com/go-redis/redis/v8"
	kafka "github.com/segmentio/kafka-go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Consumer() {
	ctx := util.ConfigureService()
	dsn := ctx.Value("mysqldsn").(string)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	redisClient := util.NewRedisClient(ctx)
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
		db.Where("match_id = ?", matchid).First(&matchrequest)
		redisClient.ZAdd(ctx, "matchbook", &redis.Z{Score: float64(matchrequest.PlayScore), Member: matchrequest.MatchId})
	}
}
