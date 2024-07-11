package main

import (
	"bytes"
	"context"

	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math"
	"math/rand"
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

// Website struct matching the Websites table
type Website struct {
	WebsiteID   int    `json:"website_id"`
	WebsiteName string `json:"website_name"`
	URL         string `json:"url"`
	Country     string `json:"country"`
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

type renderFunc func(name string, data any) ([]byte, error)
type htmlHandleFunc func(w http.ResponseWriter, r *http.Request) ([]byte, error)

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

type ExtraImage struct {
	Src string
	Alt string
}

type Content struct {
	Summary     string
	TimeElapsed string
	ExtraImages *[]ExtraImage  // optional
	ExtraText   *template.HTML // optional
}

type EventMeta struct {
	CTALink *string
	Src     *string
	Likes   int
}

type Event struct {
	Profile Profile
	Content Content
	Meta    EventMeta
}

/* models end */

var menuItems = []MenuItem{
	{"/", "Home"},
	{"/promotions/", "Promotions"},
	{"/websites/", "Websites"},
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
	/*
		if err := extractOffersFromBanners(db); err != nil {
			log.Fatal(err)
		}

		if err := processHashtags(db); err != nil {
			log.Fatal(err)
		}
	*/
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

	handle := func(path string, fn htmlHandleFunc) *mux.Route {
		return r.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {

			reportErr := func(err error) {
				err = fmt.Errorf("Error at %s %s => %v", r.Method, r.URL.Path, err)
				log.Print(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			bytes, err := fn(w, r)
			if err != nil {
				reportErr(err)
				return
			}

			if _, err := w.Write(bytes); err != nil {
				reportErr(err)
			}

		})
	}

	render := newRenderFunc(tmpl)

	/*
		Home / Index Handler
	*/
	handle("/", handleGetHomePage(render))

	/*
		Promotions Handlers
	*/
	handle("/promotions/", handleGetPromotionsPage(db, render)).Methods(http.MethodGet)
	handle("/team/", handleGetTeamPage(getPersonas, render)).Methods(http.MethodGet)
	handle("/promotions/{websiteName}", handleGetPromotionsPage(db, render)).Methods(http.MethodGet)
	handle("/{websiteName}/promotions/", handleGetPromotionsPage(db, render)).Methods(http.MethodGet)
	handle("/feed/", handleGetFeed(db, render)).Methods(http.MethodGet)
	handle("/feed/{websiteName}/", handleGetFeed(db, render)).Methods(http.MethodGet)
	handle("/websites/", handleGetWebsites(getWebsites, render)).Methods(http.MethodGet)
	handle("/websites/{website_id}/", handleGetWebsiteByID(getWebsiteByID, render)).Methods(http.MethodGet)
	handle("/subscribe/", handlePostSubscribe(mode, port, productionDomain, db, render)).Methods(http.MethodPost)
	handle("/subscribe/", handleGetSubscribePage(render)).Methods(http.MethodGet)
	handle("/subscribe/verify", handleGetVerifySubscription(db, render)).Methods(http.MethodGet)

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

				q := `SELECT count(*) FROM post_hashtags WHERE post_id = ? AND hashtag_id = ?`

				var count int
				if err := db.QueryRow(q, p.ID, hashtagID).Scan(&count); err != nil {
					return fmt.Errorf("could not count relationships between %d & %d. %v", p.ID, hashtagID, err)
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
			err := db.QueryRow(`SELECT count(id) FROM posts WHERE src_url = ?`, u).Scan(&bannerCount)
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

			author := getRandomPersona()

			// I picked 8 randomly for author id
			authorID := author.ID

			_, err = db.Exec(
				"INSERT INTO posts(website_id, src_url, author_id, description, timestamp) VALUES (? , ? , ?, ?, ?)",
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
func handleGetHomePage(render renderFunc) htmlHandleFunc {
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
		return render("home", map[string]any{"MenuItems": menuItems, "Request": r})
	}
}

func handleGetTeamPage(getPersonas func(a, b int) []Persona, render renderFunc) htmlHandleFunc {
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
		return render("teampage", map[string]any{"MenuItems": menuItems, "Request": r, "Team": getPersonas(0, 0)})
	}
}

func handleGetPromotionsPage(db *sql.DB, render renderFunc) htmlHandleFunc {
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
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
			return nil, fmt.Errorf("failed to get posts %w", err)
		}

		return render("promotionspage", map[string]any{"Promotions": promos, "MenuItems": menuItems, "Request": r})
	}
}

func handleGetFeed(db *sql.DB, render renderFunc) htmlHandleFunc {
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {

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
				params.WebsiteID = 0
			}

		}

		if hashtagQuery != "" {
			params.Hashtag = hashtagQuery
		}

		var postIDs []int
		if params.Hashtag != "" {
			hashtagID, err := getHashtagIDByPhrase(db, params.Hashtag)
			if err != nil {
				return nil, fmt.Errorf("could not get hashtag id in get by phrase. %w", err)
			}

			postIdRows, err := db.Query("SELECT post_id FROM post_hashtags WHERE hashtag_id = ?", hashtagID)
			if err != nil {
				return nil, fmt.Errorf("error getting post_ids from post_hashtags db. %w", err)
			}
			defer postIdRows.Close()

			ids := []int{}
			for postIdRows.Next() {
				var id int
				err := postIdRows.Scan(&id)
				if err != nil {
					return nil, fmt.Errorf("error scanning post_id in getposts. %w", err)
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
			return nil, fmt.Errorf("cant get posts: %w", err)
		}

		personas := getPersonas(0, 0)

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
				return nil, fmt.Errorf("could not get website by id %d. %v", promos[i].WebsiteID, err)
			}
			e.Content.Summary = fmt.Sprintf("posted an update about %s", website.WebsiteName)
			e.Content.ExtraImages = nil
			events = append(events, e)
		}

		websites := getWebsites(10, 0)

		limit := 5
		q := `SELECT hashtag_id, count(post_id) FROM post_hashtags GROUP BY hashtag_id ORDER BY count(post_id) DESC LIMIT ?`
		rows, err := db.Query(q, limit)
		if err != nil {
			return nil, fmt.Errorf("could not count hashtag mentions in db: %w", err)
		}
		defer rows.Close()

		top := []GetTopByPostCountResponse{}
		for rows.Next() {
			var row GetTopByPostCountResponse
			if err := rows.Scan(&row.HashtagID, &row.PostCount); err != nil {
				return nil, err
			}
			top = append(top, row)
		} // should expect an array like {hashtag, postcount}

		var trendingHashtags []*Trending
		for _, row := range top {
			hashtag, err := getHashtagByID(db, row.HashtagID)
			if err != nil {
				return nil, fmt.Errorf("could not get hashtag by id at GetTrending in hashtagsvc. %w", err)
			}
			trendingHashtags = append(trendingHashtags, &Trending{Category: "Topic", Phrase: hashtag.Phrase, PostCount: row.PostCount})
		}

		data := map[string]any{"Events": events, "Websites": websites, "Trending": trendingHashtags, "Request": r, "MenuItems": menuItems}
		return render("feedpage", data)
	}

}

