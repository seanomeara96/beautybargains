package main

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

func TestProcessHashtags(t *testing.T) {
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatal(fmt.Errorf("failed to load .env file: %w", err))
	}

	db, err := sql.Open("sqlite3", "../../main.db")
	if err != nil {
		t.Fatal(fmt.Errorf("failed to open database: %w", err))
	}
	defer db.Close()

	service, err := NewService(db)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to create new service: %w", err))
	}
	defer service.Close()

	if err := processHashtags(service); err != nil {
		t.Errorf("failed to process hashtahgs %v", err)
	}

}

func TestExtractBannerURLs(t *testing.T) {
	for i := range websites {
		bannerData, err := extractWebsiteBannerURLs(websites[i])
		if err != nil {
			t.Error(err)
		}

		if len(bannerData) < 1 {
			t.Errorf("extract banner urls should return at least one banner")
		}

	}
}
