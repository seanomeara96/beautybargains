package main

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/sessions"
)

type Handler struct {
	store   *sessions.CookieStore
	mode    Mode
	domain  string
	service *Service
	render  *Renderer
}

func (h *Handler) handleGetHomePage(w http.ResponseWriter, r *http.Request) error {

	rows, err := h.service.db.Query(`
	SELECT
		*
	FROM
		(
			SELECT
				id,
				website_id,
				src_url,
				author_id,
				score,
				description,
				timestamp
			FROM
				posts
			ORDER BY
				timestamp DESC
		)
	GROUP BY
		website_id
	LIMIT
		6`)
	if err != nil {
		return err
	}

	posts, err := scanPosts(rows, make([]Post, 0, 6))
	if err != nil {
		return err
	}

	events, err := h.service.ConvertPostsToEvents(posts)
	if err != nil {
		return err
	}

	trendingHashtags, err := h.service.GetTrendingHashtags()
	if err != nil {
		return err
	}

	c, err := r.Cookie("subscription_status")
	subscribed := err == nil && c.Value == "subscribed"
	if err != nil && err != http.ErrNoCookie {
		log.Printf("Warning: Error getting subscription_status cookie: %v", err)
	}

	data := map[string]any{
		"PageTitle":         "All of Ireland's Top Beauty Deals and Discount Codes in One Place!",
		"MetaDescription":   "We keep an eye on all your favourite beauty retailers' top offers and discount codes so you dont have to.",
		"Canonical":         r.URL.Path,
		"AlreadySubscribed": subscribed,
		"Events":            events,
		"Websites":          getWebsites(0, 0),
		"Trending":          trendingHashtags,
	}

	return h.render.Page(w, "feedpage", data)
}

func (h *Handler) handleGetFeed(w http.ResponseWriter, r *http.Request) error {

	hashtagQuery := r.URL.Query().Get("hashtag")

	websitePath := r.PathValue("websitePath")

	website, _ := getWebsiteByPath(websitePath)

	var postIDs []int
	if hashtagQuery != "" {
		if err := h.service.getPostIDsByHashtagQuery(hashtagQuery, postIDs); err != nil {
			return err
		}
	}

	posts, err := h.service.GetPreviewPosts(website, postIDs)
	if err != nil {
		return err
	}

	events, err := h.service.ConvertPostsToEvents(posts)
	if err != nil {
		return err
	}

	trendingHashtags, err := h.service.GetTrendingHashtags()
	if err != nil {
		return err
	}

	c, err := r.Cookie("subscription_status")
	subscribed := err == nil && c.Value == "subscribed"
	if err != nil && err != http.ErrNoCookie {
		log.Printf("Warning: Error getting subscription_status cookie: %v", err)
	}

	data := map[string]any{
		"PageTitle":         fmt.Sprintf("Latest offers and Discount Codes for %s", website.WebsiteName),
		"MetaDescription":   fmt.Sprintf("we track the offers and discounts on %s so you dont have to. Signup to the newsletter to have all the latest and greatest deals delivered staight to your inbox.", website.WebsiteName),
		"Canonical":         r.URL.Path,
		"AlreadySubscribed": subscribed,
		"Events":            events,
		"Websites":          getWebsites(0, 0),
		"Trending":          trendingHashtags,
		"SelectedStore":     website,
	}

	return h.render.Page(w, "feedpage", data)
}

