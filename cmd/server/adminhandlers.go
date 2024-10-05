package main

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) adminHandleGetDashboard(w http.ResponseWriter, r *http.Request) error {

	rows, err := h.db.Query("SELECT id, description, author_id, score, src_url, timestamp, website_id FROM posts ORDER BY timestamp DESC")
	if err != nil {
		return err
	}

	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.Description, &post.AuthorID, &post.Score, &post.SrcURL, &post.Timestamp, &post.WebsiteID); err != nil {
			return err
		}
		posts = append(posts, post)
	}

	personas := getPersonas(0, 0)

	events := make([]Event, 0, len(posts))
	for i := 0; i < len(posts); i++ {
		e := Event{}

		e.ID = posts[i].ID

		for _, persona := range personas {
			if persona.ID == posts[i].AuthorID {
				e.Profile.Username = persona.Name
				e.Profile.Photo = persona.ProfilePhoto
			}
		}
		//	e.Profile.Username

		// Step 1: Calculate Time Difference
		timeDiff := time.Since(posts[i].Timestamp)

		// Step 2: Determine Unit (Days or Hours)
		var unit string
		var magnitude int

		hours := int(timeDiff.Hours())
		days := hours / 24

		if days > 0 {
			unit = "Days"
			magnitude = days
		} else {
			if hours == 1 {
				unit = "Hour"
			} else {
				unit = "Hours"
			}
			magnitude = hours
		}

		// Step 3: Format String
		e.Content.TimeElapsed = fmt.Sprintf("%d %s ago", magnitude, unit)
		e.Meta.Src = &posts[i].SrcURL

		if posts[i].Link.Valid {
			e.Meta.CTALink = &posts[i].Link.String
		}

		pattern := regexp.MustCompile(`#(\w+)`)

		extraText := posts[i].Description

		matches := pattern.FindAllStringSubmatch(extraText, -1)

		for _, match := range matches {
			phrase := strings.ToLower(match[1])
			extraText = strings.Replace(extraText, match[0], fmt.Sprintf("<a class='text-blue-500' href='?hashtag=%s'>%s</a>", phrase, match[0]), 1)
		}

		extraTextHTML := template.HTML(extraText)

		e.Content.ExtraText = &extraTextHTML
		website, err := getWebsiteByID(posts[i].WebsiteID)
		if err != nil {
			return fmt.Errorf("could not get website by id %d. %v", posts[i].WebsiteID, err)
		}
		e.Content.Summary = fmt.Sprintf("posted an update about %s", website.WebsiteName)
		// e.Content.ExtraImages = nil
		e.Content.ExtraImages = &[]ExtraImage{{posts[i].SrcURL, ""}}
		events = append(events, e)
	}

	limit := 5
	q := `SELECT hashtag_id, count(post_id) FROM post_hashtags GROUP BY hashtag_id ORDER BY count(post_id) DESC LIMIT ?`
	rows, err = h.db.Query(q, limit)
	if err != nil {
		return fmt.Errorf("could not count hashtag mentions in db: %w", err)
	}
	defer rows.Close()

	top := make([]GetTopByPostCountResponse, 0, limit)
	for rows.Next() {
		var row GetTopByPostCountResponse
		if err := rows.Scan(&row.HashtagID, &row.PostCount); err != nil {
			return err
		}
		top = append(top, row)
	} // should expect an array like {hashtag, postcount}

	var trendingHashtags []Trending
	for _, row := range top {
		hashtag, err := h.service.getHashtagByID(row.HashtagID)
		if err != nil {
			return fmt.Errorf("could not get hashtag by id at GetTrending in hashtagsvc. %w", err)
		}
		trendingHashtags = append(trendingHashtags, Trending{Category: "Topic", Phrase: hashtag.Phrase, PostCount: row.PostCount})
	}

	data := map[string]any{
		"Events":     events,
		"Websites":   getWebsites(0, 0),
		"Trending":   trendingHashtags,
		"Categories": getCategories(0, 0),
		"Admin":      true,
	}

	return renderPage(w, "admindashboard", data)
}

func (h *Handler) adminHandleGetSignIn(w http.ResponseWriter, r *http.Request) error {
	return renderPage(w, "adminsignin", nil)
}

