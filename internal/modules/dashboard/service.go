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

// ============================================================================
// User Overview Methods
// ============================================================================

// GetUserOverview retrieves user overview metrics for the dashboard
func (s *DashboardService) GetUserOverview(req UserOverviewRequest) (*UserOverviewResponse, error) {
	period := req.Period
	if period == "" {
		period = "today"
	}

	dateRange, err := s.GetDateRange(period, "", "")
	if err != nil {
		return nil, err
	}

	// Get current metrics
	metrics, err := s.getUserMetrics(dateRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get user metrics: %w", err)
	}

	// Get comparison metrics if requested
	if req.Compare {
		prevRange := s.GetPreviousPeriodRange(dateRange)
		s.addUserComparison(metrics, prevRange)
	}

	// Get breakdowns
	breakdown, err := s.getUserBreakdown()
	if err != nil {
		return nil, fmt.Errorf("failed to get user breakdown: %w", err)
	}

	return &UserOverviewResponse{
		Period: PeriodInfo{
			Type:         dateRange.Type,
			StartDate:    dateRange.StartDate.Format("2006-01-02"),
			EndDate:      dateRange.EndDate.Format("2006-01-02"),
			DisplayLabel: dateRange.Label,
		},
		Metrics:     *metrics,
		Breakdown:   *breakdown,
		GeneratedAt: time.Now(),
	}, nil
}

// getUserMetrics retrieves the main user metrics
func (s *DashboardService) getUserMetrics(dateRange *DateRange) (*UserMetrics, error) {
	// Total users count
	var totalUsers int
	err := s.db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE deleted_at IS NULL").Scan(&totalUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to count total users: %w", err)
	}

	// New users in period
	var newUsers int
	err = s.db.DB.QueryRow(
		"SELECT COUNT(*) FROM users WHERE created_at >= $1 AND created_at <= $2 AND deleted_at IS NULL",
		dateRange.StartDate, dateRange.EndDate,
	).Scan(&newUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to count new users: %w", err)
	}

	// Online users (active session in last 15 minutes)
	var onlineUsers int
	err = s.db.DB.QueryRow(`
		SELECT COUNT(DISTINCT user_id)
		FROM user_sessions
		WHERE last_activity_at >= NOW() - INTERVAL '15 minutes'
		AND is_active = true
	`).Scan(&onlineUsers)
	if err != nil {
		onlineUsers = 0 // Default to 0 if table doesn't exist or query fails
	}

	// Business accounts (users with business profile)
	var businessAccounts int
	err = s.db.DB.QueryRow(`
		SELECT COUNT(DISTINCT b.user_id)
		FROM businesses b
		INNER JOIN users u ON b.user_id = u.id
		WHERE u.deleted_at IS NULL AND b.is_active = true
	`).Scan(&businessAccounts)
	if err != nil {
		businessAccounts = 0
	}

	// Unverified users (primary email not verified)
	var unverifiedUsers int
	err = s.db.DB.QueryRow(`
		SELECT COUNT(*)
		FROM users u
		LEFT JOIN user_emails ue ON u.primary_email_id = ue.id
		WHERE u.deleted_at IS NULL
		AND (ue.is_verified = false OR ue.is_verified IS NULL)
	`).Scan(&unverifiedUsers)
	if err != nil {
		unverifiedUsers = 0
	}

	// Calculate percentages
	businessPct := 0.0
	unverifiedPct := 0.0
	onlinePct := 0.0
	if totalUsers > 0 {
		businessPct = roundTo2((float64(businessAccounts) / float64(totalUsers)) * 100)
		unverifiedPct = roundTo2((float64(unverifiedUsers) / float64(totalUsers)) * 100)
		onlinePct = roundTo2((float64(onlineUsers) / float64(totalUsers)) * 100)
	}

	return &UserMetrics{
		TotalUsers: UserMetricWithChange{
			Count: totalUsers,
		},
		NewUsersToday: UserMetricWithChange{
			Count: newUsers,
		},
		OnlineUsers: OnlineUsersMetric{
			Count:             onlineUsers,
			PercentageOfTotal: onlinePct,
		},
		BusinessAccounts: UserMetricWithPercentage{
			Count:             businessAccounts,
			PercentageOfTotal: businessPct,
		},
		UnverifiedUsers: UserMetricWithPercentage{
			Count:             unverifiedUsers,
			PercentageOfTotal: unverifiedPct,
		},
	}, nil
}

