package main

import (
	"log"
	"net/http"
)

func (h *Handler) mustBeAdmin(next handleFunc) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		// do something admin

		aToken, rToken, err := h.authenticator.GetTokensFromRequest(r)
		if err != nil {
			return err
		}

		_, err = h.authenticator.ValidateToken(aToken)
		if err != nil {
			aToken, rToken, err = h.authenticator.Refresh(r.Context(), rToken)
			if err != nil {
				return err
			}
			h.authenticator.SetTokens(w, aToken, rToken)
		}

		return next(w, r)
	}
}

func (h *Handler) pathLogger(next handleFunc) handleFunc {
	return (func(w http.ResponseWriter, r *http.Request) error {
		log.Printf("%s => %s", r.Method, r.URL.Path)
		return next(w, r)
	})
}