func handleGetWebsites(getWebsites func(limit, offset int) []Website, render renderFunc) htmlHandleFunc {
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
		limit, offset, page := paginator(r, 50)
		websites := getWebsites(limit, offset)

		count := len(getWebsites(0, 0))

		maxPages := int(math.Ceil(float64(count) / float64(limit)))

		pagination := Pagination{page, maxPages}

		return render("websites", map[string]any{"MenuItems": menuItems, "Request": r, "Websites": websites, "Pagination": pagination})
	}
}

func handleGetWebsiteByID(getWebsiteByID func(website_id int) (Website, error), render renderFunc) htmlHandleFunc {
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
		vars := mux.Vars(r)

		website_id, err := strconv.Atoi(vars["website_id"])
		if err != nil {
			website_id = 1
		}

		website, err := getWebsiteByID(website_id)
		if err != nil {
			return nil, fmt.Errorf("error %v", err)
		}

		return render("website", map[string]any{"MenuItems": menuItems, "Request": r, "Website": website})
	}
}

func handleGetBrands(db *sql.DB, render renderFunc) htmlHandleFunc {
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
		limit, offset, page := paginator(r, 50)

		brands, err := getBrands(db, limit, offset)
		if err != nil {
			return nil, err
		}

		brandCount, err := countAllBrands(db)
		if err != nil {
			return nil, err
		}

		maxPages := int(math.Ceil(float64(brandCount) / float64(limit)))

		pagination := Pagination{page, maxPages}

		data := map[string]any{
			"MenuItems":  menuItems,
			"Request":    r,
			"Brands":     brands,
			"Pagination": pagination,
		}

		return render("brands", data)
	}
}

