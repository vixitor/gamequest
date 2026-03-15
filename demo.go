package main

import (
	"fmt"
	Service "gamequest/service"
	"log"
	"sync"
	"time"
)

func main() {
	service, err := Service.CreateService()
	if err != nil {
		panic(err)
	}
	var wg sync.WaitGroup
	start_time := time.Now()
	for i := range 100 {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			name := fmt.Sprintf("tests%d", i)
			password := fmt.Sprintf("testpasswords%d", i)
			service.Register(name, password)
			id, err := service.Login(name, password)
			if err != nil {
				panic(err)
			}
			println("Logged in user ID:", id)
			matchid, err := service.GetMatchId()
			log.Printf("matchid %v\n", matchid)
			if err != nil {
				panic(err)
			}
			service.CreateMatchRequest(id, matchid)
			status := ""
			for {
				status = service.GetMatchRequestStatus(matchid)
				if status != "pending" {
					println("Match request status changed to:", status)
					break
				}
				time.Sleep(1 * time.Second)
			}
			gameid := service.GetGameId(matchid)
			println("Created match request with match ID:", matchid)
			println("Match request status:", status)
			println("Associated game ID:", gameid)
		}(i)
	}
	wg.Wait()
	end_time := time.Now()
	log.Printf("Total execution time: %v\n", end_time.Sub(start_time))
}
