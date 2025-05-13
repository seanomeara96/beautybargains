package main

import (
	"database/sql"
	"time"
)

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

func (s *Service) GetSubscribers() ([]*Subscriber, error) {

	rows, err := s.db.Query(`
		SELECT
			id,
			email,
			full_name,
			consent,
			signup_date,
			verification_token,
			is_verified,
			preferences
		FROM
			subscribers
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscribers []*Subscriber
	for rows.Next() {
		var s Subscriber
		if err := rows.Scan(
			&s.ID,
			&s.Email,
			&s.FullName,
			&s.Consent,
			&s.SignupDate,
			&s.VerificationToken,
			&s.IsVerified,
			&s.Preferences,
		); err != nil {
			return nil, err
		}
		subscribers = append(subscribers, &s)
	}

	return subscribers, nil
}
