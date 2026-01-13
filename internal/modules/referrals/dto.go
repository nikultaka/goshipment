package referrals

import (
	"time"

	"gogin/internal/models"
)

// ListReferralsRequest represents query parameters for listing referrals
type ListReferralsRequest struct {
	Status     string `form:"status" binding:"omitempty,oneof=active revoked expired"`
	FromDate   string `form:"from_date" binding:"omitempty"` // YYYY-MM-DD
	ToDate     string `form:"to_date" binding:"omitempty"`   // YYYY-MM-DD
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// ReferralRedemptionResponse represents a referral redemption in API responses
type ReferralRedemptionResponse struct {
	ID                  string    `json:"id"`
	ReferenceNumber     string    `json:"reference_number"`
	ReferrerUserID      string    `json:"referrer_user_id"`
	ReferredUserID      string    `json:"referred_user_id"`
	ReferredUserEmail   string    `json:"referred_user_email,omitempty"` // Masked for privacy if needed
	Status              string    `json:"status"`
	RedeemedAt          time.Time `json:"redeemed_at"`
	CommissionNumber    string    `json:"commission_number"`
	CommissionBase      string    `json:"commission_base"`
	CommissionValueType string    `json:"commission_value_type"`
	CommissionValue     float64   `json:"commission_value"`
	SnapshotTakenAt     time.Time `json:"snapshot_taken_at"`
	CreatedAt           time.Time `json:"created_at"`
}

// ListReferralsResponse represents the response for listing referrals
type ListReferralsResponse struct {
	Referrals      []*ReferralRedemptionResponse `json:"referrals"`
	TotalCount     int                           `json:"total_count"`
	Page           int                           `json:"page"`
	PageSize       int                           `json:"page_size"`
	TotalPages     int                           `json:"total_pages"`
	ActiveCount    int                           `json:"active_count"`
	RevokedCount   int                           `json:"revoked_count"`
	ExpiredCount   int                           `json:"expired_count"`
}

// ReferralDetailResponse represents detailed referral information
type ReferralDetailResponse struct {
	ID                  string    `json:"id"`
	ReferenceNumber     string    `json:"reference_number"`
	OrganizationID      string    `json:"organization_id,omitempty"`
	ReferrerUserID      string    `json:"referrer_user_id"`
	ReferrerEmail       string    `json:"referrer_email,omitempty"`
	ReferredUserID      string    `json:"referred_user_id"`
	ReferredUserEmail   string    `json:"referred_user_email,omitempty"`
	Status              string    `json:"status"`
	RedeemedAt          time.Time `json:"redeemed_at"`
	CommissionPlanID    string    `json:"commission_plan_id,omitempty"`
	CommissionNumber    string    `json:"commission_number"`
	CommissionBase      string    `json:"commission_base"`
	CommissionValueType string    `json:"commission_value_type"`
	CommissionValue     float64   `json:"commission_value"`
	SnapshotTakenAt     time.Time `json:"snapshot_taken_at"`
	Metadata            string    `json:"metadata,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

// ToReferralRedemptionResponse converts a ReferralRedemption model to list response DTO
func ToReferralRedemptionResponse(r *models.ReferralRedemption, referredEmail string) *ReferralRedemptionResponse {
	return &ReferralRedemptionResponse{
		ID:                  r.ID,
		ReferenceNumber:     r.ReferenceNumber,
		ReferrerUserID:      r.ReferrerUserID,
		ReferredUserID:      r.ReferredUserID,
		ReferredUserEmail:   referredEmail,
		Status:              string(r.Status),
		RedeemedAt:          r.RedeemedAt,
		CommissionNumber:    r.CommissionNumber,
		CommissionBase:      string(r.CommissionBase),
		CommissionValueType: string(r.CommissionValueType),
		CommissionValue:     r.CommissionValue,
		SnapshotTakenAt:     r.SnapshotTakenAt,
		CreatedAt:           r.CreatedAt,
	}
}

// ToReferralDetailResponse converts a ReferralRedemption model to detailed response DTO
func ToReferralDetailResponse(r *models.ReferralRedemption, referrerEmail, referredEmail string) *ReferralDetailResponse {
	resp := &ReferralDetailResponse{
		ID:                  r.ID,
		ReferenceNumber:     r.ReferenceNumber,
		ReferrerUserID:      r.ReferrerUserID,
		ReferrerEmail:       referrerEmail,
		ReferredUserID:      r.ReferredUserID,
		ReferredUserEmail:   referredEmail,
		Status:              string(r.Status),
		RedeemedAt:          r.RedeemedAt,
		CommissionNumber:    r.CommissionNumber,
		CommissionBase:      string(r.CommissionBase),
		CommissionValueType: string(r.CommissionValueType),
		CommissionValue:     r.CommissionValue,
		SnapshotTakenAt:     r.SnapshotTakenAt,
		CreatedAt:           r.CreatedAt,
	}

	if r.OrganizationID.Valid {
		resp.OrganizationID = r.OrganizationID.String
	}

	if r.CommissionPlanID.Valid {
		resp.CommissionPlanID = r.CommissionPlanID.String
	}

	if r.Metadata.Valid {
		resp.Metadata = r.Metadata.String
	}

	return resp
}
