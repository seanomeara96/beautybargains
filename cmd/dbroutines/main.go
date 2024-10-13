package main

import (
	"database/sql"
	"log"

	"github.com/gosimple/slug"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	rows, err := tx.Query("SELECT id, name FROM brands WHERE path IS NULL")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	stmt, err := tx.Prepare("UPDATE brands SET path = ? WHERE id = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			log.Fatal(err)
		}
		_, err := stmt.Exec(slug.Make(name), id)
		if err != nil {
			log.Fatal(err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
}