func handlePostSubscribe(mode Mode, port, productionDomain string, db *sql.DB, render renderFunc) htmlHandleFunc {
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
		if err := r.ParseForm(); err != nil {
			return nil, fmt.Errorf("could not parse form: %w", err)
		}

		email := r.FormValue("email")
		consent := r.FormValue("consent")

		if consent == "on" {

			q := `INSERT INTO subscribers(email, consent) VALUES (?, ?)`
			_, err := db.Exec(q, email, 1)
			if err != nil {
				return nil, fmt.Errorf("could not insert email into subscibers table => %w", err)
			}

			// Calculate the required byte size based on the length of the token
			byteSize := 20 / 2 // Each byte is represented by 2 characters in hexadecimal encoding

			// Create a byte slice to store the random bytes
			randomBytes := make([]byte, byteSize)

			// Read random bytes from the crypto/rand package
			_, err = rand.Read(randomBytes)
			if err != nil {
				return nil, err
			}

			// Encode the random bytes into a hexadecimal string
			token := fmt.Sprintf("%x", randomBytes)

			// TODO move gernate token to service level
			// TODO do one insert with both email and verification token

			setVerificationTokenQuery := `UPDATE subscribers SET verification_token = ?, is_verified = 0 WHERE email = ?`
			_, err = db.Exec(setVerificationTokenQuery, token, email)
			if err != nil {
				return nil, fmt.Errorf("could not add verification token to user by email => %w", err)
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

			return render("subscriptionsuccess", nil)
		}

		return render("subscriptionform", nil)
	}
}

func handleGetSubscribePage(render renderFunc) htmlHandleFunc {
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
		return render("subscribepage", map[string]any{"MenuItems": menuItems, "Request": r})
	}
}

func handleGetVerifySubscription(db *sql.DB, render renderFunc) htmlHandleFunc {
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
		vars := r.URL.Query()
		token := vars.Get("token")

		if token == "" {
			// ("Warning: subscription verification attempted with no token")
			return handleGetHomePage(render)(w, r)
		}

		q := `UPDATE subscribers SET is_verified = 1 WHERE verification_token = ?`
		_, err := db.Exec(q, token)
		if err != nil {
			return nil, fmt.Errorf("could not verify subscription via verification token => %w", err)
		}

		// subscription confirmed

		return render("subscriptionverification", map[string]any{"MenuItems": menuItems, "Request": r})
	}
}

/* handlers end */

/* template functions start*/

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

func newRenderFunc(tmpl *template.Template) renderFunc {
	return func(name string, data any) ([]byte, error) {
		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
			return nil, fmt.Errorf("render error: %w", err)
		}
		return buf.Bytes(), nil
	}
}

func paginator(r *http.Request, defaultLimit int) (int, int, int) {
	q := r.URL.Query()

	limit := defaultLimit
	offset := 0
	page := 1

	_limit := q.Get("limit")
	if _limit != "" {
		l, err := strconv.Atoi(_limit)
		if err != nil {
			log.Printf("error parsing limit from %s, %v", r.URL.String(), err)
		} else {
			limit = l
		}
	}

	_offset := q.Get("offset")
	if _offset != "" {
		o, err := strconv.Atoi(_offset)
		if err != nil {
			log.Printf("error parsing offset from %s, %v", r.URL.String(), err)
		} else {
			offset = o
		}
	}

	_page := q.Get("page")
	if _page != "" {
		p, err := strconv.Atoi(_page)
		if err != nil {
			log.Printf("error parsing page from %s, %v", r.URL.String(), err)
		} else {
			page = p
			offset = (limit * (p - 1))
		}
	}

	return limit, offset, page
}

/* db funcs */

var errDBNil = errors.New("db is nil")

