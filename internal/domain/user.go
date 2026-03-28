package domain

import (
	"time"
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleGuest  Role = "guest"
)

type User struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string    `gorm:"type:varchar(100);not null"`
	Email     string    `gorm:"type:varchar(150);uniqueIndex;not null"`
	Password  string    `gorm:"type:varchar(255);not null"`
	Role      Role      `gorm:"type:varchar(20);not null;default:'guest'"`
	QuizQuota      int  `gorm:"not null;default:1"`
	SummarizeQuota int  `gorm:"not null;default:1"`
	OTP       string    `gorm:"type:varchar(6)"`
	IsVerified bool     `gorm:"default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
