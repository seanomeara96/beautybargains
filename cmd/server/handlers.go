package main

import (
	"crypto/rand"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func handleGetFeed(w http.ResponseWriter, r *http.Request) error {
	websiteName := r.PathValue("websiteName")
	hashtagQuery := r.URL.Query().Get("hashtag")

	var website Website
	if websiteName != "" {
		w, err := getWebsiteByName(websiteName)
		if err == nil {
			website = w
		}
	}

	var postIDs []int
	if hashtagQuery != "" {
		hashtagID, err := getHashtagIDByPhrase(db, hashtagQuery)
		if err != nil {
			return fmt.Errorf("could not get hashtag id in get by phrase. %w", err)
		}

		postIdRows, err := db.Query("SELECT post_id FROM post_hashtags WHERE hashtag_id = ?", hashtagID)
		if err != nil {
			return fmt.Errorf("error getting post_ids from post_hashtags db. %w", err)
		}
		defer postIdRows.Close()

		for postIdRows.Next() {
			var id int
			err := postIdRows.Scan(&id)
			if err != nil {
				return fmt.Errorf("error scanning post_id in getposts. %w", err)
			}
			postIDs = append(postIDs, id)
		}
	}

	args := []any{}
	var q strings.Builder
	q.WriteString(`WITH orderedPosts AS (
	SELECT 
        p.id,
        p.description,
		p.author_id,
		p.Score,
        p.src_url,
        p.timestamp,
        p.website_id
    FROM 
        posts p `)

	if len(postIDs) > 0 {
		q.WriteString(`WHERE p.id IN (`)
		for i, id := range postIDs {
			if i > 0 {
				q.WriteString(", ")
			}
			q.WriteString("?")
			args = append(args, id)
		}
		q.WriteString(") ")
	} else if website.WebsiteID != 0 {
		q.WriteString(`WHERE website_id = ? `)
		args = append(args, website.WebsiteID)
	}

	q.WriteString(`ORDER BY 
        p.timestamp DESC
	LIMIT 6
	) SELECT 
        op.id,
        op.description,
		op.author_id,
		op.Score,
        op.src_url,
        op.timestamp,
        op.website_id
    FROM 
        orderedPosts op
	ORDER BY
		score DESC
	LIMIT 6
	`)

	rows, err := db.Query(q.String(), args...)
	if err != nil {
		return err
	}
	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(
			&post.ID,
			&post.Description,
			&post.AuthorID,
			&post.Score,
			&post.SrcURL,
			&post.Timestamp,
			&post.WebsiteID,
		); err != nil {
			return err
		}
		posts = append(posts, post)
	}

	events := make([]Event, 0, len(posts))
	for i := 0; i < len(posts); i++ {
		e := Event{}
		for _, persona := range getPersonas(0, 0) {
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

	rows, err = db.Query(`SELECT hashtag_id, count(post_id) FROM post_hashtags GROUP BY hashtag_id ORDER BY count(post_id) DESC LIMIT 5`)
	if err != nil {
		return fmt.Errorf("could not count hashtag mentions in db: %w", err)
	}
	defer rows.Close()

	top := make([]GetTopByPostCountResponse, 0, 5)
	for rows.Next() {
		var row GetTopByPostCountResponse
		if err := rows.Scan(&row.HashtagID, &row.PostCount); err != nil {
			return err
		}
		top = append(top, row)
	} // should expect an array like {hashtag, postcount}

	var trendingHashtags []*Trending
	for _, row := range top {
		hashtag, err := getHashtagByID(db, row.HashtagID)
		if err != nil {
			return fmt.Errorf("could not get hashtag by id at GetTrending in hashtagsvc. %w", err)
		}
		trendingHashtags = append(trendingHashtags, &Trending{Category: "Topic", Phrase: hashtag.Phrase, PostCount: row.PostCount})
	}

	subscribed := false
	c, err := r.Cookie("subscription_status")
	if err != nil && err != http.ErrNoCookie {
		return err
	}

	if c != nil {
		subscribed = c.Value == "subscribed"
	}

	data := map[string]any{
		"AlreadySubscribed": subscribed,
		"Events":            events,
		"Websites":          getWebsites(0, 0),
		"Trending":          trendingHashtags,
		"Categories":        getCategories(0, 0),
	}

	return renderPage(w, "feedpage", data)
}

func handlePostSubscribe(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("could not parse form: %w", err)
	}

	email := r.FormValue("email")
	consent := r.FormValue("consent")

	if consent != "on" {
		// TODO maybe create an error state
		return render(w, "subscriptionform", map[string]any{"ConsentErr": "Please consent so we can add you to our mailing list. Thanks!"})
	}

	q := `INSERT INTO subscribers(email, consent) VALUES (?, 1)`
	if _, err := db.Exec(q, email); err != nil {
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
	if _, err := db.Exec(setVerificationTokenQuery, token, email); err != nil {
		return fmt.Errorf("could not add verification token to user by email => %w", err)
	}

	domain := "http://localhost:" + port
	if mode == Prod {
		domain = productionDomain
	}

	log.Printf("Verify subscription at %s/subscribe/verify?token=%s", domain, token)
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

	return render(w, "subscriptionsuccess", nil)

}

func handleGetSubscribePage(w http.ResponseWriter, r *http.Request) error {
	return renderPage(w, "subscribepage", map[string]any{})
}

func handleGetVerifySubscription(w http.ResponseWriter, r *http.Request) error {
	vars := r.URL.Query()
	token := vars.Get("token")

	if token == "" {
		// ("Warning: subscription verification attempted with no token")
		return handleGetFeed(w, r)
	}

	q := `UPDATE subscribers SET is_verified = 1 WHERE verification_token = ?`
	_, err := db.Exec(q, token)
	if err != nil {
		return fmt.Errorf("could not verify subscription via verification token => %w", err)
	}

	// subscription confirmed

	return renderPage(w, "subscriptionverification", map[string]any{})
}