func (h *Handler) adminhandleGetSubscribers(w http.ResponseWriter, r *http.Request) error {

	var subscribers []Subscriber

	rows, err := h.db.Query(`SELECT
		id,
		email,
		full_name,
		consent,
		signup_date,
		verification_token,
		is_verified,
		preferences
	FROM
		subscribers`)
	if err != nil {
		return err
	}
	defer rows.Close()

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
			return err
		}
		subscribers = append(subscribers, s)
	}

	return renderPage(w, "adminsubscribers", map[string]any{
		"Subscribers": subscribers,
	})
}

// Handler to load the post for editing
func (h *Handler) adminHandleEditPostPage(w http.ResponseWriter, r *http.Request) error {
	// Parse post ID from URL
	postID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return err
	}

	// Fetch post from the database
	var post Post
	err = h.db.QueryRow(`SELECT
		website_id,
		id,
		description,
		src_url,
		link,
		timestamp,
		author_id,
		score
	FROM
		posts
	WHERE
		id = ?`, postID).Scan(
		&post.WebsiteID,
		&post.ID,
		&post.Description,
		&post.SrcURL,
		&post.Link,
		&post.Timestamp,
		&post.AuthorID,
		&post.Score,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("Post not found %d", http.StatusNotFound)
		}
		return err
	}

	// Render the edit post page

	if err = renderPage(w, "admineditpost", map[string]any{
		"Post": post,
	}); err != nil {
		return err
	}
	return nil
}

// Handler to update the post
func (h *Handler) adminHandlePostEditPost(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return fmt.Errorf("invalid request method %d", http.StatusMethodNotAllowed)
	}

	// Parse form values
	postID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return fmt.Errorf("invalid Post ID %d", http.StatusBadRequest)

	}

	websiteID, err := strconv.Atoi(r.FormValue("websiteID"))
	if err != nil {
		return fmt.Errorf("invalid Website ID %d", http.StatusBadRequest)
	}

	description := r.FormValue("description")
	srcURL := r.FormValue("srcURL")
	link := r.FormValue("link")
	authorID, err := strconv.Atoi(r.FormValue("authorID"))
	if err != nil {
		return fmt.Errorf("invalid Author ID %d", http.StatusBadRequest)
	}
	score, err := strconv.ParseFloat(r.FormValue("score"), 64)
	if err != nil {
		return fmt.Errorf("invalid Score %d", http.StatusBadRequest)
	}

	// Update post in the database
	_, err = h.db.Exec(`UPDATE posts
	SET
		website_id = ?,
		description = ?,
		src_url = ?,
		link = ?,
		author_id = ?,
		score = ?
	WHERE
		id = ?`,
		websiteID,
		description,
		srcURL,
		link,
		authorID,
		score,
		postID,
	)
	if err != nil {
		return fmt.Errorf("unable to update post %d", http.StatusInternalServerError)
	}

	// Redirect to the posts list or success page
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
	return nil
}

func (h *Handler) adminHandleGetSignOut(w http.ResponseWriter, r *http.Request) error {
	session, err := h.store.Get(r, "admin_session")
	if err != nil {
		return err
	}

	if session.Values["admin_email"] == nil || session.Values["admin_email"] != os.Getenv("ADMIN_EMAIL") {
		return errors.New("signout called but not logged in")
	}

	session.Values["admin_email"] = ""
	if err := session.Save(r, w); err != nil {
		return err
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil

}

func (h *Handler) adminHandlePostSignIn(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	if os.Getenv("ADMIN_EMAIL") == "" || os.Getenv("HASHED_PASSWORD") == "" {
		return fmt.Errorf("either admin_email or hashed_password is not set in env")
	}

	email, password := r.Form.Get("email"), r.Form.Get("password")

	if email != os.Getenv("ADMIN_EMAIL") {
		log.Println("incorrect admin email supplied")
		return renderPage(w, "adminsignin", nil)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(os.Getenv("HASHED_PASSWORD")), []byte(password)); err != nil {
		log.Printf("incorrect password supplied %v", err)
		return renderPage(w, "adminsignin", nil)
	}

	session, err := h.store.Get(r, "admin_session")
	if err != nil {
		return err
	}

	session.Values["admin_email"] = email

	if err := h.store.Save(r, w, session); err != nil {
		return err
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
	return nil
}

func (h *Handler) adminDeletePostPage(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return err
	}

	return renderPage(w, "adminconfirmdeletepost", map[string]any{
		"PostID": id,
	})

}

func (h *Handler) adminConfirmDeletePost(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return err
	}

	if _, err := h.db.Exec(`DELETE FROM posts WHERE id = ?`, id); err != nil {
		return err
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
	return nil
}
