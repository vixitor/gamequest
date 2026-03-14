package types

type User struct {
	Id       int    `gorm:"primaryKey;autoIncrement"`
	Username string `gorm:"size:255;not null;uniqueIndex"`
	Password string `gorm:"size:255;not null"`
	Score    int    `gorm:"not null"`
}

type MatchRequest struct {
	PlayId    int `gorm:"not null"`
	PlayScore int `gorm:"not null"`
	MatchId   int `gorm:"primaryKey;autoIncrement"`
}
