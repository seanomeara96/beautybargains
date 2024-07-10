package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sashabaranov/go-openai"
)

/* models begin */
type Mode string

const (
	Dev  Mode = "dev"
	Prod Mode = "prod"
)

// Product struct matching the Products table
type Product struct {
	ProductID   int       `json:"product_id"`
	WebsiteID   int       `json:"website_id"`
	ProductName string    `json:"product_name"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	BrandID     int       `json:"brand_id"`
	Image       string    `json:"image"`
	LastCrawled time.Time `json:"last_crawled"`
	Brand       Brand
}

type ProductError struct {
	Product
	ErrorReason string
}

func NewProduct(websiteID int, url string) Product {
	return Product{
		WebsiteID:   websiteID,
		ProductName: url,
		URL:         url,
		Description: url,
	}
}

// Website struct matching the Websites table
type Website struct {
	WebsiteID   int    `json:"website_id"`
	WebsiteName string `json:"website_name"`
	URL         string `json:"url"`
	Country     string `json:"country"`
}

// PriceData struct matching the PriceData table
type PriceData struct {
	PriceID   int       `json:"price_id"`
	Name      string    `json:"name"`
	ProductID int       `json:"product_id"`
	SKU       string    `json:"sku"`
	Gtin12    string    `json:"gtin12"`
	Gtin13    string    `json:"gtin13"`
	Gtin14    string    `json:"gtin14"`
	Price     float64   `json:"price"`
	Currency  string    `json:"currency"`
	InStock   bool      `json:"in_stock"`
	Timestamp time.Time `json:"timestamp"`
	Image     string    `json:"image"`
}

type PriceAtTime struct {
	Price float64
	Date  time.Time
}

type PriceChange struct {
	ProductID         int
	CurrentPrice      float64
	CurrentTimeStamp  time.Time
	PreviousPrice     float64
	PreviousTimestamp time.Time
}

type ProductWithPrice struct {
	Product
	PriceData PriceData
}

type Brand struct {
	ID   int
	Name string
	Path string
}

type Post struct {
	WebsiteID   int
	ID          int
	Description string
	SrcURL      string
	Link        string
	Timestamp   time.Time
	AuthorID    int // supposed to correspond with a persona id
}

type Hashtag struct {
	ID     int
	Phrase string
}

type PostHashtag struct {
	ID        int
	PostID    int
	HashtagID int
}

type Trending struct {
	Category  string
	Phrase    string
	PostCount int
}

var DummyTrending = []Trending{
	{"Topic", "#this", 2},
	{"Topic", "#that", 2},
	{"Topic", "#the", 2},
	{"Topic", "#other", 2},
	{"Topic", "#thing", 2},
}

type handleFunc func(w http.ResponseWriter, r *http.Request) error

type MenuItem struct {
	Path string
	Name string
}

type BasePageData struct {
	Request   *http.Request
	MenuItems []MenuItem
}

type Pagination struct {
	PageNumber int
	MaxPages   int
}

type Persona struct {
	ID           int
	Name         string
	Description  string
	ProfilePhoto string
}

type ProfilePhoto struct {
	ID      int
	URL     string
	Name    string
	ModelID int
}

type Profile struct {
	Photo    string
	Username string
}

var DummyProfile = Profile{"https://vogue.implicitdev.com/images/liana-agron69e2c.png", "Leanne"}

type ExtraImage struct {
	Src string
	Alt string
}

var DummyExtraImages = []ExtraImage{
	{"https://vogue.implicitdev.com/images/liana-agron69e2c.png", "dummy image"},
	{"https://vogue.implicitdev.com/images/liana-agron69e2c.png", "dummy image"},
}

type Content struct {
	Summary     string
	TimeElapsed string
	ExtraImages *[]ExtraImage  // optional
	ExtraText   *template.HTML // optional
}

var DummyExtraText = template.HTML("Ours is a life of constant reruns. We're always circling back to where we'd we started, then starting all over again. Even if we don't run extra laps that day, we surely will come back for more of the same another day soon.")

var DummyContent = Content{
	"added 2 new photos",
	"4 Days Ago",
	&DummyExtraImages,
	&DummyExtraText,
}

type EventMeta struct {
	CTALink *string
	Src     *string
	Likes   int
}

var DummyEventMetaSrc = "/"

var DummyEventMeta = EventMeta{&DummyEventMetaSrc, &DummyEventMetaSrc, 0}

type Event struct {
	Profile Profile
	Content Content
	Meta    EventMeta
}

var DummyEvent = Event{DummyProfile, DummyContent, DummyEventMeta}

/* models end */

var menuItems = []MenuItem{
	{"/", "Home"},
	{"/promotions/", "Promotions"},
	{"/websites/", "Websites"},
	{"/products/", "Products"},
	{"/brands/", "Brands"},
}

func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := extractOffersFromBanners(db); err != nil {
		log.Fatal(err)
	}

	if err := processHashtags(db); err != nil {
		log.Fatal(err)
	}

	if err := server(db); err != nil {
		log.Fatal(err)
	}
}

func server(db *sql.DB) error {
	_port := flag.String("port", "", "http port")
	_mode := flag.String("mode", "", "deployment mode")

	flag.Parse()

	port := *_port
	mode := Mode(*_mode)

	if port == "" {
		return fmt.Errorf("port is required via -port flag")
	}

	if mode == "" {
		log.Println("No mode was supplied, starting server in development mode.")
		mode = Dev
	}

	productionDomain := os.Getenv("PROD_DOMAIN")
	if mode == Prod && productionDomain == "" {
		return errors.New("must supply production domain to run server in prod")
	}

	funcMap := template.FuncMap{
		"longDate":            formatLongDate,
		"placeholderImage":    placeholderImage,
		"truncateDescription": truncateDescription,
		"proper":              proper,
		"unescape":            unescape,
		"isCurrentPage":       isCurrentPage,
		"add":                 add,
		"subtract":            subtract,
		"lower":               lower,
	}

	tmpl, err := template.New("web").Funcs(funcMap).ParseGlob("web/templates/**/*.tmpl")
	if err != nil {
		return fmt.Errorf("error parsing templates. %v", err)
	}

	r := mux.NewRouter()
	r.StrictSlash(true)

	assetsDir := http.Dir("assets/dist")
	assetsFileServer := http.FileServer(assetsDir)
	r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", assetsFileServer))

	imageDir := http.Dir("images")
	imagesFileServer := http.FileServer(imageDir)
	r.PathPrefix("/images/").Handler(http.StripPrefix("/images/", imagesFileServer))

	/*
		Serve robots.txt & sitemap
	*/
	r.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/robots.txt")
	})
	r.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/sitemap.xml")
	})

	handle := func(path string, fn handleFunc) *mux.Route {
		return r.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if err := fn(w, r); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
	}

	/*
		Home / Index Handler
	*/
	handle("/", handleGetHomePage(db, tmpl))

	/*
		Promotions Handlers
	*/
	handle("/promotions/", handleGetPromotionsPage(db, tmpl)).Methods(http.MethodGet)
	handle("/promotions/{websiteName}", handleGetPromotionsPage(db, tmpl)).Methods(http.MethodGet)
	handle("/{websiteName}/promotions/", handleGetPromotionsPage(db, tmpl)).Methods(http.MethodGet)

	handle("/feed/", handleGetFeed(db, tmpl)).Methods(http.MethodGet)
	handle("/feed/{websiteName}/", handleGetFeed(db, tmpl)).Methods(http.MethodGet)

	handle("/websites/", handleGetWebsites(db, tmpl)).Methods(http.MethodGet)
	handle("/websites/{website_id}/", handleGetWebsiteByID(db, tmpl)).Methods(http.MethodGet)
	handle("/brands/", handleGetBrands(db, tmpl)).Methods(http.MethodGet)
	// handle("/brands/{brand_path}/", handleGetBrandByPath(db, tmpl)).Methods(http.MethodGet)
	handle("/subscribe/", handlePostSubscribe(mode, port, productionDomain, db, tmpl)).Methods(http.MethodPost)
	// handle("/subscribe/", handleGetSubscribePage(db, tmpl)).Methods(http.MethodGet)
	handle("/subscribe/verify", handleGetVerifySubscription(db, tmpl)).Methods(http.MethodGet)

	log.Println("Server listening on http://localhost:" + port)
	if err = http.ListenAndServe(":"+port, r); err != nil {
		return fmt.Errorf("failure to launch. %w", err)
	}
	return nil
}

func processHashtags(db *sql.DB) error {

	/*
		Get all posts. At some point I will have to implement a way to filter for posts
		that have not already been processed
	*/
	posts, err := getPosts(db, getPostParams{})
	if err != nil {
		return err
	}

	// Define a regular expression pattern for hashtags
	pattern := regexp.MustCompile(`#(\w+)`)

	for _, p := range posts {
		// Find all matches in the post
		matches := pattern.FindAllStringSubmatch(p.Description, -1)

		// Extract hashtags from the matches
		for _, match := range matches {
			if len(match) < 2 {
				return errors.New("match was less than 2")
			}
			phrase := strings.ToLower(match[1])
			count, err := countHashtagsByPhrase(db, phrase)
			if err != nil {
				return err
			}
			exists := count > 0

			/*
				If the phrase exists we want to check if it has a relationship to this post
				If it does not have a relationship we need to save the relationship
				If the phrase does not exist we need to save the phrase and the relationship.
			*/
			if exists {
				hashtagID, err := getHashtagIDByPhrase(db, phrase)
				if err != nil {
					return err
				}

				q := `SELECT count(*) FROM 	post_hashtags WHERE post_id = ? AND hashtag_id = ?`

				var count int
				if err := db.QueryRow(q, p.ID, hashtagID).Scan(&count); err != nil {
					return fmt.Errorf("could not count relationships between %d & %d. %v", p.ID, hashtagID, err)
				}
				if err != nil {
					return err
				}

				noRelationShip := count < 1
				if noRelationShip {
					err = insertPostHashtagRelationship(db, p.ID, hashtagID)
					if err != nil {
						return err
					}
				}
			} else {
				newTagID, err := insertHashtag(db, &Hashtag{Phrase: phrase})
				if err != nil {
					return err
				}
				err = insertPostHashtagRelationship(db, p.ID, newTagID)
				if err != nil {
					return err
				}
			}
		}

	}
	return nil
}

