package main

import (
	"beautybargains/services"
	"fmt"
	"testing"

	"github.com/joho/godotenv"
)

func TestDescribeBanner(t *testing.T) {
	// t.Fatal("not implemented")
	err := godotenv.Load("../.env")
	if err != nil {
		t.Error("Error loading .env file")
		return
	}

	chat := services.InitChat()

	description, err := chat.GetOfferDescription("beautyfeatures.ie", "https://cdn11.bigcommerce.com/s-63354/images/stencil/1920w/carousel/363/winter_sale_banner__16363.jpg?c=2")
	if err != nil {
		t.Error(err)
		return
	}

	if description == "" {
		t.Error("no description")
	}

	fmt.Println(description)
	t.Error(0)

}