// addUserComparison adds comparison data to metrics
func (s *DashboardService) addUserComparison(metrics *UserMetrics, prevRange *DateRange) {
	// Previous total users (created before end of prev period)
	var prevTotal int
	s.db.DB.QueryRow(
		"SELECT COUNT(*) FROM users WHERE created_at <= $1 AND deleted_at IS NULL",
		prevRange.EndDate,
	).Scan(&prevTotal)

	// Previous new users (created during prev period)
	var prevNew int
	s.db.DB.QueryRow(
		"SELECT COUNT(*) FROM users WHERE created_at >= $1 AND created_at <= $2 AND deleted_at IS NULL",
		prevRange.StartDate, prevRange.EndDate,
	).Scan(&prevNew)

	// Previous business accounts
	var prevBusiness int
	s.db.DB.QueryRow(`
		SELECT COUNT(DISTINCT b.user_id)
		FROM businesses b
		INNER JOIN users u ON b.user_id = u.id
		WHERE u.deleted_at IS NULL AND b.is_active = true
		AND b.created_at <= $1
	`, prevRange.EndDate).Scan(&prevBusiness)

	// Previous unverified users
	var prevUnverified int
	s.db.DB.QueryRow(`
		SELECT COUNT(*)
		FROM users u
		LEFT JOIN user_emails ue ON u.primary_email_id = ue.id
		WHERE u.deleted_at IS NULL
		AND u.created_at <= $1
		AND (ue.is_verified = false OR ue.is_verified IS NULL)
	`, prevRange.EndDate).Scan(&prevUnverified)

	// Total users comparison
	metrics.TotalUsers.Change = metrics.TotalUsers.Count - prevTotal
	if prevTotal > 0 {
		metrics.TotalUsers.ChangePercentage = roundTo2((float64(metrics.TotalUsers.Change) / float64(prevTotal)) * 100)
	}
	metrics.TotalUsers.Trend = getTrend(metrics.TotalUsers.Change)

	// New users comparison
	metrics.NewUsersToday.Change = metrics.NewUsersToday.Count - prevNew
	if prevNew > 0 {
		metrics.NewUsersToday.ChangePercentage = roundTo2((float64(metrics.NewUsersToday.Change) / float64(prevNew)) * 100)
	}
	metrics.NewUsersToday.Trend = getTrend(metrics.NewUsersToday.Change)

	// Business accounts comparison
	metrics.BusinessAccounts.Change = metrics.BusinessAccounts.Count - prevBusiness
	if prevBusiness > 0 {
		metrics.BusinessAccounts.ChangePercentage = roundTo2((float64(metrics.BusinessAccounts.Change) / float64(prevBusiness)) * 100)
	}
	metrics.BusinessAccounts.Trend = getTrend(metrics.BusinessAccounts.Change)

	// Unverified users comparison
	metrics.UnverifiedUsers.Change = metrics.UnverifiedUsers.Count - prevUnverified
	if prevUnverified > 0 {
		metrics.UnverifiedUsers.ChangePercentage = roundTo2((float64(metrics.UnverifiedUsers.Change) / float64(prevUnverified)) * 100)
	}
	metrics.UnverifiedUsers.Trend = getTrend(metrics.UnverifiedUsers.Change)
}

// getUserBreakdown retrieves user breakdown by role and status
func (s *DashboardService) getUserBreakdown() (*UserBreakdown, error) {
	// Get total for percentage calculation
	var totalUsers int
	err := s.db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE deleted_at IS NULL").Scan(&totalUsers)
	if err != nil {
		return nil, err
	}

	// Role breakdown (join user_roles and roles tables)
	roleQuery := `
		SELECT COALESCE(r.name, 'user') as role, COUNT(DISTINCT u.id) as count
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id AND ur.is_active = true
		LEFT JOIN roles r ON ur.role_id = r.id AND r.is_active = true
		WHERE u.deleted_at IS NULL
		GROUP BY r.name
		ORDER BY count DESC
	`
	roleRows, err := s.db.DB.Query(roleQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query role breakdown: %w", err)
	}
	defer roleRows.Close()

	var roleBreakdown []RoleBreakdown
	for roleRows.Next() {
		var rb RoleBreakdown
		if err := roleRows.Scan(&rb.Role, &rb.Count); err != nil {
			return nil, err
		}
		if totalUsers > 0 {
			rb.Percentage = roundTo2((float64(rb.Count) / float64(totalUsers)) * 100)
		}
		roleBreakdown = append(roleBreakdown, rb)
	}

	// Status breakdown (based on is_active field)
	statusQuery := `
		SELECT
			CASE WHEN is_active = true THEN 'active' ELSE 'inactive' END as status,
			COUNT(*) as count
		FROM users
		WHERE deleted_at IS NULL
		GROUP BY is_active
		ORDER BY count DESC
	`
	statusRows, err := s.db.DB.Query(statusQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query status breakdown: %w", err)
	}
	defer statusRows.Close()

	var statusBreakdown []UserStatusBreakdown
	for statusRows.Next() {
		var sb UserStatusBreakdown
		if err := statusRows.Scan(&sb.Status, &sb.Count); err != nil {
			return nil, err
		}
		if totalUsers > 0 {
			sb.Percentage = roundTo2((float64(sb.Count) / float64(totalUsers)) * 100)
		}
		statusBreakdown = append(statusBreakdown, sb)
	}

	return &UserBreakdown{
		ByRole:   roleBreakdown,
		ByStatus: statusBreakdown,
	}, nil
}

func getTrend(change int) string {
	if change > 0 {
		return "up"
	} else if change < 0 {
		return "down"
	}
	return "neutral"
}
