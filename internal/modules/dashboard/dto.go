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
	Cancelled      StatusCount `json:"cancelled"`        // cancelled, voided
	Refunded       StatusCount `json:"refunded"`         // refunded
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
