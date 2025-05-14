package main

import (
	"net/http"

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

func (h *Handler) adminHandleGetSignIn(w http.ResponseWriter, r *http.Request) error {
	return h.render.Page(w, "adminsignin", nil)
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
