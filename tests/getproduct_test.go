package test

import (
	"beautybargains/services"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestGetProducts(t *testing.T) {
	db, err := sql.Open("sqlite3", "../data")
	if err != nil {
		t.Error(err)
		return
	}

	s := services.NewService(db)

	products, err := s.GetProductsByBrand(529, 100000, 0)
	if err != nil {
		t.Error(err)
		return
	}

	if len(products) > 10 {
		t.Errorf("product count %d", len(products))
	}

	if len(products) < 1 {
		t.Error("there should be products associated with this brand")
		return
	}
}
