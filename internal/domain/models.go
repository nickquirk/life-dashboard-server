package domain

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email        string    `gorm:"unique;not null;size:255"`
	Picture      string    `gorm:"type:text"`
	AccessToken  string    `gorm:"type:text"`
	RefreshToken string    `gorm:"type:text"`
	TokenExpiry  time.Time `gorm:"type:datetime"`
}
