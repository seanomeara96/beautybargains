package main

import (
	"database/sql"
	"flag"
	"log"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

const adminEmailContextKey contextKey = "admin_email"
const (
	Dev  Mode = "dev"
	Prod Mode = "prod"
)

/* models end */

func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	service := &Service{db}

	_skip := flag.Bool("skip", false, "skip bannner extraction and hashtag processing")
	_port := flag.String("port", "", "http port")
	_mode := flag.String("mode", "", "deployment mode")

	flag.Parse()

	port := *_port
	mode := Mode(*_mode)
	skip := *_skip

	if !skip {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			process(service)
			for {
				<-ticker.C
				process(service)
			}
		}()
	} else {
		log.Println("skipping banner extractions and hashtag process")
	}

	if err := server(port, mode, service); err != nil {
		log.Fatal(err)
	}

}
