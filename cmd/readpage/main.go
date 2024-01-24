package main

import (
	"log"
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

func main() {

	res, err := http.Get("https://millies.ie/")
	if err != nil {
		log.Fatal(err)
		return
	}

	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return
	}

}
