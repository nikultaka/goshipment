package dashboard

import (
	"database/sql"
	"fmt"
	"time"

	"gogin/internal/clients"
)

// DashboardService handles dashboard business logic
type DashboardService struct {
	db *clients.Database
}

// NewDashboardService creates a new dashboard service
func NewDashboardService(db *clients.Database) *DashboardService {
	return &DashboardService{db: db}
}

// GetDateRange calculates date range based on period
func (s *DashboardService) GetDateRange(period string, startDate, endDate string) (*DateRange, error) {
	loc, err := time.LoadLocation("America/Toronto")
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	switch period {
	case "today", "":
		return &DateRange{
			StartDate: today,
			EndDate:   today.Add(24*time.Hour - time.Second),
			Label:     "Today",
			Type:      "today",
		}, nil

	case "yesterday":
		yesterday := today.AddDate(0, 0, -1)
		return &DateRange{
			StartDate: yesterday,
			EndDate:   yesterday.Add(24*time.Hour - time.Second),
			Label:     "Yesterday",
			Type:      "yesterday",
		}, nil

	case "last_7_days":
		return &DateRange{
			StartDate: today.AddDate(0, 0, -6),
			EndDate:   today.Add(24*time.Hour - time.Second),
			Label:     "Last 7 Days",
			Type:      "last_7_days",
		}, nil

	case "last_week":
		weekday := int(today.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		lastMonday := today.AddDate(0, 0, -weekday-6)
		lastSunday := lastMonday.AddDate(0, 0, 6)
		return &DateRange{
			StartDate: lastMonday,
			EndDate:   lastSunday.Add(24*time.Hour - time.Second),
			Label:     "Last Week",
			Type:      "last_week",
		}, nil

	case "this_month":
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		return &DateRange{
			StartDate: firstOfMonth,
			EndDate:   now,
			Label:     now.Format("January 2006"),
			Type:      "this_month",
		}, nil

	case "last_month":
		firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		lastMonth := firstOfThisMonth.AddDate(0, -1, 0)
		lastDayOfLastMonth := firstOfThisMonth.Add(-time.Second)
		return &DateRange{
			StartDate: lastMonth,
			EndDate:   lastDayOfLastMonth,
			Label:     lastMonth.Format("January 2006"),
			Type:      "last_month",
		}, nil

	case "this_quarter":
		quarter := (int(now.Month()) - 1) / 3
		firstOfQuarter := time.Date(now.Year(), time.Month(quarter*3+1), 1, 0, 0, 0, 0, loc)
		return &DateRange{
			StartDate: firstOfQuarter,
			EndDate:   now,
			Label:     fmt.Sprintf("Q%d %d", quarter+1, now.Year()),
			Type:      "this_quarter",
		}, nil

	case "this_year":
		firstOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, loc)
		return &DateRange{
			StartDate: firstOfYear,
			EndDate:   now,
			Label:     fmt.Sprintf("%d", now.Year()),
			Type:      "this_year",
		}, nil

	case "custom":
		if startDate == "" || endDate == "" {
			return nil, fmt.Errorf("start_date and end_date are required for custom period")
		}
		start, err := time.ParseInLocation("2006-01-02", startDate, loc)
		if err != nil {
			return nil, fmt.Errorf("invalid start_date format: %w", err)
		}
		end, err := time.ParseInLocation("2006-01-02", endDate, loc)
		if err != nil {
			return nil, fmt.Errorf("invalid end_date format: %w", err)
		}
		if end.Before(start) {
			return nil, fmt.Errorf("end_date must be after start_date")
		}
		end = end.Add(24*time.Hour - time.Second)
		return &DateRange{
			StartDate: start,
			EndDate:   end,
			Label:     fmt.Sprintf("%s - %s", startDate, endDate),
			Type:      "custom",
		}, nil

	default:
		return nil, fmt.Errorf("invalid period: %s", period)
	}
}

// GetPreviousPeriodRange calculates the comparison period range
func (s *DashboardService) GetPreviousPeriodRange(current *DateRange) *DateRange {
	duration := current.EndDate.Sub(current.StartDate)

	return &DateRange{
		StartDate: current.StartDate.Add(-duration - time.Second),
		EndDate:   current.StartDate.Add(-time.Second),
		Label:     "Previous Period",
		Type:      "comparison",
	}
}

