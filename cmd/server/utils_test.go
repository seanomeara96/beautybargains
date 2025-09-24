package main

import "testing"

func TestExtractWebsiteBannerURLs(t *testing.T) {

	for _, website := range websites {

		banners, err := extractWebsiteBannerURLs(website)
		if err != nil {
			t.Fatal(err)
		}

		if len(banners) < 1 || banners[0].Src == "" {
			t.Errorf("Expected banners with src values for %s", website.WebsiteName)
		}
	}

}
