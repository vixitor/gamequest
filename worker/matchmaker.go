package worker

import (
	"fmt"
	"gamequest/types"
	"log"
	"strconv"
	"time"

	"gamequest/util"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func MatchMaker() {
	ctx := util.ConfigureService()
	dsn := ctx.Value("mysqldsn").(string)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	redisClient := util.NewRedisClient(ctx)
	for {
		luaScript := `
			local result = redis.call('ZRANGE', KEYS[1], 0, 9, 'WITHSCORES')
        	redis.call('ZREMRANGEBYRANK', KEYS[1], 0, 9)
        	return result `
		time.Sleep(1 * time.Second)
		log.Printf("matchbook size %v\n", redisClient.ZCard(ctx, "matchbook").Val())
		for redisClient.ZCard(ctx, "matchbook").Val() >= 10 {
			result, err := redisClient.Eval(ctx, luaScript, []string{"matchbook"}).Result()
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
			err = db.Transaction(func(tx *gorm.DB) error {
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
					redisClient.ZAdd(ctx, "matchbook", &redis.Z{Score: float64(score), Member: matchId})
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
				if err := db.Create(&gameinfo).Error; err != nil {
					log.Printf("Failed to create game info: %v", err)
				}
				gameid := gameinfo.GameId
				for i := 0; i < len(matchids); i += 1 {
					matchId := matchids[i]
					game2match := types.Game2Match{GameId: gameid, MatchId: matchId}
					if err := db.Create(&game2match).Error; err != nil {
						log.Printf("Failed to create game2match record for match ID %d: %v", matchId, err)
					}
					log.Printf("Created game2match record for match ID %d and game ID %d", matchId, gameid)
				}
			}
		}
	}
}
