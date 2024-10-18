package chat

import (
	"testing"

	"github.com/joho/godotenv"
)

func TestGetBannerDescriptionsV2(t *testing.T) {

	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatal(err)
	}

	rating, err := ChatRateBrand("Color Wow")
	if err != nil {
		t.Fatal(err)
	}

	if rating == 0 {
		t.Error("expected a rating of more than 0")
	}

}
