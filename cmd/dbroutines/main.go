package main

import (
	"beautybargains/internal/chat"
	"database/sql"
	"flag"
	"fmt"
	"log"

	"github.com/gosimple/slug"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

func main() {

	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := godotenv.Load(".env"); err != nil {
		log.Fatal(err)
	}

	funcName := flag.String("func", "list", "specify the function wou want to run")
	flag.Parse()

	funcs := map[string]func(db *sql.DB){
		"update_brand_paths": updateBrandPaths,
		"rate_brands":        rateBrands,
	}

	if *funcName == "list" {
		for k := range funcs {
			fmt.Println(k)
		}
		return
	}

	fn, found := funcs[*funcName]
	if !found {
		log.Printf("no such function exists '%s'", *funcName)
		return
	}

	fn(db)

}

func rateBrands(db *sql.DB) {

	rows, err := db.Query(`SELECT id, name FROM brands`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := tx.Prepare("UPDATE brands SET score = ? WHERE id = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for rows.Next() {
		var id int
		var brandName string
		if err := rows.Scan(&id, &brandName); err != nil {
			tx.Rollback()
			log.Fatal(err)
		}

		score, err := chat.ChatRateBrand(brandName)
		if err != nil {
			tx.Rollback()
			log.Println(err)
		}

		log.Printf("Brand: %s	Score: %d", brandName, score)

		_, err = stmt.Exec(score, id)
		if err != nil {
			tx.Rollback()
			log.Fatal(err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
}

func updateBrandPaths(db *sql.DB) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	rows, err := tx.Query(
		"SELECT id, name FROM brands WHERE path IS NULL",
	)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	stmt, err := tx.Prepare(
		"UPDATE brands SET path = ? WHERE id = ?",
	)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(
			&id, &name,
		); err != nil {
			log.Fatal(err)
		}
		if _, err := stmt.Exec(
			slug.Make(name), id,
		); err != nil {
			log.Fatal(err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
}