func extractOffersFromBanners(db *sql.DB) error {

	websites := getWebsites(0, 0)

	for _, website := range websites {
		bannerURLs, err := extractWebsiteBannerURLs(website)
		if err != nil {
			continue
		}

		uniqueBanners := []string{}
		for _, u := range bannerURLs {

			var bannerCount int
			err := db.QueryRow(`SELECT count(id) FROM posts WHERE srcURL = ?`, u).Scan(&bannerCount)
			if err != nil {
				return fmt.Errorf("error checking existance of banner %v", err)
			}

			bannerExists := bannerCount > 0

			if bannerExists {
				continue
			}

			uniqueBanners = append(uniqueBanners, u)

		}

		for _, url := range uniqueBanners {

			if url == "" {
				continue
			}

			description, err := generateOfferDescription(website.WebsiteName, url)
			if err != nil {
				return fmt.Errorf(`error getting offer description from chatgpt. 
				WebsiteName: %s,
				URL: %s,
				%v`, website.WebsiteName, url, err)
			}

			author, err := getRandomPersona(db)
			if err != nil {
				return fmt.Errorf("warning: could not get author from repo. %v", err)
			}

			// I picked 8 randomly for author id
			authorID := 8
			if author != nil {
				authorID = author.ID
			}

			_, err = db.Exec(
				"INSERT INTO posts(websiteID, srcURL, author_id, description, timestamp) VALUES (? , ? , ?, ?, ?)",
				website.WebsiteID, url, authorID, description, time.Now())
			if err != nil {
				return fmt.Errorf(`error saving banner promotion. 
				Website id: %d,
				URL: %s,
				AuthorID: %d,
				Description: %s,
				%v`, website.WebsiteID, url, authorID, description, err)
			}
		}

	}

	if err := processHashtags(db); err != nil {
		return err
	}
	return nil
}

