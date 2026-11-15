package snack

import (
	"errors"
	"fmt"
	"sync"
)

type Chat interface {
	Notify(volunteerID, message string) error
}

type HR interface {
	SubmitClaim(idempotencyKey, employeeID string, amountMinor int64, currency Currency, description, receiptURL string) (string, error)
}

type MockChat struct {
	mu       sync.Mutex
	Messages []string
	Fail     bool
}

func (c *MockChat) Notify(volunteerID, message string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Fail {
		return errors.New("simulated chat failure")
	}
	c.Messages = append(c.Messages, fmt.Sprintf("%s: %s", volunteerID, message))
	return nil
}

type MockHR struct {
	mu     sync.Mutex
	claims map[string]string
	Fail   bool
}

func NewMockHR() *MockHR { return &MockHR{claims: make(map[string]string)} }

func (h *MockHR) SubmitClaim(key, employeeID string, amount int64, currency Currency, description, receiptURL string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.Fail {
		return "", errors.New("simulated HR outage")
	}
	if id, ok := h.claims[key]; ok {
		return id, nil
	}
	if employeeID == "" || amount <= 0 || receiptURL == "" {
		return "", errors.New("invalid claim details")
	}
	id := "HR-" + key
	h.claims[key] = id
	return id, nil
}
