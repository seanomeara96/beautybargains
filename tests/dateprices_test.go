package main

import (
	"beautybargains/services"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestPriceByDate(t *testing.T) {
	// t.Fatal("not implemented")

	db, err := sql.Open("sqlite3", "../data")
	if err != nil {
		t.Error(err.Error())
		return
	}

	service := services.NewService(db)

	priceData, err := service.GetPriceDrops()
	if err != nil {
		t.Error(err.Error())
		return
	}

	fmt.Println(priceData[0])

	t.Error("")
}
