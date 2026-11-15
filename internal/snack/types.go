package snack

import "time"

type Office string

const (
	Oslo      Office = "OSLO"
	Bengaluru Office = "BENGALURU"
)

type Currency string

const (
	NOK Currency = "NOK"
	INR Currency = "INR"
)

type Status string

const (
	EmptyReported          Status = "EMPTY_REPORTED"
	Assigned               Status = "ASSIGNED"
	PurchaseConfirmed      Status = "PURCHASE_CONFIRMED"
	Purchased              Status = "PURCHASED"
	Refilled               Status = "REFILLED"
	ReimbursementSubmitted Status = "REIMBURSEMENT_SUBMITTED"
	Reimbursed             Status = "REIMBURSED"
	Expired                Status = "EXPIRED"
	Rejected               Status = "REJECTED"
)

type Volunteer struct {
	ID               string
	Name             string
	Office           Office
	Active           bool
	Available        bool
	PayrollCurrency  Currency
	CompletedRefills int
	LastAssignedAt   time.Time
}

type Refill struct {
	ID                  string
	Office              Office
	Item                string
	ReportedBy          string
	Status              Status
	AssignedTo          string
	CreatedAt           time.Time
	AssignmentExpiresAt time.Time
	PurchaseAmountMinor int64
	Currency            Currency
	ReceiptURL          string
	HRClaimID           string
	FailureReason       string
}

type Budget struct {
	Office            Office
	Currency          Currency
	MonthlyLimitMinor int64
	SpentMinor        int64
	ReservedMinor     int64
}

type SpendTotal struct {
	Office        Office
	Currency      Currency
	SpentMinor    int64
	ReservedMinor int64
}
