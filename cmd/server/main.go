package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
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

	"github.com/gorilla/mux"
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

	db, err := sql.Open("sqlite3", "main.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

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
		return fmt.Errorf("Port is required via -port flag.")
	}

	if mode == "" {
		log.Println("No mode was supplied, starting server in development mode.")
		mode = Dev
	}

	tmpl, err := template.New("web").Funcs(getFuncMap()).ParseGlob("web/templates/**/*.tmpl")
	if err != nil {
		return fmt.Errorf("Error parsing templates. %v", err)
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

	if mode == Dev {
		handle("/websites/add/", handleGetAddWebsitePage(db, tmpl)).Methods(http.MethodGet)
		handle("/add-website/", handlePostAddWebsiteFormSubmission(db, tmpl)).Methods(http.MethodPost)
		handle("/websites/", handleGetWebsites(db, tmpl)).Methods(http.MethodGet)
		handle("/websites/{website_id}/", handleGetWebsiteByID(db, tmpl)).Methods(http.MethodGet)
		handle("/brands/", handleGetBrands(db, tmpl)).Methods(http.MethodGet)
		// handle("/brands/{brand_path}/", handleGetBrandByPath(db, tmpl)).Methods(http.MethodGet)
		handle("/subscribe/", handlePostSubscribe(db, tmpl)).Methods(http.MethodPost)
		// handle("/subscribe/", handleGetSubscribePage(db, tmpl)).Methods(http.MethodGet)
		handle("/subscribe/verify", handleGetVerifySubscription(db, tmpl)).Methods(http.MethodGet)
	}

	log.Println("Server listening on http://localhost:" + port)
	if err = http.ListenAndServe(":"+port, r); err != nil {
		return fmt.Errorf("Error during listen and serve. %v", err)
	}
	return nil
}

func processHashtags(db *sql.DB) error {

	/*
		Get all posts. At some point I will have to implement a way to filter for posts
		that have not already been processed
	*/
	posts, err := getAllPosts(db, getAllPostParams{})
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

				count, err := countPostHashtagRelationships(db, p.ID, hashtagID)
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

		params := getAllPostParams{
			SortByTimestampDesc: true,
		}

		if websiteName != "" {
			website, err := getWebsiteByName(db, websiteName)
			if err == nil {
				params.WebsiteID = website.WebsiteID
			}
		}

		promos, err := getAllPosts(db, params)
		if err != nil {
			return fmt.Errorf("Error: %w", err)
		}

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

		type GetPostParams struct {
			WebsiteID           int
			SortByTimestampDesc bool
			Hashtag             string
		}

		params := GetPostParams{
			SortByTimestampDesc: true,
		}

		if websiteName != "" {
			website, err := getWebsiteByName(db, websiteName)
			if err == nil {
				params.WebsiteID = website.WebsiteID
			} else {
				log.Printf("Warning: User tried to get posts for %s. %v", websiteName, err)
			}
		}

		if hashtagQuery != "" {
			params.Hashtag = hashtagQuery
		}

		promos, err := func(params GetPostParams) ([]*Post, error) {

			var postIDs []int
			if params.Hashtag != "" {
				hashtagID, err := getHashtagIDByPhrase(db, params.Hashtag)
				if err != nil {
					return nil, fmt.Errorf("Could not get hashtag id in get by phrase. %w", err)
				}
				ids, err := getPostIDsByHashtagID(db, hashtagID)
				if err != nil {
					return nil, fmt.Errorf("Could not get post ids for hashtag id, %d phrase: %s. %w", hashtagID, params.Hashtag, err)
				}
				postIDs = ids
			}

			repoParams := getAllPostParams{}
			repoParams.IDs = postIDs
			repoParams.SortByTimestampDesc = params.SortByTimestampDesc
			repoParams.WebsiteID = params.WebsiteID
			p, err := getAllPosts(db, repoParams)
			if err != nil {
				return nil, fmt.Errorf("Error with postrepo GetAll func at postsvc.GetAll. %w", err)
			}
			return p, nil
		}(params)
		if err != nil {
			return fmt.Errorf("Could not get banner promotions for feed page.  %v", err)
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
			website, err := getWebsiteByID(db, promos[i].WebsiteID)
			if err != nil {
				return fmt.Errorf("Could not get website by id %d. %v", promos[i].WebsiteID, err)
			}
			e.Content.Summary = fmt.Sprintf("posted an update about %s", website.WebsiteName)
			e.Content.ExtraImages = nil
			events = append(events, e)
		}

		websites, err := getAllWebsites(db, 10, 0)
		if err != nil {
			return err
		}

		trendingHashtags, err := func(limit int) ([]*Trending, error) {
			top, err := getTopHashtagByPostCount(db, limit) // should expect an array like {hashtag, postcount}
			if err != nil {
				return nil, fmt.Errorf("Could not get postHashtags at GetTrending. %v", err)
			}
			var trending []*Trending
			for _, row := range top {
				hashtag, err := getHashtagByID(db, row.HashtagID)
				if err != nil {
					return nil, fmt.Errorf("Could not get hashtag by id at GetTrending in hashtagsvc. %w", err)
				}
				trending = append(trending, &Trending{Category: "Topic", Phrase: hashtag.Phrase, PostCount: row.PostCount})
			}
			return trending, nil
		}(5)

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

func handleGetAddWebsitePage(db *sql.DB, tmpl *template.Template) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := tmpl.ExecuteTemplate(w, "addwebsite", map[string]any{"MenuItems": menuItems, "Request": r}); err != nil {
			return err
		}
		return nil
	}
}

