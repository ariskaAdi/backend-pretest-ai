package domain

import "time"

type QuizStatus string

const (
	QuizStatusPending   QuizStatus = "pending"
	QuizStatusCompleted QuizStatus = "completed"
	QuizStatusCancelled QuizStatus = "cancelled"
)

type Quiz struct {
	ID        string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    string     `gorm:"type:uuid;not null;index"`
	ModuleID  string     `gorm:"type:uuid;not null;index"`
	Module    Module     `gorm:"foreignKey:ModuleID"`
	NumQuestions int     `gorm:"not null"`
	Score     *int       `gorm:"default:null"`
	Status    QuizStatus `gorm:"type:varchar(20);not null;default:'pending'"`
	Questions []Question `gorm:"foreignKey:QuizID"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Question struct {
	ID            string   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	QuizID        string   `gorm:"type:uuid;not null;index"`
	Text          string   `gorm:"type:text;not null"`
	Options       string   `gorm:"type:jsonb;not null"` // JSON array ["A. ...", "B. ...", "C. ...", "D. ..."]
	CorrectAnswer string   `gorm:"type:varchar(1);not null"` // "A" | "B" | "C" | "D"
	UserAnswer    string   `gorm:"type:varchar(1)"` // diisi saat submit
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
