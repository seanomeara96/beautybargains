package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/seanomeara96/paginator"
)

func (h *Handler) adminHandleGetDashboard(w http.ResponseWriter, r *http.Request) error {

	limit, offset, _ := paginator.Paginate(r, 50)

	posts, err := h.service.getPosts(getPostParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return err
	}

	data := map[string]any{
		"Posts": posts,
		"Admin": true,
	}

	return h.render.Page(w, "admindashboard", data)
}

func (h *Handler) adminHandleListPosts(w http.ResponseWriter, r *http.Request) error {

	limit, offset, _ := paginator.Paginate(r, 50)

	posts, err := h.service.getPosts(getPostParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return err
	}

	data := map[string]any{
		"Posts": posts,
		"Admin": true,
	}

	return h.render.Page(w, "adminposts", data)
}

func (h *Handler) adminHandleGetSignIn(w http.ResponseWriter, r *http.Request) error {
	return h.render.Page(w, "adminsignin", nil)
}

func (h *Handler) adminhandleGetSubscribers(w http.ResponseWriter, r *http.Request) error {

	var subscribers []Subscriber

	rows, err := h.service.db.Query(`
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

	return h.render.Page(w, "adminsubscribers", map[string]any{
		"Subscribers": subscribers,
	})
}

// Handler to load the post for editing
func (h *Handler) adminHandleEditPost(w http.ResponseWriter, r *http.Request) error {
	// Parse post ID from URL
	postID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return err
	}

	// Fetch post from the database
	var post Post
	err = h.service.db.QueryRow(`
	SELECT
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

	if err = h.render.Page(w, "admineditpost", map[string]any{
		"Post": post,
	}); err != nil {
		return err
	}
	return nil
}

// Handler to update the post
func (h *Handler) adminHandleUpdatePost(w http.ResponseWriter, r *http.Request) error {
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
	_, err = h.service.db.Exec(`
	UPDATE
		posts
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

	refreshToken, err := h.authenticator.GetRefreshTokenFromRequest(r)
	if err != nil {
		return err
	}
	if err := h.authenticator.Logout(r.Context(), refreshToken); err != nil {
		return err
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil

}

func (h *Handler) adminHandlePostSignIn(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	email, password := r.Form.Get("email"), r.Form.Get("password")

	aToken, rToken, err := h.authenticator.Login(r.Context(), email, password)
	if err != nil {
		return err
	}

	h.authenticator.SetTokens(w, aToken, rToken)

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
	return nil
}

func (h *Handler) adminHandleDeletePostConfirmation(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return err
	}

	return h.render.Page(w, "adminconfirmdeletepost", map[string]any{
		"PostID": id,
	})

}

func (h *Handler) adminHandleDeletePost(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodDelete {
		return errors.New("delete method required")
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return err
	}

	if _, err := h.service.db.Exec(`DELETE FROM posts WHERE id = ?`, id); err != nil {
		return err
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
	return nil
}
