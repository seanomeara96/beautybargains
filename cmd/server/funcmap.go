package main

import (
	"html/template"
	"net/http"
	"strings"
	"time"
)

var funcMap = template.FuncMap{
	"longDate":            formatLongDate,
	"placeholderImage":    placeholderImage,
	"truncateDescription": truncateDescription,
	"proper":              proper,
	"unescape":            unescape,
	"isCurrentPage":       isCurrentPage,
	"add":                 add,
	"subtract":            subtract,
	"lower":               lower,
}

func add(a, b int) int {
	return a + b
}

func subtract(a, b int) int {
	return a - b
}

func formatLongDate(t time.Time) string {
	return t.Format("January 2, 2006")
}

func placeholderImage(url string) string {
	if url == "" {
		return "https://semantic-ui.com/images/avatar/small/jenny.jpg"
	}
	return url
}

func truncateDescription(description string) string {
	if len(description) < 100 {
		return description
	}
	return string(description[0:100] + "...")
}

func unescape(s string) template.HTML {
	return template.HTML(s)
}

func proper(str string) string {
	words := strings.Fields(str)
	for i, word := range words {
		words[i] = strings.ToUpper(string(word[0])) + word[1:]
	}
	return strings.Join(words, " ")
}

func isCurrentPage(r *http.Request, path string) bool {
	return r.URL.Path == path
}

func lower(s string) string {
	return strings.ToLower(s)
}
