package main

import (
	"context"
	"log"
	"net/http"
	"os"
)

func (h *Handler) mustBeAdmin(next handleFunc) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		// do something admin

		session, err := h.store.Get(r, "admin_session")
		if err != nil {
			return err
		}

		email, found := session.Values["admin_email"]
		if !found || email != os.Getenv("ADMIN_EMAIL") {
			return h.Unauthorized(w, r)
		}

		return next(w, r.WithContext(context.WithValue(r.Context(), adminEmailContextKey, email)))
	}
}

func (h *Handler) pathLogger(next handleFunc) handleFunc {
	return (func(w http.ResponseWriter, r *http.Request) error {
		log.Printf("%s => %s", r.Method, r.URL.Path)
		return next(w, r)
	})
}