func handleGetWebsites(db *sql.DB, tmpl *template.Template) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		limit, offset, page := paginator(r)
		websites, err := getAllWebsites(db, limit, offset)
		if err != nil {
			return err
		}

		count, err := countWebsites(db)
		if err != nil {
			return err
		}

		maxPages := int(math.Ceil(float64(count) / float64(limit)))

		pagination := Pagination{page, maxPages}

		err = tmpl.ExecuteTemplate(w, "websites", map[string]any{"MenuItems": menuItems, "Request": r, "Websites": websites, "Pagination": pagination})
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
			return fmt.Errorf("Error %v", err)
		}

		website, err := getWebsiteByID(db, websiteID)
		if err != nil {
			return fmt.Errorf("Error %v", err)
		}

		err = tmpl.ExecuteTemplate(w, "website", map[string]any{"MenuItems": menuItems, "Request": r, "Website": website})
		if err != nil {
			return fmt.Errorf("Error %v", err)
		}
		return nil
	}
}

func handlePostAddWebsiteFormSubmission(db *sql.DB, tmpl *template.Template) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		err := r.ParseForm()
		if err != nil {
			return err
		}

		name := r.Form.Get("name")
		url := r.Form.Get("url")
		country := r.Form.Get("country")

		website := Website{WebsiteName: name, URL: url, Country: country}

		err = createWebsite(db, &website)
		if err != nil {
			return err
		}

		return handleGetWebsites(db, tmpl)(w, r)
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

		err = tmpl.ExecuteTemplate(w, "brands", map[string]any{"MenuItems": menuItems, "Request": r, "Brands": brands, "Pagination": pagination})
		if err != nil {
			return err
		}
		return nil
	}
}

