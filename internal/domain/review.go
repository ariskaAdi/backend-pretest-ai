package domain

import (
	"time"
)

type Review struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    string    `gorm:"type:uuid;not null"`
	Username  string    `gorm:"type:varchar(100);not null"`
	Position  string    `gorm:"type:varchar(100);not null"`
	Review    string    `gorm:"type:text;not null"`
	Rating    int       `gorm:"not null;check:rating >= 1 AND rating <= 5"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