/* Handlers */
func handleGetHomePage(db *sql.DB, tmpl *template.Template) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		err := tmpl.ExecuteTemplate(w, "home", map[string]any{"MenuItems": menuItems, "Request": r})
		if err != nil {
			return (err)
		}
		return nil
	}
}

func handleGetPromotionsPage(db *sql.DB, tmpl *template.Template) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		websiteName := vars["websiteName"]

		params := getPostParams{
			SortByTimestampDesc: true,
		}

		if websiteName != "" {
			website, err := getWebsiteByName(websiteName)
			if err == nil {
				params.WebsiteID = website.WebsiteID
			}
		}

		promos, err := getPosts(db, params)
		if err != nil {
			return fmt.Errorf("failed to get posts %w", err)
		}

		var buf bytes.Buffer
		err = tmpl.ExecuteTemplate(w, "promotionspage", map[string]any{"Promotions": promos, "MenuItems": menuItems, "Request": r})
		if err != nil {
			return fmt.Errorf("Error: %w", err)
		}

		return nil
	}
}

func handleGetFeed(db *sql.DB, tmpl *template.Template) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) error {

		vars := mux.Vars(r)
		websiteName := vars["websiteName"]

		hashtagQuery := r.URL.Query().Get("hashtag")

		type FeedParams struct {
			WebsiteID           int
			SortByTimestampDesc bool
			Hashtag             string
		}

		params := FeedParams{
			SortByTimestampDesc: true,
		}

		if websiteName != "" {
			website, err := getWebsiteByName(websiteName)
			if err == nil {
				params.WebsiteID = website.WebsiteID
			} else {
				log.Printf("Warning: User tried to get posts for %s. %v", websiteName, err)
			}
		}

		if hashtagQuery != "" {
			params.Hashtag = hashtagQuery
		}

		var postIDs []int
		if params.Hashtag != "" {
			hashtagID, err := getHashtagIDByPhrase(db, params.Hashtag)
			if err != nil {
				return fmt.Errorf("Could not get hashtag id in get by phrase. %w", err)
			}

			postIdRows, err := db.Query("SELECT post_id FROM post_hashtags WHERE hashtag_id = ?", hashtagID)
			if err != nil {
				return fmt.Errorf("Error getting post_ids from post_hashtags db. %w", err)
			}
			defer postIdRows.Close()

			ids := []int{}
			for postIdRows.Next() {
				var id int
				err := postIdRows.Scan(&id)
				if err != nil {
					return fmt.Errorf("Error scanning post_id in getposts. %w", err)
				}
				ids = append(ids, id)
			}
			postIDs = ids
		}

		getPostParams := getPostParams{}
		getPostParams.IDs = postIDs
		getPostParams.SortByTimestampDesc = params.SortByTimestampDesc
		getPostParams.WebsiteID = params.WebsiteID
		promos, err := getPosts(db, getPostParams)
		if err != nil {
			return fmt.Errorf("Error with postrepo GetAll func at postsvc.GetAll. %w", err)
		}

		personas, err := getAllPersonas(db)
		if err != nil {
			return fmt.Errorf("Could not get all personas feed page. %w", err)
		}

		events := []Event{}
		for i := 0; i < len(promos); i++ {
			e := Event{}

			for _, persona := range personas {
				if persona.ID == promos[i].AuthorID {
					e.Profile.Username = persona.Name
					e.Profile.Photo = persona.ProfilePhoto
				}
			}
			//	e.Profile.Username

			// Step 1: Calculate Time Difference
			timeDiff := time.Since(promos[i].Timestamp)

			// Step 2: Determine Unit (Days or Hours)
			var unit string
			var magnitude int

			hours := int(timeDiff.Hours())
			days := hours / 24

			if days > 0 {
				unit = "Days"
				magnitude = days
			} else {
				if hours == 1 {
					unit = "Hour"
				} else {
					unit = "Hours"
				}
				magnitude = hours
			}

			// Step 3: Format String
			e.Content.TimeElapsed = fmt.Sprintf("%d %s ago", magnitude, unit)
			e.Meta.Src = &promos[i].SrcURL

			if promos[i].Link != "" {
				e.Meta.CTALink = &promos[i].Link
			}

			pattern := regexp.MustCompile(`#(\w+)`)

			extraText := promos[i].Description

			matches := pattern.FindAllStringSubmatch(extraText, -1)

			for _, match := range matches {
				phrase := strings.ToLower(match[1])
				extraText = strings.Replace(extraText, match[0], fmt.Sprintf("<a class='text-blue-500' href='/feed/?hashtag=%s'>%s</a>", phrase, match[0]), 1)
			}

			extraTextHTML := template.HTML(extraText)

			e.Content.ExtraText = &extraTextHTML
			website, err := getWebsiteByID(promos[i].WebsiteID)
			if err != nil {
				return fmt.Errorf("Could not get website by id %d. %v", promos[i].WebsiteID, err)
			}
			e.Content.Summary = fmt.Sprintf("posted an update about %s", website.WebsiteName)
			e.Content.ExtraImages = nil
			events = append(events, e)
		}

		websites := getWebsites(10, 0)

		limit := 5
		q := `SELECT  hashtag_id,  count(post_id) FROM post_hashtags GROUP BY hashtag_idORDER BY count(post_id) DESC LIMIT ?`

		rows, err := db.Query(q, limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		top := []GetTopByPostCountResponse{}
		for rows.Next() {
			var row GetTopByPostCountResponse
			if err := rows.Scan(&row.HashtagID, &row.PostCount); err != nil {
				return err
			}
			top = append(top, row)
		} // should expect an array like {hashtag, postcount}
		if err != nil {
			return fmt.Errorf("Could not get postHashtags at GetTrending. %v", err)
		}

		var trendingHashtags []*Trending
		for _, row := range top {
			hashtag, err := getHashtagByID(db, row.HashtagID)
			if err != nil {
				return fmt.Errorf("Could not get hashtag by id at GetTrending in hashtagsvc. %w", err)
			}
			trendingHashtags = append(trendingHashtags, &Trending{Category: "Topic", Phrase: hashtag.Phrase, PostCount: row.PostCount})
		}

		if err != nil {
			return fmt.Errorf("Error trying to get trending hashtags in feed handler. %v", err)

		}

		var buf bytes.Buffer

		data := map[string]any{"Events": events, "Websites": websites, "Trending": trendingHashtags, "Request": r, "MenuItems": menuItems}
		if err := tmpl.ExecuteTemplate(&buf, "feedpage", data); err != nil {
			return fmt.Errorf("problem rebdering the feedpage template")
		}

		_, err = w.Write(buf.Bytes())
		if err != nil {
			return err
		}
		return nil
	}

}

