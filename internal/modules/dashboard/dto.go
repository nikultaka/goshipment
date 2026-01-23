package dashboard

import "time"

// FinancialMetricsRequest represents the request parameters for financial metrics
type FinancialMetricsRequest struct {
	Period         string `form:"period" binding:"omitempty,oneof=today yesterday last_7_days last_week this_month last_month this_quarter this_year custom"`
	StartDate      string `form:"start_date" binding:"omitempty"`
	EndDate        string `form:"end_date" binding:"omitempty"`
	OrganizationID string `form:"organization_id" binding:"omitempty,uuid"`
	Currency       string `form:"currency" binding:"omitempty,len=3"`
	Compare        bool   `form:"compare"`
}

// DateRange represents a date range for filtering
type DateRange struct {
	StartDate time.Time
	EndDate   time.Time
	Label     string
	Type      string
}

// PeriodInfo represents period information in the response
type PeriodInfo struct {
	Type         string `json:"type"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	DisplayLabel string `json:"display_label"`
}

// MoneyField represents a monetary value with formatting
type MoneyField struct {
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Formatted string  `json:"formatted"`
}

// ProfitField represents a profit value with margin percentage
type ProfitField struct {
	Amount           float64 `json:"amount"`
	Currency         string  `json:"currency"`
	Formatted        string  `json:"formatted"`
	MarginPercentage float64 `json:"margin_percentage"`
}

// MetricsSummary represents the summary of financial metrics
type MetricsSummary struct {
	GrossRevenue MoneyField  `json:"gross_revenue"`
	CarrierCosts MoneyField  `json:"carrier_costs"`
	GrossProfit  ProfitField `json:"gross_profit"`
	Commissions  MoneyField  `json:"commissions"`
	OtherCosts   MoneyField  `json:"other_costs"`
	NetProfit    ProfitField `json:"net_profit"`
}

// ComparisonField represents a comparison value with change metrics
type ComparisonField struct {
	Amount           float64 `json:"amount"`
	ChangeAmount     float64 `json:"change_amount"`
	ChangePercentage float64 `json:"change_percentage"`
	Trend            string  `json:"trend"` // "up", "down", "neutral"
}

// ComparisonMetrics represents comparison with previous period
type ComparisonMetrics struct {
	Period       PeriodInfo      `json:"period"`
	GrossRevenue ComparisonField `json:"gross_revenue"`
	CarrierCosts ComparisonField `json:"carrier_costs"`
	GrossProfit  ComparisonField `json:"gross_profit"`
	Commissions  ComparisonField `json:"commissions"`
	NetProfit    ComparisonField `json:"net_profit"`
}

// CarrierBreakdown represents financial breakdown by carrier
type CarrierBreakdown struct {
	CarrierID        string  `json:"carrier_id"`
	CarrierName      string  `json:"carrier_name"`
	ShipmentCount    int     `json:"shipment_count"`
	GrossRevenue     float64 `json:"gross_revenue"`
	CarrierCost      float64 `json:"carrier_cost"`
	GrossProfit      float64 `json:"gross_profit"`
	MarginPercentage float64 `json:"margin_percentage"`
}

// StatusCount represents count and revenue by shipment status
type StatusCount struct {
	Count   int     `json:"count"`
	Revenue float64 `json:"revenue"`
}

// StatusBreakdown represents breakdown by shipment status
type StatusBreakdown struct {
	TotalCreated   int         `json:"total_created"`
	Pending        StatusCount `json:"pending"`          // draft, awaiting_payment, payment_failed
	LabelPurchased StatusCount `json:"label_purchased"`  // label_purchased, ready_for_pickup, pickup_scheduled
	InTransit      StatusCount `json:"in_transit"`       // in_transit, out_for_delivery
	Delivered      StatusCount `json:"delivered"`        // delivered
	Failed         StatusCount `json:"failed"`           // delivery_failed, returned
	Cancelled      StatusCount `json:"cancelled"`        // cancelled
	Voided         StatusCount `json:"voided"`           // voided
	Refunded       StatusCount `json:"refunded"`         // refunded
	Archived       StatusCount `json:"archived"`         // is_archived = true
}

// MetricsBreakdown represents detailed breakdown of metrics
type MetricsBreakdown struct {
	ByCarrier []CarrierBreakdown `json:"by_carrier"`
	ByStatus  StatusBreakdown    `json:"by_status"`
}

// MetricsTotals represents total aggregated values
type MetricsTotals struct {
	ShipmentCount           int     `json:"shipment_count"`
	AverageOrderValue       float64 `json:"average_order_value"`
	AverageMarginPerShipment float64 `json:"average_margin_per_shipment"`
}

// FinancialMetricsResponse represents the complete financial metrics response
type FinancialMetricsResponse struct {
	Period      PeriodInfo         `json:"period"`
	Summary     MetricsSummary     `json:"summary"`
	Comparison  *ComparisonMetrics `json:"comparison,omitempty"`
	Breakdown   MetricsBreakdown   `json:"breakdown"`
	Totals      MetricsTotals      `json:"totals"`
	GeneratedAt time.Time          `json:"generated_at"`
}

// ============================================================================
// User Overview DTOs
// ============================================================================

// UserOverviewRequest represents the request parameters for user overview
type UserOverviewRequest struct {
	Period  string `form:"period" binding:"omitempty,oneof=today yesterday last_7_days last_week this_month last_month this_quarter this_year"`
	Compare bool   `form:"compare"`
}

// UserMetricWithChange represents a user metric with comparison data
type UserMetricWithChange struct {
	Count            int     `json:"count"`
	Change           int     `json:"change,omitempty"`
	ChangePercentage float64 `json:"change_percentage,omitempty"`
	Trend            string  `json:"trend,omitempty"` // "up", "down", "neutral"
}

// UserMetricWithPercentage represents a user metric with percentage of total
type UserMetricWithPercentage struct {
	Count             int     `json:"count"`
	PercentageOfTotal float64 `json:"percentage_of_total"`
	Change            int     `json:"change,omitempty"`
	ChangePercentage  float64 `json:"change_percentage,omitempty"`
	Trend             string  `json:"trend,omitempty"`
}

// OnlineUsersMetric represents online users metric
type OnlineUsersMetric struct {
	Count             int     `json:"count"`
	PercentageOfTotal float64 `json:"percentage_of_total"`
}

// UserMetrics represents all user metrics
type UserMetrics struct {
	TotalUsers       UserMetricWithChange     `json:"total_users"`
	NewUsersToday    UserMetricWithChange     `json:"new_users_today"`
	OnlineUsers      OnlineUsersMetric        `json:"online_users"`
	BusinessAccounts UserMetricWithPercentage `json:"business_accounts"`
	UnverifiedUsers  UserMetricWithPercentage `json:"unverified_users"`
}

// RoleBreakdown represents user count by role
type RoleBreakdown struct {
	Role       string  `json:"role"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// UserStatusBreakdown represents user count by status
type UserStatusBreakdown struct {
	Status     string  `json:"status"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// UserBreakdown represents breakdown of users by role and status
type UserBreakdown struct {
	ByRole   []RoleBreakdown       `json:"by_role"`
	ByStatus []UserStatusBreakdown `json:"by_status"`
}

// UserOverviewResponse represents the complete user overview response
type UserOverviewResponse struct {
	Period      PeriodInfo    `json:"period"`
	Metrics     UserMetrics   `json:"metrics"`
	Breakdown   UserBreakdown `json:"breakdown"`
	GeneratedAt time.Time     `json:"generated_at"`
}

// ============================================================================
// Top Spending Businesses DTOs
// ============================================================================

// TopSpendingBusinessRequest represents the request parameters
type TopSpendingBusinessRequest struct {
	Limit int `form:"limit" binding:"omitempty,min=1,max=50"`
}

// TopSpendingBusiness represents a single business in the top spending list
type TopSpendingBusiness struct {
	Rank             int     `json:"rank" example:"1"`
	OrganizationID   string  `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	OrganizationName string  `json:"organization_name" example:"Acme Shipping Co."`
	BusinessName     string  `json:"business_name,omitempty" example:"Acme Inc."`
	TotalSpent       float64 `json:"total_spent" example:"15250.75"`
	ShipmentCount    int     `json:"shipment_count" example:"142"`
	AveragePerOrder  float64 `json:"average_per_order" example:"107.40"`
	Currency         string  `json:"currency" example:"CAD"`
}

// TopSpendingBusinessesResponse represents the response for top spending businesses
type TopSpendingBusinessesResponse struct {
	Period      PeriodInfo            `json:"period"`
	Businesses  []TopSpendingBusiness `json:"businesses"`
	TotalCount  int                   `json:"total_count" example:"10"`
	GeneratedAt time.Time             `json:"generated_at"`
}

// ============================================================================
// Pickups & Exceptions DTOs
// ============================================================================

// PickupsExceptionsRequest represents the request parameters
type PickupsExceptionsRequest struct {
	Days  int `form:"days" binding:"omitempty,min=1,max=30"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

// PickupSummary represents a pickup in the summary list
type PickupSummary struct {
	ID               string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	PickupNumber     string  `json:"pickup_number" example:"PKU-a1b2c3d4"`
	OrganizationID   string  `json:"organization_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	OrganizationName string  `json:"organization_name" example:"Acme Shipping Co."`
	ShipmentID       string  `json:"shipment_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	ShipmentNumber   string  `json:"shipment_number" example:"SHP-xyz98765"`
	CarrierName      string  `json:"carrier_name" example:"Canada Post"`
	PickupDate       string  `json:"pickup_date" example:"2026-01-24"`
	PickupWindow     string  `json:"pickup_window" example:"09:00 - 17:00"`
	PickupCity       string  `json:"pickup_city" example:"Toronto"`
	PickupStatus     string  `json:"pickup_status" example:"scheduled"`
	FailureReason    string  `json:"failure_reason,omitempty" example:"Address not accessible"`
	IsException      bool    `json:"is_exception" example:"false"`
	CreatedAt        string  `json:"created_at" example:"2026-01-23T10:30:00Z"`
}

// PickupStatusCount represents count by status
type PickupStatusCount struct {
	Scheduled   int `json:"scheduled" example:"15"`
	Rescheduled int `json:"rescheduled" example:"2"`
	Completed   int `json:"completed" example:"45"`
	Cancelled   int `json:"cancelled" example:"3"`
	Failed      int `json:"failed" example:"1"`
}

// PickupsExceptionsResponse represents the response for pickups and exceptions
type PickupsExceptionsResponse struct {
	Period         PeriodInfo        `json:"period"`
	Summary        PickupStatusCount `json:"summary"`
	TodaysPickups  []PickupSummary   `json:"todays_pickups"`
	Exceptions     []PickupSummary   `json:"exceptions"`
	TotalPickups   int               `json:"total_pickups" example:"66"`
	TotalExceptions int              `json:"total_exceptions" example:"4"`
	GeneratedAt    time.Time         `json:"generated_at"`
}