/* Brand DB Funcs */
func getBrandByID(db *sql.DB, id int) (*Brand, error) {
	if db == nil {
		return nil, errDBNil
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
		return nil, errDBNil
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
		return -1, errDBNil
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
		return -1, errDBNil
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
		return -1, errDBNil
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
		return nil, errDBNil
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
		return nil, errDBNil
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
		return errDBNil
	}

	if brand == nil {
		return errors.New("brand passed to updateBrand is nil")
	}
	if brand.ID < 1 {
		return errors.New("need to supply a brand id to update")
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
		return errDBNil
	}

	if brandID < 1 {
		return errors.New("need to supply a valid brand id to update")
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
		return -1, errDBNil
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
		return -1, errDBNil
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
		return -1, errDBNil
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
		return nil, errDBNil
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
func getPersonas(limit, offset int) []Persona {
	personas := []Persona{
		{1, "Fiona", "", "https://replicate.delivery/pbxt/hHQ0aXYNnab3IRT39NsqDrgfJ4c3OfPeMz0Qe4MXPvj5NidMB/output_1.png"},
		{2, "Emily", "", "https://replicate.delivery/pbxt/OrPxbF0Z9V78EhxVTXONM7msHZSKJgEIQGG9eXz3Vjw8YsjJA/output_1.png"},
		{3, "Sarah", "", "https://replicate.delivery/pbxt/SNyHoT1Jla70GpYLNPgkMe2x1RJPyUaU4meT4fQHsiJO3wOmA/output_1.png"},
		{4, "Aoife", "", "https://replicate.delivery/pbxt/7HKCIPYGhqqPEZ10YnjeuYeLRkkvOokRVIpHFC47lH7vSYHTA/output_1.png"},
		{5, "Liz", "", "https://replicate.delivery/pbxt/asJWtatC9zKNL5qnZo5NPUs3bJfu8G4z6feGzBE9E49xgyOmA/output_1.png"},
		{6, "Mia", "", "https://replicate.delivery/pbxt/40x0CWfDTUSaLqoZfyG5VX4ewSWp6XWRefQRUYt72AavKK7YC/output_1.png"},
		{7, "Sinead", "", "https://replicate.delivery/pbxt/tAmmkAgi1exlWyWjxNgePDRBu9cnietW5y2h8dXkelKTHldMB/output_1.png"},
	}

	lenPersonas := len(personas)

	if limit == 0 || limit > lenPersonas {
		limit = lenPersonas
	}

	if offset > len(personas) {
		offset = 0
	}

	res := []Persona{}
	for i := offset; i < limit; i++ {
		res = append(res, personas[i])
	}

	return res
}

// get one persona
func getPersonaByID(id int) (Persona, error) {
	for _, p := range getPersonas(0, 0) {
		if p.ID == id {
			return p, nil
		}
	}
	return Persona{}, fmt.Errorf("no persona with id %d", id)
}

// get one random persona
func getRandomPersona() Persona {
	personas := getPersonas(0, 0)
	randomInt := rand.Intn(len(personas))
	return personas[randomInt]
}

/* posthashtag db funcs*/

func insertPostHashtagRelationship(db *sql.DB, postID, hashtagID int) error {

	if db == nil {
		return errDBNil
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
		return errDBNil
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

// TODO update these functions to addres website_id column
func updatePost(db *sql.DB, p *Post) error {

	if db == nil {
		return errDBNil
	}

	if p == nil {
		return errors.New("post past to update post is nil")
	}

	q := `UPDATE 
		posts 
	SET 
		author_id = ?, 
		src_url = ?, 
		website_id = ?, 
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
		return nil, errDBNil
	}

	promos := []*Post{}

	q := `SELECT id, website_id, src_url, author_id, description, timestamp FROM posts`

	args := []any{}
	if params.WebsiteID != 0 || len(params.IDs) > 0 {
		subQ := []string{}
		if params.WebsiteID != 0 {
			subQ = append(subQ, "website_id = ?")
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

/* website funcs*/

// Retrieve a website by its ID from the Websites table
func getWebsiteByID(website_id int) (Website, error) {
	for _, website := range getWebsites(0, 0) {
		if website.WebsiteID == website_id {
			return website, nil
		}
	}
	return Website{}, fmt.Errorf("no website with id %d", website_id)
}

// Retrieve a website by its ID from the Websites table
func getWebsiteByName(websiteName string) (Website, error) {
	for _, website := range getWebsites(0, 0) {
		if strings.EqualFold(website.WebsiteName, websiteName) {
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

	lenWebsites := len(websites)

	if limit == 0 || limit > lenWebsites {
		limit = lenWebsites
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
		return []string{}, fmt.Errorf("could not find banner extraction rules for website %s", website.WebsiteName)
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
		return []string{}, fmt.Errorf("could not find banner extraction rules for website %s", website.WebsiteName)
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