// GetFinancialMetrics retrieves financial metrics for the dashboard
func (s *DashboardService) GetFinancialMetrics(req FinancialMetricsRequest) (*FinancialMetricsResponse, error) {
	// Get date range
	period := req.Period
	if period == "" {
		period = "today"
	}

	dateRange, err := s.GetDateRange(period, req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	currency := req.Currency
	if currency == "" {
		currency = "CAD"
	}

	// Get main metrics
	summary, totals, err := s.getMetricsSummary(dateRange, req.OrganizationID, currency)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics summary: %w", err)
	}

	// Get breakdown by carrier
	carrierBreakdown, err := s.getCarrierBreakdown(dateRange, req.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get carrier breakdown: %w", err)
	}

	// Get breakdown by status
	statusBreakdown, err := s.getStatusBreakdown(dateRange, req.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get status breakdown: %w", err)
	}

	// Build response
	response := &FinancialMetricsResponse{
		Period: PeriodInfo{
			Type:         dateRange.Type,
			StartDate:    dateRange.StartDate.Format("2006-01-02"),
			EndDate:      dateRange.EndDate.Format("2006-01-02"),
			DisplayLabel: dateRange.Label,
		},
		Summary: *summary,
		Breakdown: MetricsBreakdown{
			ByCarrier: carrierBreakdown,
			ByStatus:  *statusBreakdown,
		},
		Totals:      *totals,
		GeneratedAt: time.Now(),
	}

	// Add comparison if requested
	if req.Compare {
		prevRange := s.GetPreviousPeriodRange(dateRange)
		comparison, err := s.getComparisonMetrics(prevRange, req.OrganizationID, currency, summary)
		if err == nil {
			response.Comparison = comparison
		}
	}

	return response, nil
}

// getMetricsSummary retrieves the main financial metrics
func (s *DashboardService) getMetricsSummary(dateRange *DateRange, organizationID, currency string) (*MetricsSummary, *MetricsTotals, error) {
	// Query for shipment financial metrics
	var shipmentQuery string
	var args []interface{}

	if organizationID == "" {
		shipmentQuery = `
			SELECT
				COUNT(*) as shipment_count,
				COALESCE(SUM(COALESCE(retail_rate, base_amount)), 0) as gross_revenue,
				COALESCE(SUM(base_amount), 0) as carrier_costs
			FROM shipments
			WHERE created_at >= $1
			  AND created_at <= $2
			  AND shipment_status NOT IN ('cancelled', 'voided', 'draft')
			  AND is_deleted = FALSE
		`
		args = []interface{}{dateRange.StartDate, dateRange.EndDate}
	} else {
		shipmentQuery = `
			SELECT
				COUNT(*) as shipment_count,
				COALESCE(SUM(COALESCE(retail_rate, base_amount)), 0) as gross_revenue,
				COALESCE(SUM(base_amount), 0) as carrier_costs
			FROM shipments
			WHERE created_at >= $1
			  AND created_at <= $2
			  AND organization_id = $3
			  AND shipment_status NOT IN ('cancelled', 'voided', 'draft')
			  AND is_deleted = FALSE
		`
		args = []interface{}{dateRange.StartDate, dateRange.EndDate, organizationID}
	}

	var shipmentCount int
	var grossRevenue, carrierCosts float64

	err := s.db.DB.QueryRow(shipmentQuery, args...).Scan(&shipmentCount, &grossRevenue, &carrierCosts)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, fmt.Errorf("failed to query shipment metrics: %w", err)
	}

	// Query for commission metrics
	var commissionQuery string
	if organizationID == "" {
		commissionQuery = `
			SELECT COALESCE(SUM(commission_amount), 0) as total_commissions
			FROM commission_transactions
			WHERE created_at >= $1
			  AND created_at <= $2
			  AND status IN ('eligible', 'paid')
		`
		args = []interface{}{dateRange.StartDate, dateRange.EndDate}
	} else {
		commissionQuery = `
			SELECT COALESCE(SUM(commission_amount), 0) as total_commissions
			FROM commission_transactions
			WHERE created_at >= $1
			  AND created_at <= $2
			  AND organization_id = $3
			  AND status IN ('eligible', 'paid')
		`
		args = []interface{}{dateRange.StartDate, dateRange.EndDate, organizationID}
	}

	var commissions float64
	err = s.db.DB.QueryRow(commissionQuery, args...).Scan(&commissions)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, fmt.Errorf("failed to query commission metrics: %w", err)
	}

	// Calculate profits
	grossProfit := grossRevenue - carrierCosts
	otherCosts := 0.0 // Placeholder for future cost tracking
	netProfit := grossProfit - commissions - otherCosts

	// Calculate margins
	grossMargin := 0.0
	if grossRevenue > 0 {
		grossMargin = (grossProfit / grossRevenue) * 100
	}
	netMargin := 0.0
	if grossRevenue > 0 {
		netMargin = (netProfit / grossRevenue) * 100
	}

	// Calculate averages
	avgOrderValue := 0.0
	avgMarginPerShipment := 0.0
	if shipmentCount > 0 {
		avgOrderValue = grossRevenue / float64(shipmentCount)
		avgMarginPerShipment = grossProfit / float64(shipmentCount)
	}

	summary := &MetricsSummary{
		GrossRevenue: MoneyField{
			Amount:    grossRevenue,
			Currency:  currency,
			Formatted: formatMoney(grossRevenue, currency),
		},
		CarrierCosts: MoneyField{
			Amount:    carrierCosts,
			Currency:  currency,
			Formatted: formatMoney(carrierCosts, currency),
		},
		GrossProfit: ProfitField{
			Amount:           grossProfit,
			Currency:         currency,
			Formatted:        formatMoney(grossProfit, currency),
			MarginPercentage: roundTo2(grossMargin),
		},
		Commissions: MoneyField{
			Amount:    commissions,
			Currency:  currency,
			Formatted: formatMoney(commissions, currency),
		},
		OtherCosts: MoneyField{
			Amount:    otherCosts,
			Currency:  currency,
			Formatted: formatMoney(otherCosts, currency),
		},
		NetProfit: ProfitField{
			Amount:           netProfit,
			Currency:         currency,
			Formatted:        formatMoney(netProfit, currency),
			MarginPercentage: roundTo2(netMargin),
		},
	}

	totals := &MetricsTotals{
		ShipmentCount:            shipmentCount,
		AverageOrderValue:        roundTo2(avgOrderValue),
		AverageMarginPerShipment: roundTo2(avgMarginPerShipment),
	}

	return summary, totals, nil
}

