package main

import Service "gamequest/service"

func main() {
	service, err := Service.CreateService()
	if err != nil {
		panic(err)
	}
	if err := service.Register("testuser1", "testpassword1"); err != nil {
		panic(err)
	}
	id, err := service.Login("testuser1", "testpassword1")
	if err != nil {
		panic(err)
	}
	println("Logged in user ID:", id)
}