func handlePostSubscribe(db *sql.DB, tmpl *template.Template) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		err := r.ParseForm()

		email := r.FormValue("email")
		consent := r.FormValue("consent")

		if consent == "on" {
			err = subscribe(db, email)
			if err != nil {
				return fmt.Errorf("Error subscribing user: %v", err)
			}

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

		err := verifySubscription(db, token)
		if err != nil {
			return fmt.Errorf("Error: could not verify subscription => %v", err)
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

/*
utility functions
*/
func getFuncMap() template.FuncMap {
	add := func(a, b int) int {
		return a + b
	}
	subtract := func(a, b int) int {
		return a - b
	}
	formatLongDate := func(t time.Time) string {
		return t.Format("January 2, 2006")
	}
	placeholderImage := func(url string) string {
		if url == "" {
			return "https://semantic-ui.com/images/avatar/small/jenny.jpg"
		}
		return url
	}
	truncateDescription := func(description string) string {
		if len(description) < 100 {
			return description
		}
		return string(description[0:100] + "...")
	}
	unescape := func(s string) template.HTML {
		return template.HTML(s)
	}
	proper := func(str string) string {
		words := strings.Fields(str)
		for i, word := range words {
			words[i] = strings.ToUpper(string(word[0])) + word[1:]
		}
		return strings.Join(words, " ")
	}
	isCurrentPage := func(r *http.Request, path string) bool {
		return r.URL.Path == path
	}
	lower := func(s string) string {
		return strings.ToLower(s)
	}
	return template.FuncMap{
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

}

func VerifyUnverifiedEmails(
	getUnverifiedEmails func() ([]string, error),
	generateEmailVerificationToken func(int) (string, error),
	addVerficationTokenToEmailRecord func(email, token string) error,
	sendVerificationEmail func(email, token string) error,
) error {

	emails, err := getUnverifiedEmails()

	if err != nil {
		return fmt.Errorf("Error: %w", err)
	}

	for _, email := range emails {
		token, err := generateEmailVerificationToken(10)
		if err != nil {
			return fmt.Errorf("error generating token: %w", err)
		}

		err = addVerficationTokenToEmailRecord(email, token)
		if err != nil {
			return fmt.Errorf("error adding verification token to email record: %w", err)
		}

		err = sendVerificationEmail(email, token)
		if err != nil {
			return fmt.Errorf("error sending verification token: %w", err)
		}

	}
	return nil
}

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

func getTopHashtagByPostCount(db *sql.DB, limit int) ([]GetTopByPostCountResponse, error) {

	if db == nil {
		return nil, dbNil
	}

	q := `SELECT 
		hashtag_id, 
		count(post_id) 
	FROM 
		post_hashtags 
	GROUP BY
		hashtag_id
	ORDER BY 
		count(post_id) 
	DESC 
	LIMIT ?`

	rows, err := db.Query(q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := []GetTopByPostCountResponse{}
	for rows.Next() {
		var row GetTopByPostCountResponse
		if err := rows.Scan(&row.HashtagID, &row.PostCount); err != nil {
			return nil, err
		}
		res = append(res, row)
	}
	return res, nil
}

func countPostHashtagRelationships(db *sql.DB, postID, hashtagID int) (int, error) {

	if db == nil {
		return -1, dbNil
	}

	q := `SELECT 
		count(*) 
	FROM 
		post_hashtags 
	WHERE 
		post_id = ? 
	AND 
		hashtag_id = ?`

	var count int
	if err := db.QueryRow(q, postID, hashtagID).Scan(&count); err != nil {
		return 0, fmt.Errorf("Could not count relationships between %d & %d. %v", postID, hashtagID, err)
	}
	return count, nil
}

func getPostIDsByHashtagID(db *sql.DB, hashtagID int) ([]int, error) {

	if db == nil {
		return nil, dbNil
	}

	rows, err := db.Query("SELECT post_id FROM post_hashtags WHERE hashtag_id = ?", hashtagID)
	if err != nil {
		return []int{}, fmt.Errorf("Error getting post_ids from post_hashtags db. %w", err)
	}
	defer rows.Close()

	res := []int{}
	for rows.Next() {
		var i int
		err := rows.Scan(&i)
		if err != nil {
			return []int{}, fmt.Errorf("Error scanning post_id in getposts. %w", err)
		}
		res = append(res, i)
	}

	return res, nil
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

type getAllPostParams struct {
	WebsiteID           int
	SortByTimestampDesc bool
	IDs                 []int
}

func getAllPosts(db *sql.DB, params getAllPostParams) ([]*Post, error) {

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

func countPostsBySrc(db *sql.DB, imgSrc string) (int, error) {

	if db == nil {
		return -1, dbNil
	}

	var count int
	err := db.QueryRow(`SELECT count(id) FROM posts WHERE srcURL = ?`, imgSrc).Scan(&count)
	if err != nil {
		return -1, err
	}
	return count, nil
}

func insertNewPost(db *sql.DB, websiteID int, url string, authorID int, description string, timestamp time.Time) error {

	if db == nil {
		return dbNil
	}

	_, err := db.Exec(
		"INSERT INTO posts(websiteID, srcURL, author_id, description, timestamp) VALUES (? , ? , ?, ?, ?)",
		websiteID, url, authorID, description, timestamp) // time.Now()

	if err != nil {
		return err
	}

	return nil
}

/*images models db  funcs*/

func getDistinctNamesFromImages(db *sql.DB, randomOrder bool) ([]string, error) {

	if db == nil {
		return nil, dbNil
	}

	names := []string{}
	q := "SELECT DISTINCT name FROM images"
	if randomOrder {
		q += " ORDER BY RANDOM()"
	}

	rows, err := db.Query(q)
	if err != nil {
		return names, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		if err != nil {
			return names, err
		}
		names = append(names, name)
	}

	return names, nil
}

func getRandomImages(db *sql.DB, limit int) ([]*ProfilePhoto, error) {

	if db == nil {
		return nil, dbNil
	}

	images := []*ProfilePhoto{}
	q := "SELECT id, file_path as url, name FROM images ORDER BY RANDOM() LIMIT ?"
	rows, err := db.Query(q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		image := ProfilePhoto{}
		err = rows.Scan(&image.ID, &image.URL, &image.Name)
		if err != nil {
			return nil, err
		}
		images = append(images, &image)
	}

	return images, nil
}

func getRandomModelImages(db *sql.DB, modelID int, limit int) ([]*ProfilePhoto, error) {

	if db == nil {
		return nil, dbNil
	}

	q := `SELECT 
		id, 
		file_path as url, 
		name 
	FROM 
		images 
	WHERE 
		model_id = ? 
	ORDER BY RANDOM() 
	LIMIT ?`

	rows, err := db.Query(q, modelID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := []*ProfilePhoto{}
	for rows.Next() {
		var image ProfilePhoto

		if err = rows.Scan(
			&image.ID,
			&image.URL,
			&image.Name,
		); err != nil {
			return nil, err
		}

		images = append(images, &image)
	}

	return images, nil
}

func deleteImageByID(db *sql.DB, id int) error {

	if db == nil {
		return dbNil
	}

	q := "DELETE FROM images WHERE  id = ?"
	_, err := db.Exec(q, id)
	if err != nil {
		return err
	}
	return nil

}

func getAllImages(db *sql.DB) ([]*ProfilePhoto, error) {

	if db == nil {
		return nil, dbNil
	}

	images := []*ProfilePhoto{}
	q := "SELECT id, file_path as url, name FROM images"
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		image := ProfilePhoto{}
		err := rows.Scan(&image.ID, &image.URL, &image.Name)
		if err != nil {
			return nil, err
		}
		images = append(images, &image)
	}

	return images, nil
}

func getAllModelImages(db *sql.DB, name string) ([]*ProfilePhoto, error) {
	if db == nil {
		return nil, dbNil
	}

	images := []*ProfilePhoto{}
	q := "SELECT id, file_path as url, name FROM images WHERE name = ?"
	rows, err := db.Query(q, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		image := ProfilePhoto{}
		err := rows.Scan(&image.ID, &image.URL, &image.Name)
		if err != nil {
			return nil, err
		}
		images = append(images, &image)
	}

	return images, nil
}

func getImageByID(db *sql.DB, id int) (*ProfilePhoto, error) {
	if db == nil {
		return nil, dbNil
	}

	image := ProfilePhoto{}
	q := "SELECT id, file_path as url, name FROM images WHERE id = ?"
	err := db.QueryRow(q, id).Scan(&image.ID, &image.URL, &image.Name)
	if err != nil {
		return nil, err
	}
	return &image, nil
}

func getImageFilePathByID(db *sql.DB, id int) (string, error) {
	if db == nil {
		return "", dbNil
	}
	var filepath string
	q := "SELECT file_path FROM images WHERE id = ?"
	err := db.QueryRow(q, id).Scan(&filepath)
	if err != nil {
		return filepath, err
	}
	return filepath, nil
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

	q := `INSERT INTO subscribers(email, consent) VALUES (?, ?)`
	_, err := db.Exec(q, email, 1)
	if err != nil {
		return fmt.Errorf("could not insert email into subscibers table => %w", err)
	}

	token, err := generateToken(20)
	if err != nil {
		return fmt.Errorf("failed to generate verification token at service.Subscribe => %w", err)
	}

	// TODO move gernate token to service level
	// TODO do one insert with both email and verification token

	err = addVerificationToken(db, email, token)
	if err != nil {
		return fmt.Errorf("failed to add verification to subscriber record => %w", err)
	}

	log.Printf("Verify subscription at https://develop.implicitdev.com/subscribe/verify?token=%s", token)
	/* TODO implement this in production. For now log to console
	err = s.SendVerificationToken(email, token)
		if err != nil {
			return fmt.Errorf("failed to send verification token => %w", err)
		}*/

	return nil
}
func getUnverifiedEmails(db *sql.DB) ([]string, error) {

	if db == nil {
		return nil, dbNil
	}

	q := `SELECT email FROM subscribers WHERE is_verified = 0`
	rows, err := db.Query(q)
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()

	emails := []string{}

	for rows.Next() {
		var email string
		err = rows.Scan(&email)
		if err != nil {
			return []string{}, err
		}
		emails = append(emails, email)
	}

	return emails, nil

}

func addVerificationToken(db *sql.DB, email, verificationToken string) error {
	if db == nil {
		return dbNil
	}
	q := `UPDATE subscribers SET verification_token = ?, is_verified = 0 WHERE email = ?`
	_, err := db.Exec(q, verificationToken, email)
	if err != nil {
		return fmt.Errorf("could not add verification token to user by email => %w", err)
	}
	return nil
}

func generateToken(length int) (string, error) {

	// Calculate the required byte size based on the length of the token
	byteSize := length / 2 // Each byte is represented by 2 characters in hexadecimal encoding

	// Create a byte slice to store the random bytes
	randomBytes := make([]byte, byteSize)

	// Read random bytes from the crypto/rand package
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	// Encode the random bytes into a hexadecimal string
	token := fmt.Sprintf("%x", randomBytes)

	return token, nil
}

func verifySubscription(db *sql.DB, verificationToken string) error {
	if db == nil {
		return dbNil
	}
	q := `UPDATE subscribers SET is_verified = 1 WHERE verification_token = ?`
	_, err := db.Exec(q, verificationToken)
	if err != nil {
		return fmt.Errorf("could not verify subscription via verification token => %w", err)
	}
	return nil
}

/* website db funcs*/

// Create a new website and insert it into the Websites table
func createWebsite(db *sql.DB, website *Website) error {
	if db == nil {
		return dbNil
	}
	if website == nil {
		return errors.New("website passed to createWebsite is nil")
	}
	_, err := db.Exec("INSERT INTO Websites (WebsiteName, URL, Country) VALUES (?, ?, ?)",
		website.WebsiteName, website.URL, website.Country)
	if err != nil {
		return err
	}
	return nil
}

// Retrieve a website by its ID from the Websites table
func getWebsiteByID(db *sql.DB, websiteID int) (*Website, error) {
	if db == nil {
		return nil, dbNil
	}
	var website Website
	err := db.QueryRow("SELECT WebsiteID, WebsiteName, URL, Country FROM Websites WHERE WebsiteID = ?", websiteID).
		Scan(&website.WebsiteID, &website.WebsiteName, &website.URL, &website.Country)
	if err != nil {
		return nil, err
	}
	return &website, nil
}

// Retrieve a website by its ID from the Websites table
func getWebsiteByName(db *sql.DB, websiteName string) (*Website, error) {
	if db == nil {
		return nil, dbNil
	}
	var website Website
	err := db.QueryRow("SELECT WebsiteID, WebsiteName, URL, Country FROM Websites WHERE LOWER(WebsiteName) = ?", websiteName).
		Scan(&website.WebsiteID, &website.WebsiteName, &website.URL, &website.Country)
	if err != nil {
		return nil, err
	}
	return &website, nil
}
func countWebsites(db *sql.DB) (int, error) {
	if db == nil {
		return -1, dbNil
	}
	q := `SELECT count(WebsiteID) FROM Websites`
	var count int
	err := db.QueryRow(q).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func getAllWebsites(db *sql.DB, limit, offset int) ([]*Website, error) {
	if db == nil {
		return nil, dbNil
	}
	websites := []*Website{}
	rows, err := db.Query("SELECT WebsiteID, WebsiteName, URL, Country FROM websites LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		website := Website{}
		err = rows.Scan(&website.WebsiteID, &website.WebsiteName, &website.URL, &website.Country)
		if err != nil {
			return nil, err
		}
		websites = append(websites, &website)
	}

	return websites, nil
}

// Update an existing website in the Websites table
func updateWebsite(db *sql.DB, website *Website) error {
	if db == nil {
		return dbNil
	}
	if website == nil {
		return errors.New("website passed to update website is nil")
	}
	_, err := db.Exec("UPDATE Websites SET WebsiteName = ?, URL = ?, Country = ? WHERE WebsiteID = ?",
		website.WebsiteName, website.URL, website.Country, website.WebsiteID)
	if err != nil {
		return err
	}
	return nil
}

// Delete a website by its ID from the Websites table
func deleteWebsite(db *sql.DB, websiteID int) error {
	if db == nil {
		return dbNil
	}
	_, err := db.Exec("DELETE FROM Websites WHERE WebsiteID = ?", websiteID)
	if err != nil {
		return err
	}
	return nil
}

/* chat service begins */
func generateOfferDescription(websiteName, url string) (string, error) {
	c := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

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
						ImageURL: &openai.ChatMessageImageURL{URL: url},
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
