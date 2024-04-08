package pricedatarepo

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestGetLatestPrice(t *testing.T) {
	db, err := sql.Open("sqlite3", "pricedata.db")
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()

	repo := New(db)

	prices, err := repo.GetByProductID(1)
	if err != nil {
		t.Error(err)
		return
	}

	latest, err := repo.GetLatestPrice(1)
	if err != nil {
		t.Error(err)
		return
	}

	found := false
	for _, price := range prices {
		if price.PriceID == latest.PriceID {
			found = true
		}
	}

	if !found {
		t.Error("lastest price not found")
	}

	maxPriceID := 0
	for _, price := range prices {
		if price.PriceID > maxPriceID {
			maxPriceID = price.PriceID
		}
	}

	if maxPriceID != latest.PriceID {
		t.Error("Expected these ids to be the same")
	}

}
