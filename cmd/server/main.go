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

	if err := server(port, mode, db, service); err != nil {
		log.Fatal(err)
	}

}

func server(port string, mode Mode, db *sql.DB, service *Service) error {

	if port == "" {
		return fmt.Errorf("port is required via -port flag")
	}

	if mode == "" || mode == "dev" {
		log.Println("Starting server in development mode.")
		mode = Dev
	}

	store := sessions.NewCookieStore([]byte(os.Getenv(`SESSION_KEY`)))

	productionDomain := os.Getenv("PROD_DOMAIN")
	if mode == Prod && productionDomain == "" {
		return errors.New("must supply production domain to run server in prod")
	}

	currentDomain := "http://localhost:" + port
	if mode == Prod {
		currentDomain = productionDomain
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

	handler := Handler{
		db:      db,
		store:   store,
		mode:    mode,
		domain:  currentDomain,
		service: service,
	}

	globalMiddleware := []middleware{handler.pathLogger}

	handle := newHandleFunc(r, globalMiddleware)

	handle("/", handler.handleGetFeed)
	handle("GET /store/{websiteName}", handler.handleGetFeed)
	handle("GET /categories", handler.handleGetCategories)
	handle("POST /subscribe", handler.handlePostSubscribe)
	handle("GET /subscribe", handler.handleGetSubscribePage)
	handle("GET /subscribe/verify", handler.handleGetVerifySubscription)

	handle("GET /admin/signin", handler.adminHandleGetSignIn)
	handle("POST /admin/signin", handler.adminHandlePostSignIn)
	handle("GET /admin/signout", handler.adminHandleGetSignOut)
	handle("GET /admin", handler.mustBeAdmin(handler.adminHandleGetDashboard))
	handle("GET /admin/manage/subscribers", handler.mustBeAdmin(handler.adminhandleGetSubscribers))
	handle("GET /admin/events/edit/{id}", handler.mustBeAdmin(handler.adminHandleEditPostPage))
	handle("POST /admin/events/edit/{id}", handler.mustBeAdmin(handler.adminHandlePostEditPost))

	handle("GET /admin/events/delete/{id}", handler.mustBeAdmin(handler.adminDeletePostPage))
	handle("GET /admin/events/delete/{id}/confirm", handler.mustBeAdmin(handler.adminConfirmDeletePost))

	log.Println("Server listening on http://localhost:" + port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		return fmt.Errorf("failure to launch. %w", err)
	}

	return nil
}

/* db funcs */

var errDBNil = errors.New("db is nil")
