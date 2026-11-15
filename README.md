# Snack Counter Automation

A focused Go implementation of the Framsikt take-home assignment.

## Scope
- Refill lifecycle from empty report through reimbursement.
- Fair volunteer selection (fewest completed refills, then oldest assignment, then ID).
- One open assignment per volunteer.
- Assignment expiry and reassignment.
- Office-specific budgets with separate NOK and INR totals.
- Budget reservation before purchase is accepted.
- Simulated chat and HR adapters; HR claim submission uses the refill ID as an idempotency key.
- Automated tests, including concurrent refill creation.

## Requirements
- Go 1.22 or newer.

## Run
```bash
go run ./cmd/snackcounter
```

## Test
```bash
go test ./...
go test -race ./...
```

## Money representation
Amounts are integer minor units (are for NOK and paise for INR). The configured limits are 600000 are (6,000 NOK) and 1500000 paise (15,000 INR). Totals remain separated by office/currency; no unsupported FX conversion is performed.

## Design boundaries
This is an assignment-sized in-memory reference implementation. It does not provide persistence across restarts, real chat/HR API clients, a user interface, FX conversion, receipt storage, or a production scheduler. A production version should persist state in a transactional database, use durable outbox/retry processing for external calls, and apply database uniqueness/locking constraints.

## Important lifecycle notes
- `ReportEmpty` creates and assigns a refill under a mutex.
- A volunteer must confirm before purchase details can be recorded.
- Budget is reserved when purchase amount is recorded; actual spend is moved from reserved to spent after reimbursement is confirmed.
- An HR outage leaves the refill in `REFILLED`, so submission can be retried.
