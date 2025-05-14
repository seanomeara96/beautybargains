package main

import (
	"net/http"
)

type Trending struct {
	Category  string
	Phrase    string
	PostCount int
}

type handleFunc func(w http.ResponseWriter, r *http.Request) error
type middleware func(next handleFunc) handleFunc

/* models begin */
type Mode string