func (h *Handler) handleStoreSubscription(w http.ResponseWriter, r *http.Request) error {

	// Set a reasonable maximum size for the form data to prevent memory exhaustion
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("could not parse form: %w", err)
	}

	// Validate and sanitize email input
	email := strings.TrimSpace(r.FormValue("email"))
	if !isValidEmail(email) {
		return h.render.Template(w, "subscriptionform", map[string]any{
			"EmailErr": "Please provide a valid email address",
		})
	}

	// Use constant-time comparison for consent check to prevent timing attacks
	consent := r.FormValue("consent")
	if !(subtle.ConstantTimeCompare([]byte(consent), []byte("on")) == 1) {
		return h.render.Template(w, "subscriptionform", map[string]any{
			"ConsentErr": "Please consent so we can add you to our mailing list. Thanks!",
		})
	}

	// Use a transaction to ensure data consistency
	tx, err := h.service.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Check for existing email first
	var existingEmail string
	err = tx.QueryRow("SELECT email FROM subscribers WHERE email = ?", email).Scan(&existingEmail)
	if err != sql.ErrNoRows {
		if err == nil {
			return h.render.Template(w, "subscriptionform", map[string]any{
				"EmailErr": "This email is already subscribed",
			})
		}
		return fmt.Errorf("error checking for existing email: %w", err)
	}

	// Insert new subscriber
	if _, err = tx.Exec(`INSERT INTO subscribers(email, consent) VALUES (?, 1)`, email); err != nil {
		if isSQLiteConstraintError(err) {
			return h.render.Template(w, "subscriptionform", map[string]any{
				"EmailErr": "This email is already subscribed",
			})
		}
		return fmt.Errorf("could not insert email into subscribers table: %w", err)
	}

	// Generate a cryptographically secure token with sufficient entropy
	token := make([]byte, 32)
	if _, err = rand.Read(token); err != nil {
		return fmt.Errorf("failed to generate secure token: %w", err)
	}
	tokenStr := hex.EncodeToString(token)

	// Update the verification token
	if _, err = tx.Exec(
		`UPDATE subscribers SET verification_token = ?, is_verified = 0 WHERE email = ?`,
		tokenStr, email,
	); err != nil {
		return fmt.Errorf("could not add verification token to user by email: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Set secure cookie with appropriate flags
	http.SetCookie(w, &http.Cookie{
		Name:     "subscription_status",
		Value:    "subscribed",
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HttpOnly: true,                    // Prevent XSS attacks
		Secure:   true,                    // Only send over HTTPS
		SameSite: http.SameSiteStrictMode, // Prevent CSRF attacks
		Path:     "/",                     // Restrict cookie scope
	})

	// Log verification URL (for development only)
	if h.mode == Dev {
		log.Printf("Verify subscription at %s/subscribe/verify?token=%s", h.domain, tokenStr)
	}

	return h.render.Template(w, "subscriptionsuccess", nil)
}

// Helper function to validate email format
func isValidEmail(email string) bool {
	if len(email) > 254 || len(email) < 3 {
		return false
	}
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// Helper function to check for SQLite constraint errors
func isSQLiteConstraintError(err error) bool {
	// SQLite error code 19 is SQLITE_CONSTRAINT
	return strings.Contains(err.Error(), "UNIQUE constraint failed") ||
		strings.Contains(err.Error(), "constraint failed")
}

func (h *Handler) handleSubscribe(w http.ResponseWriter, r *http.Request) error {
	return h.render.Page(w, "subscribepage", map[string]any{
		"PageTitle":       "Subscribe to the BeautyBargains Newsletter to never miss a Deal",
		"MetaDescription": "We drop all the latest offers from Top Beauty Retailers in Ireland into one email to save you from getting thousands of emails in your inbox every day.",
		"Canonical":       r.URL.Path,
	})
}

func (h *Handler) handleGetVerifySubscription(w http.ResponseWriter, r *http.Request) error {
	token := r.URL.Query().Get("token")

	if subtle.ConstantTimeCompare([]byte(token), []byte("")) == 1 {
		// ("Warning: subscription verification attempted with no token")
		return h.handleGetFeed(w, r)
	}

	// Validate token format (should be 64 characters hex string since we generate 32 bytes)
	if !isValidVerificationToken(token) {
		if h.mode == Dev {
			log.Printf("Warning: invalid verification token format attempted: %s", token)
		}
		return h.handleGetFeed(w, r)
	}

	result, err := h.service.db.Exec(`
	UPDATE
		subscribers
	SET
		is_verified = 1
	WHERE
		verification_token = ? 
	AND 
		is_verified = 0`, token)
	if err != nil {
		return fmt.Errorf(
			"could not verify subscription via verification token => %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected %w", err)
	}

	if rowsAffected == 0 {

		if h.mode == Dev {
			log.Printf("warning: verification token was valid but no rows were affected")
		}

		return h.handleGetFeed(w, r)
	}

	// subscription confirmed
	return h.render.Page(w, "subscriptionverification", map[string]any{
		"PageTitle":       "Thanks for Signing Up!",
		"MetaDescription": "Keep an eye out for our newsletter!",
		"Canonical":       r.URL.Path,
	})
}

// Helper function to validate token format
func isValidVerificationToken(token string) bool {
	// Token should be exactly 64 characters (32 bytes in hex)
	if len(token) != 64 {
		return false
	}
	// Check if it's a valid hex string
	_, err := hex.DecodeString(token)
	return err == nil
}

func (h *Handler) Unauthorized(w http.ResponseWriter, r *http.Request) error {

	w.WriteHeader(http.StatusUnauthorized)

	return h.render.Page(
		w, "unauthorizedpage",
		map[string]any{
			"PageTitle":       "BeautyBargains.ie | Unauthorized",
			"MetaDescription": "You are attemptig to access without authorization",
			"Canonical":       r.URL.Path,
		},
	)
}

func (h *Handler) handleListCoupons(w http.ResponseWriter, r *http.Request) error {

	coupons, err := h.service.GetCoupons(getCouponParams{Limit: 50, Offset: 0})
	if err != nil {
		return err
	}

	type WebsiteCoupon struct {
		Coupon  CouponCode
		Website Website
	}
	data := make([]WebsiteCoupon, len(coupons))
	for i, coupon := range coupons {
		data[i].Coupon = coupon
		site, err := getWebsiteByID(coupon.WebsiteID)
		if err != nil {
			return err
		}
		data[i].Website = site
	}

	return h.render.Page(w,
		"couponcodes",
		map[string]any{
			"PageTitle":       "Find Coupons/Discount Codes for top Beauty Retailers in Ireland!",
			"MetaDescription": "We monitor for new discount codes daily and put them all here in the one place to save you hours of searching",
			"WebsiteCoupons":  data,
			"Canonical":       r.URL.Path,
		},
	)
}
