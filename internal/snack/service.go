package snack

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

var (
	ErrNoVolunteer    = errors.New("no eligible volunteer")
	ErrNotFound       = errors.New("refill not found")
	ErrInvalidState   = errors.New("invalid refill state")
	ErrBudgetExceeded = errors.New("monthly budget exceeded")
	ErrInvalidAmount  = errors.New("amount must be positive")
)

type Service struct {
	mu            sync.Mutex
	volunteers    map[string]*Volunteer
	refills       map[string]*Refill
	budgets       map[Office]*Budget
	chat          Chat
	hr            HR
	nextID        int
	assignmentTTL time.Duration
}

func DefaultBudgets() []Budget {
	return []Budget{
		{Office: Oslo, Currency: NOK, MonthlyLimitMinor: 600000},
		{Office: Bengaluru, Currency: INR, MonthlyLimitMinor: 1500000},
	}
}

func NewService(chat Chat, hr HR, budgets []Budget) *Service {
	s := &Service{
		volunteers: make(map[string]*Volunteer), refills: make(map[string]*Refill),
		budgets: make(map[Office]*Budget), chat: chat, hr: hr,
		assignmentTTL: 24 * time.Hour,
	}
	for _, b := range budgets {
		copy := b
		s.budgets[b.Office] = &copy
	}
	return s
}

func (s *Service) AddVolunteer(v Volunteer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := v
	if copy.PayrollCurrency == "" {
		if v.Office == Oslo {
			copy.PayrollCurrency = NOK
		} else {
			copy.PayrollCurrency = INR
		}
	}
	s.volunteers[v.ID] = &copy
}

// ReportEmpty creates one refill and immediately assigns one eligible volunteer.
// The service mutex makes creation + selection + assignment a single atomic operation.
func (s *Service) ReportEmpty(office Office, item, reporter string, now time.Time) (Refill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.budgets[office]; !ok {
		return Refill{}, errors.New("unknown office")
	}
	s.nextID++
	r := &Refill{ID: fmt.Sprintf("REF-%04d", s.nextID), Office: office, Item: item,
		ReportedBy: reporter, Status: EmptyReported, CreatedAt: now}
	s.refills[r.ID] = r
	if err := s.assignLocked(r, now); err != nil {
		return *r, err
	}
	return *r, nil
}

func (s *Service) assignLocked(r *Refill, now time.Time) error {
	candidates := make([]*Volunteer, 0)
	for _, v := range s.volunteers {
		if v.Office == r.Office && v.Active && v.Available && !s.hasOpenAssignmentLocked(v.ID) {
			candidates = append(candidates, v)
		}
	}
	if len(candidates) == 0 {
		return ErrNoVolunteer
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].CompletedRefills != candidates[j].CompletedRefills {
			return candidates[i].CompletedRefills < candidates[j].CompletedRefills
		}
		if !candidates[i].LastAssignedAt.Equal(candidates[j].LastAssignedAt) {
			if candidates[i].LastAssignedAt.IsZero() {
				return true
			}
			if candidates[j].LastAssignedAt.IsZero() {
				return false
			}
			return candidates[i].LastAssignedAt.Before(candidates[j].LastAssignedAt)
		}
		return candidates[i].ID < candidates[j].ID
	})
	v := candidates[0]
	r.AssignedTo = v.ID
	r.Status = Assigned
	r.AssignmentExpiresAt = now.Add(s.assignmentTTL)
	v.LastAssignedAt = now
	if s.chat != nil {
		_ = s.chat.Notify(v.ID, "Please refill "+r.Item+" for "+string(r.Office))
	}
	return nil
}

func (s *Service) hasOpenAssignmentLocked(volunteerID string) bool {
	for _, r := range s.refills {
		if r.AssignedTo == volunteerID && (r.Status == Assigned || r.Status == PurchaseConfirmed || r.Status == Purchased) {
			return true
		}
	}
	return false
}