// getCarrierBreakdown retrieves financial breakdown by carrier
func (s *DashboardService) getCarrierBreakdown(dateRange *DateRange, organizationID string) ([]CarrierBreakdown, error) {
	var query string
	var args []interface{}

	if organizationID == "" {
		query = `
			SELECT
				s.carrier_id,
				COALESCE(c.name, s.carrier_id::text) as carrier_name,
				COUNT(*) as shipment_count,
				COALESCE(SUM(COALESCE(s.retail_rate, s.base_amount)), 0) as gross_revenue,
				COALESCE(SUM(s.base_amount), 0) as carrier_cost
			FROM shipments s
			LEFT JOIN carriers c ON s.carrier_id::text = c.id::text
			WHERE s.created_at >= $1
			  AND s.created_at <= $2
			  AND s.shipment_status NOT IN ('cancelled', 'voided', 'draft')
			  AND s.is_deleted = FALSE
			GROUP BY s.carrier_id, c.name
			ORDER BY gross_revenue DESC
			LIMIT 10
		`
		args = []interface{}{dateRange.StartDate, dateRange.EndDate}
	} else {
		query = `
			SELECT
				s.carrier_id,
				COALESCE(c.name, s.carrier_id::text) as carrier_name,
				COUNT(*) as shipment_count,
				COALESCE(SUM(COALESCE(s.retail_rate, s.base_amount)), 0) as gross_revenue,
				COALESCE(SUM(s.base_amount), 0) as carrier_cost
			FROM shipments s
			LEFT JOIN carriers c ON s.carrier_id::text = c.id::text
			WHERE s.created_at >= $1
			  AND s.created_at <= $2
			  AND s.organization_id = $3
			  AND s.shipment_status NOT IN ('cancelled', 'voided', 'draft')
			  AND s.is_deleted = FALSE
			GROUP BY s.carrier_id, c.name
			ORDER BY gross_revenue DESC
			LIMIT 10
		`
		args = []interface{}{dateRange.StartDate, dateRange.EndDate, organizationID}
	}

	rows, err := s.db.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query carrier breakdown: %w", err)
	}
	defer rows.Close()

	var breakdowns []CarrierBreakdown
	for rows.Next() {
		var b CarrierBreakdown
		err := rows.Scan(&b.CarrierID, &b.CarrierName, &b.ShipmentCount, &b.GrossRevenue, &b.CarrierCost)
		if err != nil {
			return nil, fmt.Errorf("failed to scan carrier breakdown: %w", err)
		}
		b.GrossProfit = b.GrossRevenue - b.CarrierCost
		if b.GrossRevenue > 0 {
			b.MarginPercentage = roundTo2((b.GrossProfit / b.GrossRevenue) * 100)
		}
		breakdowns = append(breakdowns, b)
	}

	return breakdowns, nil
}