func handleGetWebsites(db *sql.DB, tmpl *template.Template) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		limit, offset, page := paginator(r)
		websites := getWebsites(limit, offset)

		count := len(getWebsites(0, 0))

		maxPages := int(math.Ceil(float64(count) / float64(limit)))

		pagination := Pagination{page, maxPages}

		err := tmpl.ExecuteTemplate(w, "websites", map[string]any{"MenuItems": menuItems, "Request": r, "Websites": websites, "Pagination": pagination})
		if err != nil {
			return fmt.Errorf("Error: %v", err)
		}
		return nil
	}
}

func handleGetWebsiteByID(db *sql.DB, tmpl *template.Template) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		websiteID, err := strconv.Atoi(vars["website_id"])
		if err != nil {
			websiteID = 1
		}

		website, err := getWebsiteByID(websiteID)
		if err != nil {
			return fmt.Errorf("Error %v", err)
		}

		var buf bytes.Buffer
		err = tmpl.ExecuteTemplate(&buf, "website", map[string]any{"MenuItems": menuItems, "Request": r, "Website": website})
		if err != nil {
			return fmt.Errorf("Error %v", err)
		}

		if _, err := w.Write(buf.Bytes()); err != nil {
			return err
		}

		return nil
	}
}

func handleGetBrands(db *sql.DB, tmpl *template.Template) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		limit, offset, page := paginator(r)

		brands, err := getBrands(db, limit, offset)
		if err != nil {
			return fmt.Errorf("Error getting brands => %v", err)
		}

		brandCount, err := countAllBrands(db)
		if err != nil {
			return fmt.Errorf("Error counting brands => %v", err)
		}

		maxPages := int(math.Ceil(float64(brandCount) / float64(limit)))

		pagination := Pagination{page, maxPages}

		var buf bytes.Buffer
		err = tmpl.ExecuteTemplate(&buf, "brands", map[string]any{"MenuItems": menuItems, "Request": r, "Brands": brands, "Pagination": pagination})
		if err != nil {
			return err
		}

		if _, err := w.Write(buf.Bytes()); err != nil {
			return err
		}

		return nil
	}
}

