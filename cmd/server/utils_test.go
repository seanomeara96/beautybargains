package main

import "testing"

func TestExtractWebsiteBannerURLs(t *testing.T) {
	banners, err := extractWebsiteBannerURLs(websites[4])
	if err != nil {
		t.Fatal(err)
	}

	if len(banners) < 1 || banners[0].Src == "" {
		t.Error("Expected banners with src values")
	}

}
