package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
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

func (h *Handler) handleGetFeed(w http.ResponseWriter, r *http.Request) error {

	hashtagQuery := r.URL.Query().Get("hashtag")

	website, _ := getWebsiteByName(r.PathValue("websiteName"))

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
		"AlreadySubscribed": subscribed,
		"Events":            events,
		"Websites":          getWebsites(0, 0),
		"Trending":          trendingHashtags,
	}

	return h.render.Page(w, "feedpage", data)
}

func (h *Handler) handleStoreSubscription(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("could not parse form: %w", err)
	}

	email := r.FormValue("email")
	consent := r.FormValue("consent")

	if consent != "on" {
		// TODO maybe create an error state
		return h.render.Template(w, "subscriptionform", map[string]any{"ConsentErr": "Please consent so we can add you to our mailing list. Thanks!"})
	}

	q := `INSERT INTO subscribers(email, consent) VALUES (?, 1)`
	if _, err := h.service.db.Exec(q, email); err != nil {
		return fmt.Errorf("could not insert email into subscibers table => %w", err)
	}

	// Calculate the required byte size based on the length of the token
	byteSize := 20 / 2 // Each byte is represented by 2 characters in hexadecimal encoding

	// Create a byte slice to store the random bytes
	randomBytes := make([]byte, byteSize)

	// Read random bytes from the crypto/rand package
	if _, err := rand.Read(randomBytes); err != nil {
		return err
	}

	// Encode the random bytes into a hexadecimal string
	token := fmt.Sprintf("%x", randomBytes)

	// TODO move gernate token to service level
	// TODO do one insert with both email and verification token

	setVerificationTokenQuery := `UPDATE subscribers SET verification_token = ?, is_verified = 0 WHERE email = ?`
	if _, err := h.service.db.Exec(setVerificationTokenQuery, token, email); err != nil {
		return fmt.Errorf("could not add verification token to user by email => %w", err)
	}

	log.Printf("Verify subscription at %s/subscribe/verify?token=%s", h.domain, token)
	/* TODO implement this in production. For now log to console
	err = s.SendVerificationToken(email, token)
		if err != nil {
			return fmt.Errorf("failed to send verification token => %w", err)
		}*/

	http.SetCookie(w, &http.Cookie{
		Name:    "subscription_status",
		Value:   "subscribed",
		Expires: time.Now().Add(30 * 24 * time.Hour),
	})

	return h.render.Template(w, "subscriptionsuccess", nil)

}

func (h *Handler) handleSubscribe(w http.ResponseWriter, r *http.Request) error {
	return h.render.Page(w, "subscribepage", map[string]any{})
}

func (h *Handler) handleGetVerifySubscription(w http.ResponseWriter, r *http.Request) error {
	vars := r.URL.Query()
	token := vars.Get("token")

	if token == "" {
		// ("Warning: subscription verification attempted with no token")
		return h.handleGetFeed(w, r)
	}

	q := `UPDATE subscribers SET is_verified = 1 WHERE verification_token = ?`
	_, err := h.service.db.Exec(q, token)
	if err != nil {
		return fmt.Errorf("could not verify subscription via verification token => %w", err)
	}

	// subscription confirmed

	return h.render.Page(w, "subscriptionverification", map[string]any{})
}
func (h *Handler) Unauthorized(w http.ResponseWriter, r *http.Request) error {

	w.WriteHeader(http.StatusUnauthorized)

	return h.render.Page(w, "unauthorizedpage", map[string]any{})
}
