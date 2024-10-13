package main

import (
	"testing"

	"github.com/joho/godotenv"
)

func TestGetBannerDescriptionsV2(t *testing.T) {

	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatal(err)
	}

	websiteName := "millies.ie"
	bannerData := BannerData{
		Src: "https://millies.ie/cdn/shop/files/Happy_Birthday_to_Us.jpg",
	}
	offerDescription, err := analyzeOffer(websiteName, bannerData)
	if err != nil {
		t.Fatal(err)
	}

	t.Error(offerDescription)
}
