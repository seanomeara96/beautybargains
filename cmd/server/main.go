package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/seanomeara96/tgram"
)

const (
	Dev  Mode = "dev"
	Prod Mode = "prod"
)

/* models end */

func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatal(fmt.Errorf("failed to load .env file: %w", err))
	}

	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		log.Fatal(fmt.Errorf("failed to open database: %w", err))
	}
	defer db.Close()

	reportErr, err := tgram.NewErrorReporter(
		"beautybargains.ie", os.Getenv("TGRAM_BOT_API_TOKEN"), os.Getenv("TGRAM_CHAT_ID"),
	)
	if err != nil {
		log.Fatalf("failed to connect the error reporter: %v", err)
	}

	service, err := NewService(db, reportErr)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create new service: %w", err))
	}
	defer service.Close()

	_skip := flag.Bool("skip", false, "skip bannner extraction and hashtag jobs")
	_port := flag.String("port", "", "http port")
	_mode := flag.String("mode", "", "deployment mode")

	flag.Parse()

	port := *_port
	mode := Mode(*_mode)
	skip := *_skip
	// comment

	if !skip {
		go func() {
			for {
				log.Println("start jobs")
				if err := extractOffersFromBanners(service); err != nil {
					reportErr(fmt.Errorf("failed to extract offers from banners: %w", err))
				}
				if err := processHashtags(service); err != nil {
					reportErr(fmt.Errorf("failed to process hashtags: %w", err))
				}
				if err := scorePosts(service); err != nil {
					reportErr(fmt.Errorf("failed to score posts: %w", err))
				}
				log.Println("finished jobs")
				time.Sleep(5 * time.Minute)
			}
		}()
	}

	if err := server(port, mode, service); err != nil {
		log.Fatal(fmt.Errorf("server error: %w", err))
	}

}
