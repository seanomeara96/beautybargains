package main

import (
	"database/sql"
	"html/template"
	"net/http"
	"time"
)

type Brand struct {
	ID    int
	Name  string
	Path  string
	Score float64
}

type Post struct {
	WebsiteID   int
	ID          int
	Description string
	SrcURL      string
	Link        sql.NullString
	Timestamp   time.Time
	AuthorID    int // supposed to correspond with a persona id
	Score       float64
}

type Hashtag struct {
	ID     int
	Phrase string
}

type PostHashtag struct {
	ID        int
	PostID    int
	HashtagID int
}

type Trending struct {
	Category  string
	Phrase    string
	PostCount int
}

type handleFunc func(w http.ResponseWriter, r *http.Request) error
type middleware func(next handleFunc) handleFunc

type MenuItem struct {
	Path string
	Name string
}

type BasePageData struct {
	Request   *http.Request
	MenuItems []MenuItem
}

type Pagination struct {
	PageNumber int
	MaxPages   int
}

type Persona struct {
	ID           int
	Name         string
	Description  string
	ProfilePhoto string
}

type ProfilePhoto struct {
	ID      int
	URL     string
	Name    string
	ModelID int
}

type Profile struct {
	Photo    string
	Username string
}

type ExtraImage struct {
	Src string
	Alt string
}

type Content struct {
	Summary     string
	TimeElapsed string
	ExtraImages *[]ExtraImage  // optional
	ExtraText   *template.HTML // optional
}

type EventMeta struct {
	CTALink *string
	Src     *string
	Likes   int
}

type Event struct {
	ID      int
	Profile Profile
	Content Content
	Meta    EventMeta
}

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

type BannerData struct {
	Src            string
	SupportingText string
	Href           string
}

// Website struct matching the Websites table
type Website struct {
	WebsiteID   int     `json:"website_id"`
	WebsiteName string  `json:"website_name"`
	URL         string  `json:"url"`
	Country     string  `json:"country"`
	Score       float64 `json:"score"`
	Screenshot  string  `json:"screenshot"`
}

/* models begin */
type Mode string
type contextKey string
