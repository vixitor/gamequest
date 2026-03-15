package worker

import (
	"context"
	"fmt"
	"gamequest/types"
	"log"
	"time"

	"gamequest/util"
	kafka "github.com/segmentio/kafka-go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Producer() {
	ctx := util.ConfigureService()
	dsn := ctx.Value("mysqldsn").(string)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "match_requests",
	})
	for {
		time.Sleep(1 * time.Second)
		var events []types.OutboxEvent
		if err := db.Where("statue = ?", "pending").Find(&events).Error; err != nil {
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
			if err := db.Model(&types.OutboxEvent{}).Where("match_id = ?", event.MatchId).Update("statue", "processed").Error; err != nil {
				log.Printf("Failed to update outbox event status for match ID %d: %v", event.MatchId, err)
			}
		}
	}
}
