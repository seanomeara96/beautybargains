package main

import (
	"database/sql"
	"net/http"
	"time"
)

type Trending struct {
	Category  string
	Phrase    string
	PostCount int
}

type handleFunc func(w http.ResponseWriter, r *http.Request) error
type middleware func(next handleFunc) handleFunc

type Subscriber struct {
	ID                int            `json:"id"`                              // Primary key, auto-increment
	Email             string         `json:"email" validate:"required,email"` // Email, unique, not null
	FullName          sql.NullString `json:"full_name"`                       // Full name, can be null
	Consent           bool           `json:"consent" validate:"required"`     // Consent, 0 or 1, not null
	SignupDate        time.Time      `json:"signup_date"`                     // Signup date, defaults to current timestamp
	VerificationToken sql.NullString `json:"verification_token"`              // Unique verification token
	IsVerified        bool           `json:"is_verified" default:"false"`     // Verification status, defaults to false (0)
	Preferences       sql.NullString `json:"preferences"`                     // User preferences, can be null
}

/* models begin */
type Mode string
type contextKey string
