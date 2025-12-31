package reviews

import "time"

// CreateReviewRequest represents a review creation request
type CreateReviewRequest struct {
	TargetType  string `json:"target_type" binding:"required"`
	TargetID    string `json:"target_id" binding:"required"`
	Rating      int    `json:"rating" binding:"required,min=1,max=5"`
	ReviewTitle string `json:"review_title"` // Optional
	ReviewBody  string `json:"review_body" binding:"required"`
	Pros        string `json:"pros"`  // Optional
	Cons        string `json:"cons"`  // Optional
}

// UpdateReviewRequest represents a review update request
type UpdateReviewRequest struct {
	Rating      int    `json:"rating" binding:"required,min=1,max=5"`
	ReviewTitle string `json:"review_title"`
	ReviewBody  string `json:"review_body" binding:"required"`
	Pros        string `json:"pros"`
	Cons        string `json:"cons"`
}

// ReviewResponse represents a review response
type ReviewResponse struct {
	ID          string    `json:"id"`
	ExternalID  string    `json:"external_id"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	UserID      string    `json:"user_id"`
	Rating      int       `json:"rating"`
	ReviewTitle string    `json:"review_title,omitempty"`
	ReviewBody  string    `json:"review_body"`
	Pros        string    `json:"pros,omitempty"`
	Cons        string    `json:"cons,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ReviewsListResponse represents a paginated list of reviews
type ReviewsListResponse struct {
	Reviews      []*ReviewResponse `json:"reviews"`
	Total        int               `json:"total"`
	AverageRating float64          `json:"average_rating"`
	Page         int               `json:"page"`
	Limit        int               `json:"limit"`
	TotalPages   int               `json:"total_pages"`
}
