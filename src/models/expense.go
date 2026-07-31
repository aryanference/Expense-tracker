package models

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"time"
)

type Expense struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Amount   float64 `json:"amount"`
	Category string  `json:"category"`
	Date     string  `json:"date"`
}

type CreateExpenseRequest struct {
	Title    string  `json:"title"`
	Amount   float64 `json:"amount"`
	Category string  `json:"category"`
	Date     string  `json:"date"`
}

// IDs use crypto/rand and 32 hexadecimal characters, avoiding an extra dependency.
func NewID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func ValidateCreateRequest(req CreateExpenseRequest) (CreateExpenseRequest, error) {
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		return req, errors.New("title is required")
	}
	if len([]rune(req.Title)) > 200 {
		return req, errors.New("title must be 200 characters or fewer")
	}
	if req.Amount <= 0 || math.IsNaN(req.Amount) || math.IsInf(req.Amount, 0) {
		return req, errors.New("amount must be greater than 0")
	}
	req.Category = strings.TrimSpace(req.Category)
	if req.Category == "" {
		return req, errors.New("category is required")
	}
	if len([]rune(req.Category)) > 50 {
		return req, errors.New("category must be 50 characters or fewer")
	}
	if _, err := time.Parse("2006-01-02", req.Date); err != nil {
		return req, errors.New("date must be in YYYY-MM-DD format")
	}
	return req, nil
}
