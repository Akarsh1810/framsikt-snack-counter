# Snack Counter Automation — Design

## Goal and scope
Automate the refill workflow for Oslo and Bengaluru while preventing duplicate work, distributing volunteer effort fairly, controlling office budgets and tracking reimbursement. The first version is a small modular backend, not a distributed system.

## Workflow
1. An employee reports an empty item through the company chat. The backend creates a refill with office, item, reporter and timestamp.
2. The assignment service selects an active, available volunteer belonging to that refill's office. It prioritizes the fewest completed refills, then the oldest assignment timestamp, then employee ID for deterministic tie-breaking. Assignment creation and selection are atomic.
3. The selected volunteer receives a chat notification and confirms responsibility. An assignment expires after a configurable 24-hour acceptance window; an expired uncompleted task is offered to the next eligible volunteer.
4. The volunteer records the purchase amount, currency and receipt reference. The budget service checks the office's monthly limit and reserves the amount before accepting the purchase.
5. The volunteer confirms the counter is refilled. The backend submits a reimbursement claim to the external HR system using employee ID, amount, payroll currency, description and receipt reference. The refill ID is the idempotency key, so retries do not create duplicate claims in the mock integration.
6. HR approval is asynchronous. The backend tracks the submitted claim; when HR confirms reimbursement, the reservation moves to actual spend and the volunteer's completed-refill count increments.

## Components and data
- **Chat adapter:** sends assignment and lifecycle notifications. It does not own workflow state.
- **Refill service:** owns refill ID, office, item, reporter, status, assignee, timestamps, purchase details and HR claim ID.
- **Assignment service:** reads volunteer active/availability flags, office, completed-refill count and last-assigned time. One volunteer cannot hold multiple open assignments.
- **Budget service:** maintains office, currency, monthly limit, reserved amount and actual spend.
- **HR adapter:** accepts employee ID, amount, currency, description and receipt reference; returns a claim ID. Production integration must handle asynchronous approval and retries.
- **Reporting:** emits monthly totals separately for each office and currency.

## Rules and failure handling
The monthly budgets are 6,000 NOK for Oslo and 15,000 INR for Bengaluru. Money is stored as integer minor units, never floating point. NOK and INR totals are never added together; a consolidated management report can show both currency totals, with FX conversion deferred until a trusted rate source and reporting policy are agreed. Employees visiting another office are handled by separating the refill office from the employee's payroll currency in a production data model; this prototype keeps office budget currency explicit and uses the assigned employee ID for the HR claim.

Atomic assignment prevents two volunteers from committing to the same refill. Budget reservation prevents concurrent purchases from overspending the same monthly limit. HR submission is retryable and idempotent. A real deployment should use database transactions/constraints and a durable outbox for external notifications and HR requests.

## Trade-offs and deliberate omissions
The prototype uses in-memory storage and mutexes to keep the core rules easy to inspect and test. This is not durable or multi-instance safe; production should use PostgreSQL transactions, unique constraints and row-level locking. Chat and HR are simulated because their vendor APIs are outside the assignment scope. No frontend, mobile app, FX conversion, automated snack forecasting, receipt OCR, complex approval hierarchy or microservices are included. The goal is to prove the lifecycle, fairness, concurrency and budget rules before adding integrations.
