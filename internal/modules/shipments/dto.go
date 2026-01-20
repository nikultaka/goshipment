package shipments

import (
	"database/sql"
	"gogin/internal/models"
)

// CreateShipmentRequest represents the request to create a shipment from an estimate
type CreateShipmentRequest struct {
	EstimateID           string  `json:"estimate_id" binding:"required"`
	SelectedCarrierID    string  `json:"selected_carrier_id" binding:"required"`
	SelectedServiceCode  string  `json:"selected_service_code" binding:"required"`
	RequireSignature     *bool   `json:"require_signature,omitempty"`
	EmailNotifications   *bool   `json:"email_notifications,omitempty"`
	DeliveryInstructions *string `json:"delivery_instructions,omitempty"`
	PickupRequired       *bool   `json:"pickup_required,omitempty"`
}

// CancelShipmentRequest represents the request to cancel a shipment
type CancelShipmentRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// VoidShipmentRequest represents the request to void a shipment
type VoidShipmentRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// ArchiveShipmentRequest represents the request to archive a shipment
type ArchiveShipmentRequest struct {
	// No fields needed - archival is a simple toggle
}

// UpdateStatusRequest represents the request to update shipment status
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=draft awaiting_payment payment_failed label_purchased ready_for_pickup pickup_scheduled in_transit out_for_delivery delivered delivery_failed returned cancelled refunded voided" example:"delivered"`
	Reason string `json:"reason,omitempty" example:"Package delivered to recipient"`
}

// ValidStatusTransitions defines which status transitions are allowed
// Key: current status, Value: list of allowed next statuses
var ValidStatusTransitions = map[string][]string{
	// Pre-Label States
	"draft":            {"awaiting_payment", "cancelled"},
	"awaiting_payment": {"payment_failed", "label_purchased", "cancelled"},
	"payment_failed":   {"awaiting_payment", "cancelled"},

	// Label & Pickup States
	"label_purchased":  {"ready_for_pickup", "pickup_scheduled", "in_transit", "voided", "cancelled"},
	"ready_for_pickup": {"pickup_scheduled", "in_transit", "voided", "cancelled"},
	"pickup_scheduled": {"in_transit", "voided", "cancelled"},

	// In-Transit States
	"in_transit":       {"out_for_delivery", "delivered", "delivery_failed", "returned"},
	"out_for_delivery": {"delivered", "delivery_failed", "returned"},

	// Delivery Outcome States
	"delivery_failed": {"in_transit", "out_for_delivery", "returned"},
	"returned":        {}, // Terminal state

	// Final States (no transitions allowed)
	"delivered":  {}, // Terminal state - triggers commission
	"cancelled":  {}, // Terminal state
	"refunded":   {}, // Terminal state
	"voided":     {}, // Terminal state
}

// IsValidTransition checks if a status transition is allowed
func IsValidTransition(currentStatus, newStatus string) bool {
	allowedStatuses, exists := ValidStatusTransitions[currentStatus]
	if !exists {
		return false
	}
	for _, allowed := range allowedStatuses {
		if allowed == newStatus {
			return true
		}
	}
	return false
}

// GetAllowedTransitions returns the list of allowed next statuses for a given status
func GetAllowedTransitions(currentStatus string) []string {
	if transitions, exists := ValidStatusTransitions[currentStatus]; exists {
		return transitions
	}
	return []string{}
}

// UpdateStatusResponse represents the response after status update
type UpdateStatusResponse struct {
	ShipmentID        string   `json:"shipment_id"`
	ShipmentNumber    string   `json:"shipment_number"`
	PreviousStatus    string   `json:"previous_status"`
	CurrentStatus     string   `json:"current_status"`
	StatusLabel       string   `json:"status_label"`
	AllowedTransitions []string `json:"allowed_transitions"`
	CommissionCreated bool     `json:"commission_created,omitempty"`
	UpdatedAt         string   `json:"updated_at"`
}

