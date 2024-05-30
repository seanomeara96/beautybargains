package main

import (
	"beautybargains/internal/scripts"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

/*
This script is supposed to loop through the posts, extract hashtags store hashtag data and relationships
*/
func main() {
	if err := scripts.ProcessHashtags(); err != nil {
		log.Fatal(err)
	}
}