func (s *Service) ConfirmPurchase(refillID, volunteerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.refills[refillID]
	if !ok {
		return ErrNotFound
	}
	if r.Status != Assigned || r.AssignedTo != volunteerID {
		return ErrInvalidState
	}
	r.Status = PurchaseConfirmed
	return nil
}

// RecordPurchase reserves the approved estimate against the office/month budget.
// Amounts are integer minor units: øre for NOK and paise for INR.
func (s *Service) RecordPurchase(refillID, volunteerID string, amountMinor int64, currency Currency, receiptURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if amountMinor <= 0 {
		return ErrInvalidAmount
	}
	r, ok := s.refills[refillID]
	if !ok {
		return ErrNotFound
	}
	if r.Status != PurchaseConfirmed || r.AssignedTo != volunteerID {
		return ErrInvalidState
	}
	b := s.budgets[r.Office]
	if b.Currency != currency {
		return errors.New("currency does not match office budget")
	}
	if b.SpentMinor+b.ReservedMinor+amountMinor > b.MonthlyLimitMinor {
		return ErrBudgetExceeded
	}
	b.ReservedMinor += amountMinor
	r.PurchaseAmountMinor, r.Currency, r.ReceiptURL = amountMinor, currency, receiptURL
	r.Status = Purchased
	return nil
}

func (s *Service) ConfirmRefilled(refillID, volunteerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.refills[refillID]
	if !ok {
		return ErrNotFound
	}
	if r.Status != Purchased || r.AssignedTo != volunteerID {
		return ErrInvalidState
	}
	r.Status = Refilled
	return nil
}

// SubmitReimbursement is safe to retry: the refill ID is the HR idempotency key.
func (s *Service) SubmitReimbursement(refillID string) error {
	s.mu.Lock()
	r, ok := s.refills[refillID]
	if !ok {
		s.mu.Unlock()
		return ErrNotFound
	}
	if r.Status == ReimbursementSubmitted || r.Status == Reimbursed {
		s.mu.Unlock()
		return nil
	}
	if r.Status != Refilled {
		s.mu.Unlock()
		return ErrInvalidState
	}
	employeeID, amount, currency, receipt, office := r.AssignedTo, r.PurchaseAmountMinor, r.Currency, r.ReceiptURL, r.Office
	s.mu.Unlock()

	claimID, err := s.hr.SubmitClaim(refillID, employeeID, amount, currency, "Snack refill "+string(office), receipt)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	r = s.refills[refillID]
	if r.Status == Refilled {
		r.HRClaimID = claimID
		r.Status = ReimbursementSubmitted
	}
	return nil
}

func (s *Service) MarkReimbursed(refillID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.refills[refillID]
	if !ok {
		return ErrNotFound
	}
	if r.Status != ReimbursementSubmitted {
		return ErrInvalidState
	}
	b := s.budgets[r.Office]
	b.ReservedMinor -= r.PurchaseAmountMinor
	b.SpentMinor += r.PurchaseAmountMinor
	r.Status = Reimbursed
	if v := s.volunteers[r.AssignedTo]; v != nil {
		v.CompletedRefills++
	}
	return nil
}

// ExpireAndReassign hands expired, unfinished assignments to the next eligible volunteer.
func (s *Service) ExpireAndReassign(now time.Time) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := []string{}
	for _, r := range s.refills {
		if r.Status == Assigned && !now.Before(r.AssignmentExpiresAt) {
			r.Status = Expired
			r.AssignedTo = ""
			if err := s.assignLocked(r, now); err == nil {
				changed = append(changed, r.ID)
			}
		}
	}
	return changed
}

func (s *Service) GetRefill(id string) (Refill, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.refills[id]
	if !ok {
		return Refill{}, false
	}
	return *r, true
}

func (s *Service) MonthlyTotals() []SpendTotal {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]SpendTotal, 0, len(s.budgets))
	for _, b := range s.budgets {
		out = append(out, SpendTotal{Office: b.Office, Currency: b.Currency, SpentMinor: b.SpentMinor, ReservedMinor: b.ReservedMinor})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Office < out[j].Office })
	return out
}
