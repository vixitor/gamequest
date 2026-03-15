package types

type User struct {
	Id       int    `gorm:"primaryKey;autoIncrement"`
	Username string `gorm:"size:255;not null;uniqueIndex"`
	Password string `gorm:"size:255;not null"`
	Score    int    `gorm:"not null"`
}

type MatchRequest struct {
	PlayId    int    `gorm:"not null"`
	PlayScore int    `gorm:"not null"`
	MatchId   int    `gorm:"primaryKey;autoIncrement"`
	Status    string `gorm:"size:255;not null"`
}

type MatchHistory struct {
	Id      int `gorm:"primaryKey"`
	Score   int `gorm:"not null"`
	GameId  int `gorm:"not null"`
	MatchId int `gorm:"not null"`
}

type GameInfo struct {
	GameId   int `gorm:"primaryKey;autoIncrement"`
	MaxScore int `gorm:"not null"`
	MinScore int `gorm:"not null"`
}

type Game2Match struct {
	GameId  int `gorm:"not null"`
	MatchId int `gorm:"primaryKey;"`
}

type OutboxEvent struct {
	MatchId int    `gorm:"primaryKey"`
	Statue  string `gorm:"size:255;not null"`
}
