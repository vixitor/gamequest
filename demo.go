package main

import Service "gamequest/service"

func main() {
	service, err := Service.CreateService()
	if err != nil {
		panic(err)
	}
	id, err := service.Login("testuser1", "testpassword1")
	if err != nil {
		panic(err)
	}
	println("Logged in user ID:", id)
	matchid, err := service.GetMatchId()
	if err != nil {
		panic(err)
	}
	service.CreateMatchRequest(id, matchid)
	println("Created match request with match ID:", matchid)
}
