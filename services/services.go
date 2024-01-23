package services

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db}
}

/*
CREATE TABLE subscribers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    full_name TEXT,
    consent BOOLEAN NOT NULL CHECK (consent IN (0, 1)),
    signup_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    verification_token TEXT UNIQUE,
    is_verified BOOLEAN DEFAULT 0,
    preferences TEXT,
    CONSTRAINT chk_preferences CHECK (json_valid(preferences))
);
*/

func (s *Service) Subscribe(email string) error {
	q := `INSERT INTO subscribers(email, consent) VALUES (?, ?)`
	_, err := s.db.Exec(q, email, 1)
	if err != nil {
		return fmt.Errorf("could not insert email into subscibers table => %w", err)
	}

	token, err := s.GenerateToken(20)
	if err != nil {
		return fmt.Errorf("failed to generate verification token at service.Subscribe => %w", err)
	}

	err = s.AddVerificationToken(email, token)
	if err != nil {
		return fmt.Errorf("failed to add verification to subscriber record => %w", err)
	}

	log.Printf("Verify subscription at https://develop.implicitdev.com/subscribe/verify?token=%s", token)
	/* TODO implement this in production. For now log to console
	err = s.SendVerificationToken(email, token)
		if err != nil {
			return fmt.Errorf("failed to send verification token => %w", err)
		}*/

	return nil
}

func (s *Service) GenerateToken(length int) (string, error) {
	// Calculate the required byte size based on the length of the token
	byteSize := length / 2 // Each byte is represented by 2 characters in hexadecimal encoding

	// Create a byte slice to store the random bytes
	randomBytes := make([]byte, byteSize)

	// Read random bytes from the crypto/rand package
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	// Encode the random bytes into a hexadecimal string
	token := fmt.Sprintf("%x", randomBytes)

	return token, nil
}

func (s *Service) GetUnverifiedEmails() ([]string, error) {
	q := `SELECT email FROM subscribers WHERE is_verified = 0`
	rows, err := s.db.Query(q)
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()

	emails := []string{}

	for rows.Next() {
		var email string
		err = rows.Scan(&email)
		if err != nil {
			return []string{}, err
		}
		emails = append(emails, email)
	}

	return emails, nil

}

func (s *Service) AddVerificationToken(email, verificationToken string) error {
	q := `UPDATE subscribers SET verification_token = ?, is_verified = 0 WHERE email = ?`
	_, err := s.db.Exec(q, verificationToken, email)
	if err != nil {
		return fmt.Errorf("could not add verification token to user by email => %w", err)
	}
	return nil
}

func (s *Service) SendVerificationToken(email, token string) error {
	return fmt.Errorf("not yet implemented")
}

func (s *Service) VerifySubscription(verificationToken string) error {
	q := `UPDATE subscribers SET is_verified = 1 WHERE verification_token = ?`
	_, err := s.db.Exec(q, verificationToken)
	if err != nil {
		return fmt.Errorf("could not verify subscription via verification token => %w", err)
	}
	return nil
}