// ShipmentResponse represents a shipment in API responses
type ShipmentResponse struct {
	ID                 string                 `json:"id"`
	ShipmentNumber     string                 `json:"shipment_number"`
	OrganizationID     string                 `json:"organization_id"`
	CreatedByUserID    string                 `json:"created_by_user_id"`
	EstimateID         string                 `json:"estimate_id"`
	Carrier            CarrierInfo            `json:"carrier"`
	FromAddress        AddressSnapshot        `json:"from_address"`
	ToAddress          AddressSnapshot        `json:"to_address"`
	Packages           []PackageResponse      `json:"packages,omitempty"`
	DeliveryOptions    DeliveryOptions        `json:"delivery_options"`
	Pricing            PricingInfo            `json:"pricing"`
	Payment            PaymentInfo            `json:"payment"`
	Status             ShipmentStatusInfo     `json:"status"`
	Tracking           *TrackingInfo          `json:"tracking,omitempty"`
	Pickup             *PickupInfo            `json:"pickup,omitempty"`
	Timeline           []EventResponse        `json:"timeline,omitempty"`
	IsArchived         bool                   `json:"is_archived"`
	ArchivedAt         *string                `json:"archived_at,omitempty"`
	CreatedAt          string                 `json:"created_at"`
	UpdatedAt          string                 `json:"updated_at"`
}

// ShipmentListItem represents a shipment in list responses
type ShipmentListItem struct {
	ID              string             `json:"id"`
	ShipmentNumber  string             `json:"shipment_number"`
	Carrier         CarrierInfo        `json:"carrier"`
	FromCity        string             `json:"from_city"`
	FromRegion      string             `json:"from_region,omitempty"`
	FromCountry     string             `json:"from_country"`
	ToCity          string             `json:"to_city"`
	ToRegion        string             `json:"to_region,omitempty"`
	ToCountry       string             `json:"to_country"`
	TotalPackages   int                `json:"total_packages"`
	TotalAmount     float64            `json:"total_amount"`
	Currency        string             `json:"currency"`
	Status          ShipmentStatusInfo `json:"status"`
	PaymentStatus   string             `json:"payment_status"`
	TrackingNumber  *string            `json:"tracking_number,omitempty"`
	IsArchived      bool               `json:"is_archived"`
	CreatedAt       string             `json:"created_at"`
}

// CarrierInfo represents carrier details
type CarrierInfo struct {
	ID            string  `json:"id"`
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	ServiceCode   *string `json:"service_code,omitempty"`
	ServiceName   *string `json:"service_name,omitempty"`
	ShipmentID    *string `json:"shipment_id,omitempty"`
	RateID        *string `json:"rate_id,omitempty"`
	PurchaseID    *string `json:"purchase_id,omitempty"`
}

// AddressSnapshot represents a snapshotted address
type AddressSnapshot struct {
	Name         *string `json:"name,omitempty"`
	Company      *string `json:"company,omitempty"`
	Phone        *string `json:"phone,omitempty"`
	Email        *string `json:"email,omitempty"`
	Country      string  `json:"country"`
	RegionCode   *string `json:"region_code,omitempty"`
	City         string  `json:"city"`
	PostalCode   string  `json:"postal_code"`
	AddressLine1 string  `json:"address_line1"`
	AddressLine2 *string `json:"address_line2,omitempty"`
}

// PackageResponse represents a package in responses
type PackageResponse struct {
	ID             string  `json:"id"`
	PackageNumber  string  `json:"package_number"`
	PackageIndex   int     `json:"package_index"`
	PackageName    *string `json:"package_name,omitempty"`
	Length         float64 `json:"length"`
	Width          float64 `json:"width"`
	Height         float64 `json:"height"`
	DimensionUnit  string  `json:"dimension_unit"`
	Weight         float64 `json:"weight"`
	WeightUnit     string  `json:"weight_unit"`
	DeclaredValue  *float64 `json:"declared_value,omitempty"`
	TrackingNumber *string `json:"tracking_number,omitempty"`
	TrackingURL    *string `json:"tracking_url,omitempty"`
	LabelURL       *string `json:"label_url,omitempty"`
	Status         string  `json:"status"`
}

// DeliveryOptions represents delivery preferences
type DeliveryOptions struct {
	RequireSignature     bool    `json:"require_signature"`
	EmailNotifications   bool    `json:"email_notifications"`
	DeliveryInstructions *string `json:"delivery_instructions,omitempty"`
}

