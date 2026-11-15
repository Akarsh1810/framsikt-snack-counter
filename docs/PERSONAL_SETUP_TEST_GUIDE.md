# Personal Setup & Testing Guide

For your own preparation; do not attach unless requested.

## Requirements
- Go 1.22 or newer.

## Run
```bash
go version
go run ./cmd/snackcounter
```

## Test
```bash
go test ./...
go test -race ./...
```

## Before submitting
1. Run both test commands.
2. Review README and design document.
3. Push the project to GitHub.
4. Verify repository access.
5. Attach `Framsikt_Snack_Counter_Design_Document.pdf` to the submission email.

## Explain honestly
This is an in-memory assignment prototype. PostgreSQL durability, real chat/HR APIs, FX conversion and production retry infrastructure are intentionally not implemented.