func handlePostSubscribe(mode Mode, port, productionDomain string, db *sql.DB, tmpl *template.Template) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		err := r.ParseForm()

		email := r.FormValue("email")
		consent := r.FormValue("consent")

		if consent == "on" {

			q := `INSERT INTO subscribers(email, consent) VALUES (?, ?)`
			_, err := db.Exec(q, email, 1)
			if err != nil {
				return fmt.Errorf("could not insert email into subscibers table => %w", err)
			}

			// Calculate the required byte size based on the length of the token
			byteSize := 20 / 2 // Each byte is represented by 2 characters in hexadecimal encoding

			// Create a byte slice to store the random bytes
			randomBytes := make([]byte, byteSize)

			// Read random bytes from the crypto/rand package
			_, err = rand.Read(randomBytes)
			if err != nil {
				return err
			}

			// Encode the random bytes into a hexadecimal string
			token := fmt.Sprintf("%x", randomBytes)

			// TODO move gernate token to service level
			// TODO do one insert with both email and verification token

			setVerificationTokenQuery := `UPDATE subscribers SET verification_token = ?, is_verified = 0 WHERE email = ?`
			_, err = db.Exec(setVerificationTokenQuery, token, email)
			if err != nil {
				return fmt.Errorf("could not add verification token to user by email => %w", err)
			}

			domain := "http://localhost:" + port
			if mode == Prod {
				domain = productionDomain
			}

			log.Printf("Verify subscription at %s/subscribe/verify?token=%s", domain, token)
			/* TODO implement this in production. For now log to console
			err = s.SendVerificationToken(email, token)
				if err != nil {
					return fmt.Errorf("failed to send verification token => %w", err)
				}*/

			err = tmpl.ExecuteTemplate(w, "subscriptionsuccess", nil)
			if err != nil {
				return fmt.Errorf("Error: %v", err)
			}
		}

		err = tmpl.ExecuteTemplate(w, "subscriptionform", nil)
		if err != nil {
			return err
		}
		return nil
	}
}

func handleGetVerifySubscription(db *sql.DB, tmpl *template.Template) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		vars := r.URL.Query()
		token := vars.Get("token")

		if token == "" {
			// ("Warning: subscription verification attempted with no token")
			err := handleGetHomePage(db, tmpl)(w, r)
			if err != nil {
				return fmt.Errorf("problem calling handleHomePage from within handleVerifySubsription. %w", err)
			}
		}

		q := `UPDATE subscribers SET is_verified = 1 WHERE verification_token = ?`
		_, err := db.Exec(q, token)
		if err != nil {
			return fmt.Errorf("could not verify subscription via verification token => %w", err)
		}

		// subscription confirmed

		err = tmpl.ExecuteTemplate(w, "subscriptionverification", map[string]any{"MenuItems": menuItems, "Request": r})
		if err != nil {
			return fmt.Errorf("Error: could not render subscription verification page => %v", err)
		}
		return nil
	}
}

/* handlers end */

/* tempalte functions start*/

func add(a, b int) int {
	return a + b
}

func subtract(a, b int) int {
	return a - b
}

func formatLongDate(t time.Time) string {
	return t.Format("January 2, 2006")
}

func placeholderImage(url string) string {
	if url == "" {
		return "https://semantic-ui.com/images/avatar/small/jenny.jpg"
	}
	return url
}

func truncateDescription(description string) string {
	if len(description) < 100 {
		return description
	}
	return string(description[0:100] + "...")
}

func unescape(s string) template.HTML {
	return template.HTML(s)
}

func proper(str string) string {
	words := strings.Fields(str)
	for i, word := range words {
		words[i] = strings.ToUpper(string(word[0])) + word[1:]
	}
	return strings.Join(words, " ")
}

func isCurrentPage(r *http.Request, path string) bool {
	return r.URL.Path == path
}

func lower(s string) string {
	return strings.ToLower(s)
}

/* template functions end */

func paginator(r *http.Request) (int, int, int) {
	q := r.URL.Query()

	limit := 54
	offset := 0
	page := 1

	_limit := q.Get("limit")
	if _limit != "" {
		l, err := strconv.Atoi(_limit)
		if err != nil {
			log.Printf("Error parsing limit from %s, %v", r.URL.String(), err)
		} else {
			limit = l
		}
	}

	_offset := q.Get("offset")
	if _offset != "" {
		o, err := strconv.Atoi(_offset)
		if err != nil {
			log.Printf("Error parsing offset from %s, %v", r.URL.String(), err)
		} else {
			offset = o
		}
	}

	_page := q.Get("page")
	if _page != "" {
		p, err := strconv.Atoi(_page)
		if err != nil {
			log.Printf("Error parsing page from %s, %v", r.URL.String(), err)
		} else {
			page = p
			offset = (limit * (p - 1))
		}
	}

	return limit, offset, page
}

