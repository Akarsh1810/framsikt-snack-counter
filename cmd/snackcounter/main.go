package main

import (
	"fmt"
	"time"

	"framsikt-snack-counter/internal/snack"
)

func main() {
	chat := &snack.MockChat{}
	hr := snack.NewMockHR()
	service := snack.NewService(chat, hr, snack.DefaultBudgets())

	service.AddVolunteer(snack.Volunteer{ID: "emp-101", Name: "Asha", Office: snack.Bengaluru, Active: true, Available: true})
	service.AddVolunteer(snack.Volunteer{ID: "emp-102", Name: "Rahul", Office: snack.Bengaluru, Active: true, Available: true})
	service.AddVolunteer(snack.Volunteer{ID: "emp-201", Name: "Nora", Office: snack.Oslo, Active: true, Available: true})

	refill, err := service.ReportEmpty(snack.Bengaluru, "biscuits", "emp-reporter", time.Now())
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created refill: %s (status: %s)\n", refill.ID, refill.Status)
	fmt.Printf("Chat notifications: %d\n", len(chat.Messages))
	fmt.Println("Run `go test -race ./...` to execute the automated tests.")
}