// PricingInfo represents pricing details
type PricingInfo struct {
	BaseAmount          float64  `json:"base_amount"`
	InsuranceAmount     *float64 `json:"insurance_amount,omitempty"`
	PickupFeeAmount     *float64 `json:"pickup_fee_amount,omitempty"`
	TaxesAmount         *float64 `json:"taxes_amount,omitempty"`
	AdjustmentsTotal    float64  `json:"adjustments_total"`
	TotalAmount         float64  `json:"total_amount"`
	Currency            string   `json:"currency"`
	RetailRate          *float64 `json:"retail_rate,omitempty"`
	RetailCurrency      *string  `json:"retail_currency,omitempty"`
	HasPendingAdjustments bool   `json:"has_pending_adjustments"`
}

// PaymentInfo represents payment details
type PaymentInfo struct {
	Status              string  `json:"status"`
	WalletTransactionID *string `json:"wallet_transaction_id,omitempty"`
	StripePaymentIntentID *string `json:"stripe_payment_intent_id,omitempty"`
}

// ShipmentStatusInfo represents status details
type ShipmentStatusInfo struct {
	Status        string  `json:"status"`
	StatusLabel   string  `json:"status_label"`
	FailureReason *string `json:"failure_reason,omitempty"`
	CanCancel     bool    `json:"can_cancel"`
	CanVoid       bool    `json:"can_void"`
	CanArchive    bool    `json:"can_archive"`
	IsTerminal    bool    `json:"is_terminal"`
}

// TrackingInfo represents tracking details
type TrackingInfo struct {
	MasterTrackingNumber *string `json:"master_tracking_number,omitempty"`
	MasterTrackingURL    *string `json:"master_tracking_url,omitempty"`
}

// PickupInfo represents pickup details
type PickupInfo struct {
	Required bool    `json:"required"`
	PickupID *string `json:"pickup_id,omitempty"`
}

// EventResponse represents a shipment event
type EventResponse struct {
	ID         string  `json:"id"`
	EventKey   string  `json:"event_key"`
	EntityType string  `json:"entity_type"`
	Source     string  `json:"source"`
	Severity   string  `json:"severity"`
	Title      *string `json:"title,omitempty"`
	Message    *string `json:"message,omitempty"`
	OccurredAt string  `json:"occurred_at"`
	CreatedAt  string  `json:"created_at"`
}