/* db funcs */

var dbNil = errors.New("db is nil")

/* Brand DB Funcs */
func getBrandByID(db *sql.DB, id int) (*Brand, error) {
	if db == nil {
		return nil, dbNil
	}
	q := `SELECT id, name, path FROM brands WHERE id = ?`
	var brand Brand
	err := db.QueryRow(q, id).Scan(&brand.ID, &brand.Name, &brand.Path)
	if err != nil {
		return nil, err
	}
	return &brand, nil
}

func getBrands(db *sql.DB, limit, offset int) ([]*Brand, error) {
	if db == nil {
		return nil, dbNil
	}

	q := `SELECT id, name, path FROM brands ORDER BY name ASC LIMIT ? OFFSET ?`

	rows, err := db.Query(q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("could not get brands from db %v", err)
	}
	defer rows.Close()

	brands := []*Brand{}
	for rows.Next() {
		var brand Brand
		err = rows.Scan(&brand.ID, &brand.Name, &brand.Path)
		if err != nil {
			return nil, err
		}
		brands = append(brands, &brand)
	}

	return brands, nil

}

func countAllBrands(db *sql.DB) (int, error) {
	if db == nil {
		return -1, dbNil
	}

	q := `SELECT count(id) as brand_count FROM Brands`
	var count int
	err := db.QueryRow(q).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("could not count brands => %w", err)
	}
	return count, nil
}

