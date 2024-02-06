package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestExtractBanners(t *testing.T) {
	res, err := http.Get("https://www.mccauley.ie/")
	if err != nil {
		t.Errorf("error sending get request to extract banner urls %v", err)
		return
	}

	defer res.Body.Close()

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		t.Errorf("error parsing document with go query %v", err)
		return
	}

	doc.Find("[data-content-type=slide] [data-background-images]").Each(func(i int, s *goquery.Selection) {
		value, found := s.Attr("data-background-images")
		if found {
			type MCBackgroundImage struct {
				MobileImage string `json:"mobile_image"`
			}

			var x MCBackgroundImage
			value = strings.ReplaceAll(value, "\\\"", "\"")
			err = json.Unmarshal([]byte(value), &x)
			if err != nil {
				t.Error(err)
			}

			t.Error(x.MobileImage)
		}
	})

}
