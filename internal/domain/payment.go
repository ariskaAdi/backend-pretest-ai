package domain

import "time"

type LynkTransaction struct {
	ID            string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TransactionID string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	Email         string    `gorm:"type:varchar(150);not null"`
	ProductName   string    `gorm:"type:varchar(255);not null"`
	Amount        int       `gorm:"not null"`
	Status        string    `gorm:"type:varchar(50);not null"`
	CreatedAt     time.Time
}