func insertBrand(db *sql.DB, brand *Brand) (int, error) {
	if db == nil {
		return -1, dbNil
	}

	if brand == nil {
		return -1, errors.New("brand passed to insert brand is nil")
	}
	q := `INSERT INTO brands(name, path) VALUES (?, ?)`
	res, err := db.Exec(q, brand.Name, brand.Path)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func countBrandsByName(db *sql.DB, brandName string) (int, error) {
	if db == nil {
		return -1, dbNil
	}

	q := `SELECT count(id) FROM Brands WHERE name = ?`
	var count int
	err := db.QueryRow(q, brandName).Scan(&count)
	if err != nil {
		return -1, err
	}
	return count, nil
}

func getBrandByName(db *sql.DB, brandName string) (*Brand, error) {
	if db == nil {
		return nil, dbNil
	}

	q := `SELECT id, name, path FROM brands WHERE name = ?`
	var brand Brand
	err := db.QueryRow(q, brandName).Scan(&brand.ID, &brand.Name, &brand.Path)
	if err != nil {
		return nil, err
	}
	return &brand, nil
}

func getBrandByPath(db *sql.DB, brandPath string) (*Brand, error) {
	if db == nil {
		return nil, dbNil
	}

	q := `SELECT id, name, path FROM brands WHERE path = ?`
	var brand Brand
	err := db.QueryRow(q, brandPath).Scan(&brand.ID, &brand.Name, &brand.Path)
	if err != nil {
		return nil, fmt.Errorf("could not get brand for path %s => %w", brandPath, err)
	}
	return &brand, nil
}

func updateBrand(db *sql.DB, brand *Brand) error {
	if db == nil {
		return dbNil
	}

	if brand == nil {
		return errors.New("brand passed to updateBrand is nil")
	}
	if brand.ID < 1 {
		return errors.New("Need to supply a brand id to update")
	}
	q := `UPDATE brands SET name = ?, path = ? WHERE id = ?`
	_, err := db.Exec(q, brand.Name, brand.Path, brand.ID)
	if err != nil {
		return err
	}
	return nil
}

func deleteBrandByID(db *sql.DB, brandID int) error {
	if db == nil {
		return dbNil
	}

	if brandID < 1 {
		return errors.New("Need to supply a valid brand id to update")
	}

	q := `DELETE FROM brands WHERE id = ?`
	if _, err := db.Exec(q, brandID); err != nil {
		return err
	}

	return nil
}

/*Hashtag db funcs*/

func insertHashtag(db *sql.DB, h *Hashtag) (lastInsertID int, err error) {
	if db == nil {
		return -1, dbNil
	}

	if h == nil {
		return -1, errors.New("hashtag passed to insert hashtag is nil")
	}

	res, err := db.Exec(`INSERT INTO hashtags (phrase) VALUES (?)`, h.Phrase)
	if err != nil {
		return -1, err
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		return -1, err
	}

	return int(lastID), nil
}

func countHashtagsByPhrase(db *sql.DB, phrase string) (int, error) {
	if db == nil {
		return -1, dbNil
	}

	var count int
	err := db.QueryRow(`SELECT COUNT(id) FROM hashtags WHERE phrase = ?`, phrase).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func getHashtagIDByPhrase(db *sql.DB, phrase string) (int, error) {
	if db == nil {
		return -1, dbNil
	}

	q := `SELECT id FROM hashtags WHERE phrase = ?`
	var h int
	err := db.QueryRow(q, phrase).Scan(&h)
	if err != nil {
		return 0, err
	}
	return h, nil
}

func getHashtagByID(db *sql.DB, id int) (*Hashtag, error) {
	if db == nil {
		return nil, dbNil
	}

	q := `SELECT id, phrase from hashtags WHERE id = ?`
	var h Hashtag
	if err := db.QueryRow(q, id).Scan(&h.ID, &h.Phrase); err != nil {
		return nil, err
	}
	return &h, nil
}

/* persona db funcs*/

// get all
func getAllPersonas(db *sql.DB) ([]*Persona, error) {
	if db == nil {
		return nil, dbNil
	}

	q := `SELECT id, name, description, profile_photo FROM models`
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []*Persona
	for rows.Next() {
		var p Persona
		err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.ProfilePhoto)
		if err != nil {
			return nil, err
		}
		res = append(res, &p)
	}
	return res, nil
}

// get one persona
func getPersonaByID(db *sql.DB, id int) (*Persona, error) {
	if db == nil {
		return nil, dbNil
	}

	if id < 1 {
		return nil, fmt.Errorf("please supply a valid id")
	}

	return nil, fmt.Errorf("Not yet implmented")
}

// get one random persona
func getRandomPersona(db *sql.DB) (*Persona, error) {
	if db == nil {
		return nil, dbNil
	}

	q := `SELECT id, name, description, profile_photo FROM models ORDER BY RANDOM() LIMIT 1`
	var p Persona
	err := db.QueryRow(q).Scan(&p.ID, &p.Name, &p.Description, &p.ProfilePhoto)
	if err != nil {
		return nil, fmt.Errorf("error retrieving random persona. %w", err)
	}
	return &p, nil
}

// update one persona
func updatePersona(db *sql.DB, p *Persona) error {
	if db == nil {
		return dbNil
	}

	if p == nil {
		return errors.New("persona passed to updatePersona is nil")
	}

	q := `UPDATE models SET name = ?, description = ?, profile_photo = ? WHERE id = ?`
	_, err := db.Exec(q, p.Name, p.Description, p.ProfilePhoto, p.ID)
	if err != nil {
		return err
	}
	return nil
}

// delete one persona
func deletePersonaByID(db *sql.DB, id int) error {
	if db == nil {
		return dbNil
	}

	q := `DELETE FROM models WHERE id = ?`
	_, err := db.Exec(q, id)
	if err != nil {
		return err
	}
	return nil
}

/* posthashtag db funcs*/

func insertPostHashtagRelationship(db *sql.DB, postID, hashtagID int) error {

	if db == nil {
		return dbNil
	}

	_, err := db.Exec("INSERT INTO post_hashtags(post_id, hashtag_id) VALUES (?, ?)", postID, hashtagID)
	if err != nil {
		return err
	}
	return nil
}

type GetTopByPostCountResponse struct {
	HashtagID int
	PostCount int
}

func initPostHashTagTable(db *sql.DB) error {

	if db == nil {
		return dbNil
	}

	q := `CREATE TABLE 
	post_hashtags(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id  INTEGER NOT NULL,
		hashtag_id INTEGER NOT NULL
	);`
	if _, err := db.Exec(q); err != nil {
		return err
	}
	return nil
}

// TODO update these functions to addres websiteID column
func updatePost(db *sql.DB, p *Post) error {

	if db == nil {
		return dbNil
	}

	if p == nil {
		return errors.New("post past to update post is nil")
	}

	q := `UPDATE 
		posts 
	SET 
		author_id = ?, 
		srcURL = ?, 
		websiteID = ?, 
		description = ?, 
		link = ? 
	WHERE 
		id = ?`

	if _, err := db.Exec(
		q,
		p.AuthorID,
		p.SrcURL,
		p.WebsiteID,
		p.Description,
		p.Link,
		p.ID,
	); err != nil {
		return err
	}

	return nil
}

type getPostParams struct {
	WebsiteID           int
	SortByTimestampDesc bool
	IDs                 []int
}

func getPosts(db *sql.DB, params getPostParams) ([]*Post, error) {

	if db == nil {
		return nil, dbNil
	}

	promos := []*Post{}

	q := `SELECT id, websiteID, srcURL, author_id, description, timestamp FROM posts`

	args := []any{}
	if params.WebsiteID != 0 || len(params.IDs) > 0 {
		subQ := []string{}
		if params.WebsiteID != 0 {
			subQ = append(subQ, "websiteID = ?")
			args = append(args, params.WebsiteID)
		}

		if len(params.IDs) > 0 {
			idQuery := "id IN ("
			idQuery += strings.Repeat("?, ", len(params.IDs)-1)
			idQuery += "?)"
			subQ = append(subQ, idQuery)
			for _, id := range params.IDs {
				args = append(args, id)
			}
		}
		q += " WHERE " + strings.Join(subQ, " AND ")
	}

	if params.SortByTimestampDesc {
		q += " ORDER BY timestamp DESC"
	}

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var promo Post
		err = rows.Scan(&promo.ID, &promo.WebsiteID, &promo.SrcURL, &promo.AuthorID, &promo.Description, &promo.Timestamp)
		if err != nil {
			return nil, err
		}
		promos = append(promos, &promo)
	}

	return promos, nil

}

