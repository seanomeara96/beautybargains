package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

const adminEmailContextKey contextKey = "admin_email"
const (
	Dev  Mode = "dev"
	Prod Mode = "prod"
)

var db *sql.DB
var err error
var mode Mode
var port string
var productionDomain string
var store *sessions.CookieStore

/* models end */

func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	db, err = sql.Open("sqlite3", "main.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_skip := flag.Bool("skip", false, "skip bannner extraction and hashtag processing")
	_port := flag.String("port", "", "http port")
	_mode := flag.String("mode", "", "deployment mode")

	flag.Parse()

	port = *_port
	mode = Mode(*_mode)
	skip := *_skip

	if !skip {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			process()
			for {
				<-ticker.C
				process()
			}
		}()
	} else {
		log.Println("skipping banner extractions and hashtag process")
	}

	if err := server(); err != nil {
		log.Fatal(err)
	}

}

func server() error {

	if port == "" {
		return fmt.Errorf("port is required via -port flag")
	}

	if mode == "" || mode == "dev" {
		log.Println("Starting server in development mode.")
		mode = Dev
	}

	store = sessions.NewCookieStore([]byte(os.Getenv(`SESSION_KEY`)))

	productionDomain = os.Getenv("PROD_DOMAIN")
	if mode == Prod && productionDomain == "" {
		return errors.New("must supply production domain to run server in prod")
	}

	r := http.NewServeMux()

	assetsDir := http.Dir("assets/dist")
	assetsFileServer := http.FileServer(assetsDir)
	r.Handle("/assets/", http.StripPrefix("/assets/", assetsFileServer))

	imageDir := http.Dir("static/website_screenshots")
	imagesFileServer := http.FileServer(imageDir)
	r.Handle("/website_screenshots/", http.StripPrefix("/website_screenshots/", imagesFileServer))

	/*
		Serve robots.txt & sitemap
	*/
	r.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/robots.txt")
	})
	r.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/sitemap.xml")
	})

	globalMiddleware := []middleware{pathLogger}

	handle := newHandleFunc(r, globalMiddleware)

	handle("/", handleGetFeed)
	handle("GET /store/{websiteName}", handleGetFeed)
	handle("POST /subscribe", handlePostSubscribe)
	handle("GET /subscribe", handleGetSubscribePage)
	handle("GET /subscribe/verify", handleGetVerifySubscription)

	handle("GET /admin/signin", adminHandleGetSignIn)
	handle("POST /admin/signin", adminHandlePostSignIn)
	handle("GET /admin/signout", adminHandleGetSignOut)
	handle("GET /admin", mustBeAdmin(adminHandleGetDashboard))
	handle("GET /admin/manage/subscribers", mustBeAdmin(adminhandleGetSubscribers))
	handle("GET /admin/events/edit/{id}", mustBeAdmin(adminHandleEditPostPage))
	handle("POST /admin/events/edit/{id}", mustBeAdmin(adminHandlePostEditPost))

	handle("GET /admin/events/delete/{id}", mustBeAdmin(adminDeletePostPage))
	handle("GET /admin/events/delete/{id}/confirm", mustBeAdmin(adminConfirmDeletePost))

	log.Println("Server listening on http://localhost:" + port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		return fmt.Errorf("failure to launch. %w", err)
	}

	return nil
}

/* db funcs */

var errDBNil = errors.New("db is nil")
