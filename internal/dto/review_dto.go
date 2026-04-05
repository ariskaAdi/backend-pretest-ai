package dto

import (
	"time"
)

// CreateReviewRequest — body payload from the client
type CreateReviewRequest struct {
	Position string `json:"position" validate:"required"`
	Review   string `json:"review"   validate:"required"`
	Rating   int    `json:"rating"   validate:"required,min=1,max=5"`
}

// ReviewResponse — returned to the client
type ReviewResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Position  string    `json:"position"`
	Review    string    `json:"review"`
	Rating    int       `json:"rating"`
	CreatedAt time.Time `json:"created_at"`
}