/*
CREATE TABLE subscribers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    full_name TEXT,
    consent BOOLEAN NOT NULL CHECK (consent IN (0, 1)),
    signup_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    verification_token TEXT UNIQUE,
    is_verified BOOLEAN DEFAULT 0,
    preferences TEXT,
    CONSTRAINT chk_preferences CHECK (json_valid(preferences))
);
*/

func subscribe(db *sql.DB, email string) error {

	if db == nil {
		return dbNil
	}

	return nil
}

/* website funcs*/

// Retrieve a website by its ID from the Websites table
func getWebsiteByID(websiteID int) (Website, error) {
	for _, website := range getWebsites(0, 0) {
		if website.WebsiteID == websiteID {
			return website, nil
		}
	}
	return Website{}, fmt.Errorf("no website with id %d", websiteID)
}

// Retrieve a website by its ID from the Websites table
func getWebsiteByName(websiteName string) (Website, error) {
	for _, website := range getWebsites(0, 0) {
		if strings.ToLower(website.WebsiteName) == strings.ToLower(websiteName) {
			return website, nil
		}
	}
	return Website{}, fmt.Errorf("no website with name %s", websiteName)
}

func getWebsites(limit, offset int) []Website {
	websites := []Website{
		{1, "BeautyFeatures", "https://www.beautyfeatures.ie", "IE"},
		{2, "LookFantastic", "https://lookfantastic.ie", "IE"},
		{3, "Millies", "https://millies.ie", "IE"},
		{4, "McCauley Pharmacy", "https://www.mccauley.ie/", "IE"},
	}

	if limit == 0 {
		limit = len(websites)
	}

	toReturn := []Website{}
	for i := offset; i < limit; i++ {
		toReturn = append(toReturn, websites[i])
	}
	return toReturn
}

/*
For a known website, retreive the banner urls and supply them in a string slice
*/
func extractWebsiteBannerURLs(website Website) ([]string, error) {
	res, err := http.Get(website.URL)
	if err != nil {
		return []string{}, fmt.Errorf("error sending get request to extract banner urls %w", err)
	}

	defer res.Body.Close()

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return []string{}, fmt.Errorf("error parsing document with go query %w", err)
	}

	bannerURLs := []string{}

	switch website.WebsiteID {
	case 1:
		// beautyfeatures
		doc.Find("picture img").Each(func(i int, s *goquery.Selection) {
			// For each item found, get the title
			value, found := s.Attr("src")
			if found {
				bannerURLs = append(bannerURLs, value)
			}
		})
	case 2:
		// lookfantastic
		return []string{}, fmt.Errorf("Could not find banner extraction rules for website %s", website.WebsiteName)
	case 3:
		// millies
		milliesBanners := []string{}
		doc.Find(".homepage-slider-parent .swiper-wrapper img").Each(func(i int, s *goquery.Selection) {
			// For each item found, get the title
			value, found := s.Attr("data-src")
			if found {
				value = strings.ReplaceAll(value, "{width}", "800")
				if strings.HasPrefix(value, "//") {
					value = "https:" + value
				}
				milliesBanners = append(milliesBanners, value)
			}
		})
		for i := 0; i < len(milliesBanners); i += 2 {
			bannerURLs = append(bannerURLs, milliesBanners[i])
		}
	case 4:
		// mcCauleys
		doc.Find("[data-content-type=slide] [data-background-images]").Each(func(i int, s *goquery.Selection) {
			value, found := s.Attr("data-background-images")
			if found {
				type MCBackgroundImage struct {
					MobileImage string `json:"mobile_image"`
				}

				var x MCBackgroundImage
				value = strings.ReplaceAll(value, "\\\"", "\"")
				err = json.Unmarshal([]byte(value), &x)
				if err == nil {
					bannerURLs = append(bannerURLs, x.MobileImage)
				}
			}
		})
	default:
		return []string{}, fmt.Errorf("Could not find banner extraction rules for website %s", website.WebsiteName)
	}

	return bannerURLs, nil
}

/* chat service begins */
func generateOfferDescription(websiteName, imageURL string) (string, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return "", errors.New("OPENAI_API_KEY env var not set")
	}

	c := openai.NewClient(key)

	command := fmt.Sprintf("You are a joyful and excited social media manager for a health and beauty magazine with the goal of motivating people to take advantage of today's available beauty offers. Tell your audience what the beauty retailer %s is advertising today and highlight any coupons if available. Keep your response short, playful and suitable for a tweet or instagram caption.", websiteName)
	model := openai.GPT4VisionPreview

	res, err := c.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model:     model,
		MaxTokens: 1000,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: "user",
				MultiContent: []openai.ChatMessagePart{
					{
						Type: "text",
						Text: command,
					},
					{
						Type:     "image_url",
						ImageURL: &openai.ChatMessageImageURL{URL: imageURL},
					},
				},
			},
		},
	})

	if err != nil {
		return "", err
	}

	return res.Choices[0].Message.Content, nil

}

/* chat service ends */
