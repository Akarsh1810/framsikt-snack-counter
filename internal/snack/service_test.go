package snack

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func newTestService() (*Service, *MockChat, *MockHR) {
	chat, hr := &MockChat{}, NewMockHR()
	s := NewService(chat, hr, DefaultBudgets())
	s.AddVolunteer(Volunteer{ID: "a", Office: Bengaluru, Active: true, Available: true})
	s.AddVolunteer(Volunteer{ID: "b", Office: Bengaluru, Active: true, Available: true})
	s.AddVolunteer(Volunteer{ID: "oslo", Office: Oslo, Active: true, Available: true})
	return s, chat, hr
}

func TestOnlyOneVolunteerCanCommitToRefill(t *testing.T) {
	s, _, _ := newTestService()
	r, err := s.ReportEmpty(Bengaluru, "biscuits", "reporter", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	winner := r.AssignedTo
	loser := "a"
	if winner == "a" {
		loser = "b"
	}
	if err := s.ConfirmPurchase(r.ID, loser); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("loser should not confirm, got %v", err)
	}
	if err := s.ConfirmPurchase(r.ID, winner); err != nil {
		t.Fatal(err)
	}
}

func TestExpiredAssignmentIsReassigned(t *testing.T) {
	s, _, _ := newTestService()
	now := time.Now()
	r, err := s.ReportEmpty(Bengaluru, "fruit", "reporter", now)
	if err != nil {
		t.Fatal(err)
	}
	previous := r.AssignedTo
	changed := s.ExpireAndReassign(now.Add(25 * time.Hour))
	if len(changed) != 1 || changed[0] != r.ID {
		t.Fatalf("expected refill to be reassigned: %v", changed)
	}
	updated, _ := s.GetRefill(r.ID)
	if updated.AssignedTo == "" || updated.AssignedTo == previous {
		t.Fatalf("expected a different volunteer, old=%s new=%s", previous, updated.AssignedTo)
	}
}

func TestBudgetReservationAndCurrencySeparation(t *testing.T) {
	s, _, _ := newTestService()
	r, _ := s.ReportEmpty(Bengaluru, "coffee", "reporter", time.Now())
	if err := s.ConfirmPurchase(r.ID, r.AssignedTo); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordPurchase(r.ID, r.AssignedTo, 1499000, INR, "receipt.jpg"); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmRefilled(r.ID, r.AssignedTo); err != nil {
		t.Fatal(err)
	}
	if err := s.SubmitReimbursement(r.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkReimbursed(r.ID); err != nil {
		t.Fatal(err)
	}
	totals := s.MonthlyTotals()
	var inr, nok bool
	for _, total := range totals {
		if total.Office == Bengaluru {
			inr = total.Currency == INR && total.SpentMinor == 1499000
		}
		if total.Office == Oslo {
			nok = total.Currency == NOK && total.SpentMinor == 0
		}
	}
	if !inr || !nok {
		t.Fatalf("currency totals incorrect: %+v", totals)
	}
}

func TestBudgetExceeded(t *testing.T) {
	s, _, _ := newTestService()
	r, _ := s.ReportEmpty(Bengaluru, "snacks", "reporter", time.Now())
	_ = s.ConfirmPurchase(r.ID, r.AssignedTo)
	err := s.RecordPurchase(r.ID, r.AssignedTo, 1500001, INR, "receipt.jpg")
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("expected budget error, got %v", err)
	}
}

func TestHRFailureCanBeRetriedWithoutDuplicateClaim(t *testing.T) {
	s, _, hr := newTestService()
	r, _ := s.ReportEmpty(Bengaluru, "tea", "reporter", time.Now())
	_ = s.ConfirmPurchase(r.ID, r.AssignedTo)
	_ = s.RecordPurchase(r.ID, r.AssignedTo, 50000, INR, "receipt.jpg")
	_ = s.ConfirmRefilled(r.ID, r.AssignedTo)
	hr.Fail = true
	if err := s.SubmitReimbursement(r.ID); err == nil {
		t.Fatal("expected simulated HR failure")
	}
	hr.Fail = false
	if err := s.SubmitReimbursement(r.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.SubmitReimbursement(r.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetRefill(r.ID)
	if got.Status != ReimbursementSubmitted || got.HRClaimID == "" {
		t.Fatalf("claim not submitted: %+v", got)
	}
}

func TestConcurrentRefillCreationAssignsDistinctOpenTasks(t *testing.T) {
	s, _, _ := newTestService()
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.ReportEmpty(Bengaluru, "snacks", "reporter", time.Now()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	seen := map[string]bool{}
	s.mu.Lock()
	for _, r := range s.refills {
		if seen[r.AssignedTo] {
			s.mu.Unlock()
			t.Fatalf("volunteer %s assigned two open tasks", r.AssignedTo)
		}
		seen[r.AssignedTo] = true
	}
	s.mu.Unlock()
}
