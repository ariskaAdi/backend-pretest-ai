# Issue: Add Review Feature

## Summary

Implement a review system that allows authenticated users to submit reviews including their position, review text, and a star rating (1–5). The username is derived from the authenticated user's session — not from user input.

---

## Database

### Migration: `009_create_reviews.sql`

Create a new `reviews` table:

```sql
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(100) NOT NULL,
    position VARCHAR(100) NOT NULL,
    review TEXT NOT NULL,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);
```

---

## Domain

### `internal/domain/review.go`

```go
type Review struct {
    ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    Username  string    `gorm:"type:varchar(100);not null"`
    Position  string    `gorm:"type:varchar(100);not null"`
    Review    string    `gorm:"type:text;not null"`
    Rating    int       `gorm:"not null;check:rating >= 1 AND rating <= 5"`
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

---

## DTO

### `internal/dto/review_dto.go`

```go
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
```

---

## Repository

### `internal/repository/review_repo.go`

Interface and GORM implementation:

```go
type ReviewRepository interface {
    Create(ctx context.Context, review *domain.Review) error
    GetAll(ctx context.Context) ([]domain.Review, error)
}
```

---

## Service

### `internal/service/review_service.go`

Business logic layer:

```go
type ReviewService interface {
    Create(ctx context.Context, username string, req dto.CreateReviewRequest) (*dto.ReviewResponse, error)
    GetAll(ctx context.Context) ([]dto.ReviewResponse, error)
}
```

- `username` is passed from the handler — extracted from JWT claims, never from request body.
- Validate `rating` is between 1 and 5.

---

## Handler

### `internal/handler/review_handler.go`

```go
type ReviewHandler struct {
    reviewService service.ReviewService
    validate      *validator.Validate
}
```

#### Endpoints

| Method | Path                | Auth     | Description            |
|--------|---------------------|----------|------------------------|
| POST   | `/api/v1/reviews`   | Required | Submit a new review    |
| GET    | `/api/v1/reviews`   | Public   | Get all reviews        |

**POST `/api/v1/reviews`**
- Parse body into `CreateReviewRequest`.
- Extract `username` from JWT locals (set by `middleware.Auth()`).
- Call `reviewService.Create(...)`.
- Return `201 Created` with the created review.

**GET `/api/v1/reviews`**
- Call `reviewService.GetAll(...)`.
- Return `200 OK` with list of reviews.

---

## Router

### Update `internal/router/router.go`

Wire the new handler and register routes:

```go
// inside Setup()
reviewRepo    := repository.NewReviewRepository()
reviewSvc     := service.NewReviewService(reviewRepo)
reviewHandler := handler.NewReviewHandler(reviewSvc)

// Public
api.Get("/reviews", reviewHandler.GetAll)

// Protected
reviews := api.Group("/reviews", middleware.Auth())
reviews.Post("/", reviewHandler.Create)
```

> Note: `GET /reviews` is public so unauthenticated visitors (e.g., landing page) can read reviews.

> Note: after complete this task make a documentation in folder doc/review

---

## Acceptance Criteria

- [ ] Migration file `009_create_reviews.sql` added.
- [ ] `domain.Review` struct defined.
- [ ] `CreateReviewRequest` and `ReviewResponse` DTOs defined.
- [ ] `ReviewRepository` interface + GORM implementation.
- [ ] `ReviewService` interface + implementation; username sourced from JWT, not body.
- [ ] `ReviewHandler` with `POST /api/v1/reviews` (auth) and `GET /api/v1/reviews` (public).
- [ ] Routes wired in `router.go`.
- [ ] Rating validation: must be 1–5.
- [ ] No other existing logic is modified.
