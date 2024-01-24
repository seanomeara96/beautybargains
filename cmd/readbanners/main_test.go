package main

import (
	"beautybargains/models"
	"strings"
	"testing"
)

func TestExtractBanners(t *testing.T) {
	banners, err := ExtractBannerURLs(models.Website{WebsiteID: 3, URL: "https://millies.ie"})
	if err != nil {
		t.Error(err)
		return
	}

	if len(banners) < 1 {
		t.Error("Expected banners")
		return
	}

	for _, banner := range banners {
		if strings.Contains(banner, "{width}") {
			t.Errorf("Find and replace failed for millies. Expected not to find {width} in %s", banner)
		}
	}
}