// getStatusBreakdown retrieves breakdown by shipment status
func (s *DashboardService) getStatusBreakdown(dateRange *DateRange, organizationID string) (*StatusBreakdown, error) {
	var query string
	var args []interface{}

	if organizationID == "" {
		query = `
			SELECT
				shipment_status,
				COUNT(*) as count,
				COALESCE(SUM(COALESCE(retail_rate, base_amount)), 0) as revenue
			FROM shipments
			WHERE created_at >= $1
			  AND created_at <= $2
			  AND is_deleted = FALSE
			GROUP BY shipment_status
		`
		args = []interface{}{dateRange.StartDate, dateRange.EndDate}
	} else {
		query = `
			SELECT
				shipment_status,
				COUNT(*) as count,
				COALESCE(SUM(COALESCE(retail_rate, base_amount)), 0) as revenue
			FROM shipments
			WHERE created_at >= $1
			  AND created_at <= $2
			  AND organization_id = $3
			  AND is_deleted = FALSE
			GROUP BY shipment_status
		`
		args = []interface{}{dateRange.StartDate, dateRange.EndDate, organizationID}
	}

	rows, err := s.db.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query status breakdown: %w", err)
	}
	defer rows.Close()

	breakdown := &StatusBreakdown{}
	totalCreated := 0
	for rows.Next() {
		var status string
		var count int
		var revenue float64
		err := rows.Scan(&status, &count, &revenue)
		if err != nil {
			return nil, fmt.Errorf("failed to scan status breakdown: %w", err)
		}

		totalCreated += count
		switch status {
		case "draft", "awaiting_payment", "payment_failed":
			breakdown.Pending.Count += count
			breakdown.Pending.Revenue += revenue
		case "label_purchased", "ready_for_pickup", "pickup_scheduled":
			breakdown.LabelPurchased.Count += count
			breakdown.LabelPurchased.Revenue += revenue
		case "in_transit", "out_for_delivery":
			breakdown.InTransit.Count += count
			breakdown.InTransit.Revenue += revenue
		case "delivered":
			breakdown.Delivered.Count += count
			breakdown.Delivered.Revenue += revenue
		case "delivery_failed", "returned":
			breakdown.Failed.Count += count
			breakdown.Failed.Revenue += revenue
		case "cancelled", "voided":
			breakdown.Cancelled.Count += count
			breakdown.Cancelled.Revenue += revenue
		case "refunded":
			breakdown.Refunded.Count += count
			breakdown.Refunded.Revenue += revenue
		}
	}

	breakdown.TotalCreated = totalCreated
	return breakdown, nil
}

// getComparisonMetrics retrieves metrics for the comparison period
func (s *DashboardService) getComparisonMetrics(prevRange *DateRange, organizationID, currency string, current *MetricsSummary) (*ComparisonMetrics, error) {
	prevSummary, _, err := s.getMetricsSummary(prevRange, organizationID, currency)
	if err != nil {
		return nil, err
	}

	comparison := &ComparisonMetrics{
		Period: PeriodInfo{
			Type:         prevRange.Type,
			StartDate:    prevRange.StartDate.Format("2006-01-02"),
			EndDate:      prevRange.EndDate.Format("2006-01-02"),
			DisplayLabel: prevRange.Label,
		},
		GrossRevenue: calculateComparison(prevSummary.GrossRevenue.Amount, current.GrossRevenue.Amount),
		CarrierCosts: calculateComparison(prevSummary.CarrierCosts.Amount, current.CarrierCosts.Amount),
		GrossProfit:  calculateComparison(prevSummary.GrossProfit.Amount, current.GrossProfit.Amount),
		Commissions:  calculateComparison(prevSummary.Commissions.Amount, current.Commissions.Amount),
		NetProfit:    calculateComparison(prevSummary.NetProfit.Amount, current.NetProfit.Amount),
	}

	return comparison, nil
}

// Helper functions

func calculateComparison(previous, current float64) ComparisonField {
	change := current - previous
	var changePercent float64
	if previous != 0 {
		changePercent = (change / previous) * 100
	}

	trend := "neutral"
	if change > 0 {
		trend = "up"
	} else if change < 0 {
		trend = "down"
	}

	return ComparisonField{
		Amount:           previous,
		ChangeAmount:     roundTo2(change),
		ChangePercentage: roundTo2(changePercent),
		Trend:            trend,
	}
}

func formatMoney(amount float64, currency string) string {
	symbol := "$"
	switch currency {
	case "EUR":
		symbol = "\u20ac"
	case "GBP":
		symbol = "\u00a3"
	}
	return fmt.Sprintf("%s%.2f", symbol, amount)
}

func roundTo2(val float64) float64 {
	return float64(int(val*100)) / 100
}