// ListShipmentsQuery represents query parameters for listing shipments
type ListShipmentsQuery struct {
	Page          int     `form:"page" binding:"omitempty,min=1"`
	Limit         int     `form:"limit" binding:"omitempty,min=1,max=100"`
	Status        *string `form:"status" binding:"omitempty"`
	PaymentStatus *string `form:"payment_status" binding:"omitempty"`
	CarrierID     *string `form:"carrier_id" binding:"omitempty"`
	SearchQuery   *string `form:"q" binding:"omitempty"`
	IsArchived    *bool   `form:"is_archived" binding:"omitempty"`
	SortBy        *string `form:"sort_by" binding:"omitempty,oneof=created_at updated_at shipment_number"`
	SortOrder     *string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// Helper functions to convert models to DTOs

// ToShipmentResponse converts a Shipment model to ShipmentResponse DTO
func ToShipmentResponse(
	shipment *models.Shipment,
	packages []*models.ShipmentPackage,
	events []*models.ShipmentEvent,
) ShipmentResponse {
	resp := ShipmentResponse{
		ID:              shipment.ID,
		ShipmentNumber:  shipment.ShipmentNumber,
		OrganizationID:  shipment.OrganizationID,
		CreatedByUserID: shipment.CreatedByUserID,
		EstimateID:      shipment.ShipmentEstimateID,
		Carrier: CarrierInfo{
			ID:          shipment.CarrierID,
			ServiceCode: nullStringToPtr(shipment.ServicelevelCode),
			ServiceName: nullStringToPtr(shipment.ServicelevelName),
			ShipmentID:  nullStringToPtr(shipment.CarrierShipmentID),
			RateID:      nullStringToPtr(shipment.CarrierRateID),
			PurchaseID:  nullStringToPtr(shipment.CarrierPurchaseID),
		},
		FromAddress: AddressSnapshot{
			Name:         nullStringToPtr(shipment.FromName),
			Company:      nullStringToPtr(shipment.FromCompany),
			Phone:        nullStringToPtr(shipment.FromPhone),
			Email:        nullStringToPtr(shipment.FromEmail),
			Country:      shipment.FromCountry,
			RegionCode:   nullStringToPtr(shipment.FromRegionCode),
			City:         shipment.FromCity,
			PostalCode:   shipment.FromPostalCode,
			AddressLine1: shipment.FromAddressLine1,
			AddressLine2: nullStringToPtr(shipment.FromAddressLine2),
		},
		ToAddress: AddressSnapshot{
			Name:         nullStringToPtr(shipment.ToName),
			Company:      nullStringToPtr(shipment.ToCompany),
			Phone:        nullStringToPtr(shipment.ToPhone),
			Email:        nullStringToPtr(shipment.ToEmail),
			Country:      shipment.ToCountry,
			RegionCode:   nullStringToPtr(shipment.ToRegionCode),
			City:         shipment.ToCity,
			PostalCode:   shipment.ToPostalCode,
			AddressLine1: shipment.ToAddressLine1,
			AddressLine2: nullStringToPtr(shipment.ToAddressLine2),
		},
		DeliveryOptions: DeliveryOptions{
			RequireSignature:     shipment.RequireSignature,
			EmailNotifications:   shipment.EmailNotifications,
			DeliveryInstructions: nullStringToPtr(shipment.DeliveryInstructions),
		},
		Pricing: PricingInfo{
			BaseAmount:            shipment.BaseAmount,
			InsuranceAmount:       nullFloat64ToPtr(shipment.InsuranceAmount),
			PickupFeeAmount:       nullFloat64ToPtr(shipment.PickupFeeAmount),
			TaxesAmount:           nullFloat64ToPtr(shipment.TaxesAmount),
			AdjustmentsTotal:      shipment.AdjustmentsTotal,
			TotalAmount:           shipment.GetTotalAmount(),
			Currency:              shipment.Currency,
			RetailRate:            nullFloat64ToPtr(shipment.RetailRate),
			RetailCurrency:        nullStringToPtr(shipment.RetailCurrency),
			HasPendingAdjustments: shipment.HasPendingAdjustments,
		},
		Payment: PaymentInfo{
			Status:                string(shipment.PaymentStatus),
			WalletTransactionID:   nullStringToPtr(shipment.WalletTransactionID),
			StripePaymentIntentID: nullStringToPtr(shipment.StripePaymentIntentID),
		},
		Status: ShipmentStatusInfo{
			Status:        string(shipment.ShipmentStatus),
			StatusLabel:   getStatusLabel(shipment.ShipmentStatus),
			FailureReason: nullStringToPtr(shipment.FailureReason),
			CanCancel:     shipment.CanBeCancelled(),
			CanVoid:       shipment.CanBeVoided(),
			CanArchive:    shipment.CanBeArchived(),
			IsTerminal:    shipment.IsTerminal(),
		},
		IsArchived: shipment.IsArchived,
		ArchivedAt: nullTimeToStringPtr(shipment.ArchivedAt),
		CreatedAt:  shipment.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  shipment.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Add tracking if present
	if shipment.MasterTrackingNumber.Valid || shipment.MasterTrackingURL.Valid {
		resp.Tracking = &TrackingInfo{
			MasterTrackingNumber: nullStringToPtr(shipment.MasterTrackingNumber),
			MasterTrackingURL:    nullStringToPtr(shipment.MasterTrackingURL),
		}
	}

	// Add pickup if required
	if shipment.PickupRequired {
		resp.Pickup = &PickupInfo{
			Required: true,
			PickupID: nullStringToPtr(shipment.PickupID),
		}
	}

	// Convert packages
	if len(packages) > 0 {
		resp.Packages = make([]PackageResponse, len(packages))
		for i, pkg := range packages {
			resp.Packages[i] = toPackageResponse(pkg)
		}
	}

	// Convert events
	if len(events) > 0 {
		resp.Timeline = make([]EventResponse, len(events))
		for i, event := range events {
			resp.Timeline[i] = toEventResponse(event)
		}
	}

	return resp
}

// ToShipmentListItem converts a Shipment model to ShipmentListItem DTO
func ToShipmentListItem(shipment *models.Shipment) ShipmentListItem {
	return ShipmentListItem{
		ID:             shipment.ID,
		ShipmentNumber: shipment.ShipmentNumber,
		Carrier: CarrierInfo{
			ID:          shipment.CarrierID,
			ServiceCode: nullStringToPtr(shipment.ServicelevelCode),
			ServiceName: nullStringToPtr(shipment.ServicelevelName),
		},
		FromCity:    shipment.FromCity,
		FromRegion:  nullStringToString(shipment.FromRegionCode),
		FromCountry: shipment.FromCountry,
		ToCity:      shipment.ToCity,
		ToRegion:    nullStringToString(shipment.ToRegionCode),
		ToCountry:   shipment.ToCountry,
		TotalPackages: shipment.TotalPackages,
		TotalAmount:   shipment.GetTotalAmount(),
		Currency:      shipment.Currency,
		Status: ShipmentStatusInfo{
			Status:      string(shipment.ShipmentStatus),
			StatusLabel: getStatusLabel(shipment.ShipmentStatus),
			CanCancel:   shipment.CanBeCancelled(),
			CanVoid:     shipment.CanBeVoided(),
			CanArchive:  shipment.CanBeArchived(),
			IsTerminal:  shipment.IsTerminal(),
		},
		PaymentStatus:  string(shipment.PaymentStatus),
		TrackingNumber: nullStringToPtr(shipment.MasterTrackingNumber),
		IsArchived:     shipment.IsArchived,
		CreatedAt:      shipment.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toPackageResponse(pkg *models.ShipmentPackage) PackageResponse {
	packageNumber := ""
	if pkg.PackageNumber.Valid {
		packageNumber = pkg.PackageNumber.String
	}

	resp := PackageResponse{
		ID:             pkg.ID,
		PackageNumber:  packageNumber,
		PackageIndex:   pkg.PackageIndex,
		PackageName:    nullStringToPtr(pkg.PackageName),
		Length:         pkg.Length,
		Width:          pkg.Width,
		Height:         pkg.Height,
		DimensionUnit:  pkg.DimensionUnit,
		Weight:         pkg.Weight,
		WeightUnit:     pkg.WeightUnit,
		DeclaredValue:  nullFloat64ToPtr(pkg.DeclaredValue),
		TrackingNumber: nullStringToPtr(pkg.TrackingNumber),
		TrackingURL:    nullStringToPtr(pkg.TrackingURL),
		LabelURL:       nullStringToPtr(pkg.LabelURL),
		Status:         pkg.PackageStatus,
	}
	return resp
}

func toEventResponse(event *models.ShipmentEvent) EventResponse {
	return EventResponse{
		ID:         event.ID,
		EventKey:   event.EventKey,
		EntityType: event.EntityType,
		Source:     event.Source,
		Severity:   event.Severity,
		Title:      nullStringToPtr(event.Title),
		Message:    nullStringToPtr(event.Message),
		OccurredAt: event.OccurredAt.Format("2006-01-02T15:04:05Z07:00"),
		CreatedAt:  event.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// Helper functions for null conversions

func nullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func nullFloat64ToPtr(nf sql.NullFloat64) *float64 {
	if nf.Valid {
		return &nf.Float64
	}
	return nil
}

func nullTimeToStringPtr(nt sql.NullTime) *string {
	if nt.Valid {
		str := nt.Time.Format("2006-01-02T15:04:05Z07:00")
		return &str
	}
	return nil
}

func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func getStatusLabel(status models.ShipmentStatus) string {
	labels := map[models.ShipmentStatus]string{
		models.ShipmentStatusDraft:            "Draft",
		models.ShipmentStatusAwaitingPayment:  "Awaiting Payment",
		models.ShipmentStatusPaymentFailed:    "Payment Failed",
		models.ShipmentStatusLabelPurchased:   "Label Purchased",
		models.ShipmentStatusReadyForPickup:   "Ready for Pickup",
		models.ShipmentStatusPickupScheduled:  "Pickup Scheduled",
		models.ShipmentStatusInTransit:        "In Transit",
		models.ShipmentStatusOutForDelivery:   "Out for Delivery",
		models.ShipmentStatusDelivered:        "Delivered",
		models.ShipmentStatusDeliveryFailed:   "Delivery Failed",
		models.ShipmentStatusReturned:         "Returned to Sender",
		models.ShipmentStatusCancelled:        "Cancelled",
		models.ShipmentStatusRefunded:         "Refunded",
		models.ShipmentStatusVoided:           "Voided",
	}
	if label, ok := labels[status]; ok {
		return label
	}
	return string(status)
}
