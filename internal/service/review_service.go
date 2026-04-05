package service

import (
	"context"
	"errors"

	"backend-pretest-ai/internal/domain"
	"backend-pretest-ai/internal/dto"
	"backend-pretest-ai/internal/repository"
)

var (
	ErrUserNotFoundReview = errors.New("user not found for review")
	ErrReviewNotFound     = errors.New("review not found")
	ErrNotReviewOwner     = errors.New("you are not the author of this review")
	ErrReviewAlreadyExist = errors.New("you have already submitted a review")
)

type ReviewService interface {
	Create(ctx context.Context, userID string, req dto.CreateReviewRequest) (*dto.ReviewResponse, error)
	GetAll(ctx context.Context) ([]dto.ReviewResponse, error)
	Update(ctx context.Context, userID string, reviewID string, req dto.CreateReviewRequest) (*dto.ReviewResponse, error)
	Delete(ctx context.Context, userID string, reviewID string) error
}

type reviewService struct {
	reviewRepo repository.ReviewRepository
	userRepo   repository.UserRepository
}

func NewReviewService(reviewRepo repository.ReviewRepository, userRepo repository.UserRepository) ReviewService {
	return &reviewService{
		reviewRepo: reviewRepo,
		userRepo:   userRepo,
	}
}

func (s *reviewService) Create(ctx context.Context, userID string, req dto.CreateReviewRequest) (*dto.ReviewResponse, error) {
	// Enforce one review per user
	exists, err := s.reviewRepo.ExistsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrReviewAlreadyExist
	}

	// Fetch user to get the name
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFoundReview
	}

	review := &domain.Review{
		UserID:   userID,
		Username: user.Name,
		Position: req.Position,
		Review:   req.Review,
		Rating:   req.Rating,
	}

	if err := s.reviewRepo.Create(ctx, review); err != nil {
		return nil, err
	}

	return &dto.ReviewResponse{
		ID:        review.ID,
		Username:  review.Username,
		Position:  review.Position,
		Review:    review.Review,
		Rating:    review.Rating,
		CreatedAt: review.CreatedAt,
	}, nil
}

func (s *reviewService) GetAll(ctx context.Context) ([]dto.ReviewResponse, error) {
	reviews, err := s.reviewRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.ReviewResponse, 0, len(reviews))
	for _, r := range reviews {
		responses = append(responses, dto.ReviewResponse{
			ID:        r.ID,
			Username:  r.Username,
			Position:  r.Position,
			Review:    r.Review,
			Rating:    r.Rating,
			CreatedAt: r.CreatedAt,
		})
	}

	return responses, nil
}

func (s *reviewService) Update(ctx context.Context, userID string, reviewID string, req dto.CreateReviewRequest) (*dto.ReviewResponse, error) {
	review, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, ErrReviewNotFound
	}

	if review.UserID != userID {
		return nil, ErrNotReviewOwner
	}

	review.Position = req.Position
	review.Review = req.Review
	review.Rating = req.Rating

	if err := s.reviewRepo.Update(ctx, review); err != nil {
		return nil, err
	}

	return &dto.ReviewResponse{
		ID:        review.ID,
		Username:  review.Username,
		Position:  review.Position,
		Review:    review.Review,
		Rating:    review.Rating,
		CreatedAt: review.CreatedAt,
	}, nil
}

func (s *reviewService) Delete(ctx context.Context, userID string, reviewID string) error {
	review, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return ErrReviewNotFound
	}

	if review.UserID != userID {
		return ErrNotReviewOwner
	}

	return s.reviewRepo.Delete(ctx, reviewID)
}
