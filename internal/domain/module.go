package domain

import "time"

type Module struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    string    `gorm:"type:uuid;not null;index"`
	User      User      `gorm:"foreignKey:UserID"`
	Title     string    `gorm:"type:varchar(255);not null"`
	FileURL   string    `gorm:"type:varchar(500);not null"`
	RawText   string    `gorm:"type:text"`
	Summary   string    `gorm:"type:text"`
	IsSummarized bool   `gorm:"default:false"`
	SummarizeFailed bool `gorm:"default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
