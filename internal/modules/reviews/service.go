package reviews

import (
	"fmt"
	"time"

	"gogin/internal/clients"

	"github.com/google/uuid"
)

type ReviewsService struct {
	db *clients.Database
}

func NewReviewsService(db *clients.Database) *ReviewsService {
	return &ReviewsService{db: db}
}

func (s *ReviewsService) CreateReview(userID string, req *CreateReviewRequest) (*ReviewResponse, error) {
	id := uuid.New().String()
	externalID := fmt.Sprintf("REV-%s", uuid.New().String()[:8])

	query := `
		INSERT INTO reviews (
			id, external_id, user_id, rating, review_title, review_body,
			pros, cons, target_type, target_id, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	var createdAt, updatedAt time.Time
	err := s.db.QueryRow(
		query,
		id,
		externalID,
		userID,
		req.Rating,
		req.ReviewTitle,
		req.ReviewBody,
		req.Pros,
		req.Cons,
		req.TargetType,
		req.TargetID,
		"pending", // New reviews should be pending until approved
	).Scan(&createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create review: %w", err)
	}

	return &ReviewResponse{
		ID:          id,
		ExternalID:  externalID,
		TargetType:  req.TargetType,
		TargetID:    req.TargetID,
		UserID:      userID,
		Rating:      req.Rating,
		ReviewTitle: req.ReviewTitle,
		ReviewBody:  req.ReviewBody,
		Pros:        req.Pros,
		Cons:        req.Cons,
		Status:      "pending",
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

func (s *ReviewsService) ListReviews(resourceType, resourceID string, page, limit int) ([]*ReviewResponse, int, float64, error) {
	offset := (page - 1) * limit

	var total int
	var avgRating float64
	var countQuery string
	var selectQuery string
	var args []interface{}

	// If both resourceType and resourceID are provided, filter by them
	if resourceType != "" && resourceID != "" {
		countQuery = `SELECT COUNT(*), COALESCE(AVG(rating), 0) FROM reviews WHERE target_type = $1 AND target_id = $2 AND status = 'published'`
		selectQuery = `SELECT id, target_type as resource_type, target_id as resource_id, user_id, rating, review_title as title, review_body as content, status, created_at, updated_at FROM reviews WHERE target_type = $1 AND target_id = $2 AND status = 'published' ORDER BY created_at DESC LIMIT $3 OFFSET $4`
		args = []interface{}{resourceType, resourceID, limit, offset}

		err := s.db.QueryRow(countQuery, resourceType, resourceID).Scan(&total, &avgRating)
		if err != nil {
			return nil, 0, 0, err
		}
	} else {
		// Otherwise, return all published reviews
		countQuery = `SELECT COUNT(*), COALESCE(AVG(rating), 0) FROM reviews WHERE status = 'published'`
		selectQuery = `SELECT id, target_type as resource_type, target_id as resource_id, user_id, rating, review_title as title, review_body as content, status, created_at, updated_at FROM reviews WHERE status = 'published' ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		args = []interface{}{limit, offset}

		err := s.db.QueryRow(countQuery).Scan(&total, &avgRating)
		if err != nil {
			return nil, 0, 0, err
		}
	}

	rows, err := s.db.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()

	var reviews []*ReviewResponse
	for rows.Next() {
		var r ReviewResponse
		rows.Scan(&r.ID, &r.TargetType, &r.TargetID, &r.UserID, &r.Rating, &r.ReviewTitle, &r.ReviewBody, &r.Status, &r.CreatedAt, &r.UpdatedAt)
		reviews = append(reviews, &r)
	}

	return reviews, total, avgRating, nil
}

func (s *ReviewsService) GetReview(id string) (*ReviewResponse, error) {
	query := `
		SELECT id, external_id, target_type, target_id, user_id, rating,
		       review_title, review_body, pros, cons, status, created_at, updated_at
		FROM reviews WHERE id = $1
	`

	var r ReviewResponse
	err := s.db.QueryRow(query, id).Scan(
		&r.ID, &r.ExternalID, &r.TargetType, &r.TargetID, &r.UserID, &r.Rating,
		&r.ReviewTitle, &r.ReviewBody, &r.Pros, &r.Cons, &r.Status, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *ReviewsService) UpdateReview(id, userID string, req *UpdateReviewRequest) (*ReviewResponse, error) {
	query := `
		UPDATE reviews
		SET rating = $1, review_title = $2, review_body = $3, pros = $4, cons = $5, updated_at = NOW()
		WHERE id = $6 AND user_id = $7
	`
	result, err := s.db.Exec(query, req.Rating, req.ReviewTitle, req.ReviewBody, req.Pros, req.Cons, id, userID)
	if err != nil {
		return nil, err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return nil, fmt.Errorf("review not found")
	}
	return s.GetReview(id)
}

func (s *ReviewsService) DeleteReview(id, userID string) error {
	result, err := s.db.Exec(`DELETE FROM reviews WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("review not found")
	}
	return nil
}
