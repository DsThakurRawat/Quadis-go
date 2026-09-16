package main

import (
	"encoding/json"
	"fmt"
	"os"
	"quadis-backend-go/internal/service/auth"
)

type HashOutput struct {
	Password string `json:"password"`
	Hash     string `json:"hash"`
}

func main() {
	passwords := []string{
		"HotelGuest#2026",
		"PinPass789",
		"Special@Char$Pass",
		"admin_secret_4444",
		"होटल-अमलतास-2026",
	}

	var results []HashOutput
	for _, p := range passwords {
		h, err := auth.HashPassword(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error hashing %s: %v\n", p, err)
			os.Exit(1)
		}
		results = append(results, HashOutput{Password: p, Hash: h})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(results); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}
