package main

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	mrand "math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sashabaranov/go-openai"
	"golang.org/x/crypto/bcrypt"
)

/* models begin */
type Mode string

const (
	Dev  Mode = "dev"
	Prod Mode = "prod"
)

type BannerData struct {
	Src            string
	SupportingText string
	Href           string
}

// Website struct matching the Websites table
type Website struct {
	WebsiteID   int     `json:"website_id"`
	WebsiteName string  `json:"website_name"`
	URL         string  `json:"url"`
	Country     string  `json:"country"`
	Score       float64 `json:"score"`
	Screenshot  string  `json:"screenshot"`
}

var websites = []Website{
	{1, "BeautyFeatures", "https://www.beautyfeatures.ie", "IE", 8, "www.beautyfeatures.ie_.png"},
	{2, "LookFantastic", "https://lookfantastic.ie", "IE", 8, "www.lookfantastic.ie_.png"},
	{3, "Millies", "https://millies.ie", "IE", 9, "millies.ie_.png"},
	{4, "McCauley Pharmacy", "https://www.mccauley.ie/", "IE", 3, "www.mccauley.ie_.png"},
}

type Category struct {
	Name string
}

var categories = []Category{
	{"Haircare"},
	{"Skincare"},
	{"Makeup"},
	{"Fragrance"},
	{"Body Care"},
	{"Nail Care"},
	{"Men's Grooming"},
	{"Beauty Tools"},
	{"Bath & Shower"},
	{"Sun Care"},
	{"Oral Care"},
	{"Wellness"},
}

type Brand struct {
	ID    int
	Name  string
	Path  string
	Score float64
}

type getAllBrandsParams struct {
	Limit, Offset int
}
type Post struct {
	WebsiteID   int
	ID          int
	Description string
	SrcURL      string
	Link        string
	Timestamp   time.Time
	AuthorID    int // supposed to correspond with a persona id
	Score       float64
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
	ID      int
	Profile Profile
	Content Content
	Meta    EventMeta
}

var db *sql.DB
var err error
var mode Mode
var port string
var productionDomain string
var store *sessions.CookieStore

func processing(ready <-chan time.Time) {
	fmt.Println("start processing")
	if err := extractOffersFromBanners(); err != nil {
		reportErr(err)
	}
	if err := scorePosts(); err != nil {
		reportErr(err)
	}
	fmt.Println("finished processing")
	<-ready
	processing(ready)
}

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
		ticker := time.NewTicker(time.Hour)
		go processing(ticker.C)
	} else {
		log.Println("skipping banner extractions and hashtag process")
	}

	if err := server(); err != nil {
		log.Fatal(err)
	}

}

func scorePosts() error {
	posts, err := getPosts(db, getPostParams{})
	if err != nil {
		return err
	}

	for i := range posts {
		w, err := getWebsiteByID(posts[i].WebsiteID)
		if err != nil {
			return err
		}

		if posts[i].Score != float64(w.Score) {
			posts[i].Score = float64(w.Score)
			_, err := db.Exec("UPDATE posts SET score = ? WHERE id = ?", posts[i].Score, posts[i].ID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

var _tmpl *template.Template

func tmpl() *template.Template {
	if mode == Prod && _tmpl != nil {
		return _tmpl
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
	_tmpl = template.Must(template.New("web").Funcs(funcMap).ParseGlob("templates/**/*.tmpl"))
	return _tmpl
}

func reportErr(err error) {

	log.Print(err)
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

	globalMiddleware := []func(next handleFunc) handleFunc{
		func(next handleFunc) handleFunc {
			return (func(w http.ResponseWriter, r *http.Request) error {
				log.Printf("%s => %s", r.Method, r.URL.Path)
				return next(w, r)
			})
		},
	}

	handle := func(path string, fn handleFunc) {
		for i := range globalMiddleware {
			fn = globalMiddleware[i](fn)
		}
		r.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if err := fn(w, r); err != nil {
				err = fmt.Errorf("error at %s %s => %v", r.Method, r.URL.Path, err)
				reportErr(err)
				return
			}
		})
	}

	/*
		Promotions Handlers
	*/
	handle("POST /subscribe", handlePostSubscribe)
	handle("GET /subscribe", handleGetSubscribePage)
	handle("GET /subscribe/verify", handleGetVerifySubscription)

	/*
		Home / Index Handler
	*/
	handle("/", handleGetFeed)
	handle("GET /store/{websiteName}", handleGetFeed)

	admin := http.NewServeMux()

	adminMiddleware := []func(next handleFunc) handleFunc{
		func(next handleFunc) handleFunc {
			return func(w http.ResponseWriter, r *http.Request) error {
				// do something admin

				session, err := store.Get(r, "admin_session")
				if err != nil {
					return err
				}

				email, found := session.Values["admin_email"]
				if !found || email != os.Getenv("ADMIN_EMAIL") {
					w.WriteHeader(http.StatusUnauthorized)
					return nil
				}

				return next(w, r.WithContext(context.WithValue(r.Context(), "admin_email", email)))
			}
		},
	}

	adminHandle := func(path string, fn handleFunc) {
		for i := range globalMiddleware {
			fn = globalMiddleware[i](fn)
		}

		for i := range adminMiddleware {
			fn = adminMiddleware[i](fn)
		}

		admin.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if err := fn(w, r); err != nil {
				err = fmt.Errorf("error at %s %s => %v", r.Method, r.URL.Path, err)
				reportErr(err)
				return
			}

		})
	}

	handle("GET /admin/signin", adminHandleGetSignIn)
	adminHandle("GET /", adminHandleGetDashboard)
	adminHandle("POST /signin", adminHandlePostSignIn)

	r.Handle("/admin/", http.StripPrefix("/admin", admin))

	log.Println("Server listening on http://localhost:" + port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		return fmt.Errorf("failure to launch. %w", err)
	}
	return nil
}

func processHashtags() error {

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

func extractOffersFromBanners() error {

	websites := getWebsites(0, 0)

	for _, website := range websites {
		banners, err := extractWebsiteBannerURLs(website)
		if err != nil {
			continue
		}

		uniqueBanners := []BannerData{}
		for _, banner := range banners {

			var bannerCount int
			err := db.QueryRow(`SELECT count(id) FROM posts WHERE src_url = ?`, banner.Src).Scan(&bannerCount)
			if err != nil {
				return fmt.Errorf("error checking existance of banner %v", err)
			}

			bannerExists := bannerCount > 0

			if bannerExists {
				continue
			}

			uniqueBanners = append(uniqueBanners, banner)
		}

		for _, banner := range uniqueBanners {

			if banner.Src == "" {
				continue
			}

			description, err := generateOfferDescription(website.WebsiteName, banner)
			if err != nil {
				return fmt.Errorf(`error getting offer description from chatgpt: %v`, err)
			}

			author := getRandomPersona()

			// I picked 8 randomly for author id
			authorID := author.ID

			_, err = db.Exec(
				"INSERT INTO posts(website_id, src_url, author_id, description, timestamp) VALUES (? , ? , ?, ?, ?)",
				website.WebsiteID, banner.Src, authorID, description, time.Now())
			if err != nil {
				return fmt.Errorf(`error saving banner promotion: %w`, err)
			}
		}

	}

	if err := processHashtags(); err != nil {
		return err
	}
	return nil
}

func handleGetFeed(w http.ResponseWriter, r *http.Request) error {
	websiteName := r.PathValue("websiteName")
	hashtagQuery := r.URL.Query().Get("hashtag")

	var website Website
	if websiteName != "" {
		w, err := getWebsiteByName(websiteName)
		if err == nil {
			website = w
		}
	}

	var postIDs []int
	if hashtagQuery != "" {
		hashtagID, err := getHashtagIDByPhrase(db, hashtagQuery)
		if err != nil {
			return fmt.Errorf("could not get hashtag id in get by phrase. %w", err)
		}

		postIdRows, err := db.Query("SELECT post_id FROM post_hashtags WHERE hashtag_id = ?", hashtagID)
		if err != nil {
			return fmt.Errorf("error getting post_ids from post_hashtags db. %w", err)
		}
		defer postIdRows.Close()

		for postIdRows.Next() {
			var id int
			err := postIdRows.Scan(&id)
			if err != nil {
				return fmt.Errorf("error scanning post_id in getposts. %w", err)
			}
			postIDs = append(postIDs, id)
		}
	}

	var getPostParams getPostParams
	getPostParams.IDs = postIDs
	getPostParams.SortBy = "score"
	getPostParams.WebsiteID = website.WebsiteID
	getPostParams.Limit = 6

	rows, err := db.Query(`WITH orderedPosts AS (
	SELECT 
        p.id,
        p.description,
		p.author_id,
		p.Score,
        p.src_url,
        p.timestamp,
        p.website_id
    FROM 
        posts p
    ORDER BY 
        p.timestamp DESC
	LIMIT 6
	) SELECT 
        op.id,
        op.description,
		op.author_id,
		op.Score,
        op.src_url,
        op.timestamp,
        op.website_id
    FROM 
        orderedPosts op
	ORDER BY
		score DESC
	LIMIT 6
	`)
	if err != nil {
		return err
	}
	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.Description, &post.AuthorID, &post.Score, &post.SrcURL, &post.Timestamp, &post.WebsiteID); err != nil {
			return err
		}
		posts = append(posts, post)
	}

	personas := getPersonas(0, 0)

	events := []Event{}
	for i := 0; i < len(posts); i++ {
		e := Event{}
		for _, persona := range personas {
			if persona.ID == posts[i].AuthorID {
				e.Profile.Username = persona.Name
				e.Profile.Photo = persona.ProfilePhoto
			}
		}
		//	e.Profile.Username

		// Step 1: Calculate Time Difference
		timeDiff := time.Since(posts[i].Timestamp)

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
		e.Meta.Src = &posts[i].SrcURL

		if posts[i].Link != "" {
			e.Meta.CTALink = &posts[i].Link
		}

		pattern := regexp.MustCompile(`#(\w+)`)

		extraText := posts[i].Description

		matches := pattern.FindAllStringSubmatch(extraText, -1)

		for _, match := range matches {
			phrase := strings.ToLower(match[1])
			extraText = strings.Replace(extraText, match[0], fmt.Sprintf("<a class='text-blue-500' href='?hashtag=%s'>%s</a>", phrase, match[0]), 1)
		}

		extraTextHTML := template.HTML(extraText)

		e.Content.ExtraText = &extraTextHTML
		website, err := getWebsiteByID(posts[i].WebsiteID)
		if err != nil {
			return fmt.Errorf("could not get website by id %d. %v", posts[i].WebsiteID, err)
		}
		e.Content.Summary = fmt.Sprintf("posted an update about %s", website.WebsiteName)
		// e.Content.ExtraImages = nil
		e.Content.ExtraImages = &[]ExtraImage{{posts[i].SrcURL, ""}}
		events = append(events, e)
	}

	limit := 5
	q := `SELECT hashtag_id, count(post_id) FROM post_hashtags GROUP BY hashtag_id ORDER BY count(post_id) DESC LIMIT ?`
	rows, err = db.Query(q, limit)
	if err != nil {
		return fmt.Errorf("could not count hashtag mentions in db: %w", err)
	}
	defer rows.Close()

	top := make([]GetTopByPostCountResponse, 0, limit)
	for rows.Next() {
		var row GetTopByPostCountResponse
		if err := rows.Scan(&row.HashtagID, &row.PostCount); err != nil {
			return err
		}
		top = append(top, row)
	} // should expect an array like {hashtag, postcount}

	var trendingHashtags []*Trending
	for _, row := range top {
		hashtag, err := getHashtagByID(db, row.HashtagID)
		if err != nil {
			return fmt.Errorf("could not get hashtag by id at GetTrending in hashtagsvc. %w", err)
		}
		trendingHashtags = append(trendingHashtags, &Trending{Category: "Topic", Phrase: hashtag.Phrase, PostCount: row.PostCount})
	}

	subscribed := false
	c, err := r.Cookie("subscription_status")
	if err != nil && err != http.ErrNoCookie {
		return err
	}

	if c != nil {
		subscribed = c.Value == "subscribed"
	}

	data := map[string]any{
		"AlreadySubscribed": subscribed,
		"Events":            events,
		"Websites":          getWebsites(0, 0),
		"Trending":          trendingHashtags,
		"Categories":        getCategories(0, 0),
	}

	return renderPage(w, "feedpage", data)
}

func adminHandleGetDashboard(w http.ResponseWriter, r *http.Request) error {

	rows, err := db.Query("SELECT id, description, author_id, score, src_url, timestamp, website_id FROM posts ORDER BY timestamp DESC")
	if err != nil {
		return err
	}

	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.Description, &post.AuthorID, &post.Score, &post.SrcURL, &post.Timestamp, &post.WebsiteID); err != nil {
			return err
		}
		posts = append(posts, post)
	}

	personas := getPersonas(0, 0)

	events := make([]Event, 0, len(posts))
	for i := 0; i < len(posts); i++ {
		e := Event{}

		e.ID = posts[i].ID

		for _, persona := range personas {
			if persona.ID == posts[i].AuthorID {
				e.Profile.Username = persona.Name
				e.Profile.Photo = persona.ProfilePhoto
			}
		}
		//	e.Profile.Username

		// Step 1: Calculate Time Difference
		timeDiff := time.Since(posts[i].Timestamp)

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
		e.Meta.Src = &posts[i].SrcURL

		if posts[i].Link != "" {
			e.Meta.CTALink = &posts[i].Link
		}

		pattern := regexp.MustCompile(`#(\w+)`)

		extraText := posts[i].Description

		matches := pattern.FindAllStringSubmatch(extraText, -1)

		for _, match := range matches {
			phrase := strings.ToLower(match[1])
			extraText = strings.Replace(extraText, match[0], fmt.Sprintf("<a class='text-blue-500' href='?hashtag=%s'>%s</a>", phrase, match[0]), 1)
		}

		extraTextHTML := template.HTML(extraText)

		e.Content.ExtraText = &extraTextHTML
		website, err := getWebsiteByID(posts[i].WebsiteID)
		if err != nil {
			return fmt.Errorf("could not get website by id %d. %v", posts[i].WebsiteID, err)
		}
		e.Content.Summary = fmt.Sprintf("posted an update about %s", website.WebsiteName)
		// e.Content.ExtraImages = nil
		e.Content.ExtraImages = &[]ExtraImage{{posts[i].SrcURL, ""}}
		events = append(events, e)
	}

	limit := 5
	q := `SELECT hashtag_id, count(post_id) FROM post_hashtags GROUP BY hashtag_id ORDER BY count(post_id) DESC LIMIT ?`
	rows, err = db.Query(q, limit)
	if err != nil {
		return fmt.Errorf("could not count hashtag mentions in db: %w", err)
	}
	defer rows.Close()

	top := make([]GetTopByPostCountResponse, 0, limit)
	for rows.Next() {
		var row GetTopByPostCountResponse
		if err := rows.Scan(&row.HashtagID, &row.PostCount); err != nil {
			return err
		}
		top = append(top, row)
	} // should expect an array like {hashtag, postcount}

	var trendingHashtags []Trending
	for _, row := range top {
		hashtag, err := getHashtagByID(db, row.HashtagID)
		if err != nil {
			return fmt.Errorf("could not get hashtag by id at GetTrending in hashtagsvc. %w", err)
		}
		trendingHashtags = append(trendingHashtags, Trending{Category: "Topic", Phrase: hashtag.Phrase, PostCount: row.PostCount})
	}

	data := map[string]any{
		"Events":     events,
		"Websites":   getWebsites(0, 0),
		"Trending":   trendingHashtags,
		"Categories": getCategories(0, 0),
	}

	return renderPage(w, "admindashboard", data)
}

func adminHandleGetSignIn(w http.ResponseWriter, r *http.Request) error {
	return renderPage(w, "adminsignin", nil)
}

func adminHandleGetSignOut(w http.ResponseWriter, r *http.Request) error {
	session, err := store.Get(r, "admin_session")
	if err != nil {
		return err
	}

	if session.Values["admin_email"] == nil || session.Values["admin_email"] != os.Getenv("ADMIN_EMAIL") {
		return errors.New("signout called but not logged in")
	}

	session.Values["admin_email"] = ""
	if err := session.Save(r, w); err != nil {
		return err
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil

}

func adminHandlePostSignIn(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	if os.Getenv("ADMIN_EMAIL") == "" || os.Getenv("HASHED_PASSWORD") == "" {
		return fmt.Errorf("Either admin_email or hashed_password is not set in env")
	}

	email, password := r.Form.Get("email"), r.Form.Get("password")

	if email != os.Getenv("ADMIN_EMAIL") {
		log.Println("incorrect admin email supplied")
		return renderPage(w, "adminsignin", nil)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(os.Getenv("HASHED_PASSWORD")), []byte(password)); err != nil {
		log.Printf("incorrect password supplied %v", err)
		return renderPage(w, "adminsignin", nil)
	}

	session, err := store.Get(r, "admin_session")
	if err != nil {
		return err
	}

	session.Values["admin_email"] = email

	if err := store.Save(r, w, session); err != nil {
		return err
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
	return nil
}

func handlePostSubscribe(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("could not parse form: %w", err)
	}

	email := r.FormValue("email")
	consent := r.FormValue("consent")

	if consent != "on" {
		// TODO maybe create an error state
		return render(w, "subscriptionform", map[string]any{"ConsentErr": "Please consent so we can add you to our mailing list. Thanks!"})
	}

	q := `INSERT INTO subscribers(email, consent) VALUES (?, 1)`
	if _, err := db.Exec(q, email); err != nil {
		return fmt.Errorf("could not insert email into subscibers table => %w", err)
	}

	// Calculate the required byte size based on the length of the token
	byteSize := 20 / 2 // Each byte is represented by 2 characters in hexadecimal encoding

	// Create a byte slice to store the random bytes
	randomBytes := make([]byte, byteSize)

	// Read random bytes from the crypto/rand package
	if _, err := crand.Read(randomBytes); err != nil {
		return err
	}

	// Encode the random bytes into a hexadecimal string
	token := fmt.Sprintf("%x", randomBytes)

	// TODO move gernate token to service level
	// TODO do one insert with both email and verification token

	setVerificationTokenQuery := `UPDATE subscribers SET verification_token = ?, is_verified = 0 WHERE email = ?`
	if _, err := db.Exec(setVerificationTokenQuery, token, email); err != nil {
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

	http.SetCookie(w, &http.Cookie{
		Name:    "subscription_status",
		Value:   "subscribed",
		Expires: time.Now().Add(30 * 24 * time.Hour),
	})

	return render(w, "subscriptionsuccess", nil)

}

func handleGetSubscribePage(w http.ResponseWriter, r *http.Request) error {
	return renderPage(w, "subscribepage", map[string]any{})
}

func handleGetVerifySubscription(w http.ResponseWriter, r *http.Request) error {
	vars := r.URL.Query()
	token := vars.Get("token")

	if token == "" {
		// ("Warning: subscription verification attempted with no token")
		return handleGetFeed(w, r)
	}

	q := `UPDATE subscribers SET is_verified = 1 WHERE verification_token = ?`
	_, err := db.Exec(q, token)
	if err != nil {
		return fmt.Errorf("could not verify subscription via verification token => %w", err)
	}

	// subscription confirmed

	return renderPage(w, "subscriptionverification", map[string]any{})
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

// s must come from trusted source
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

func renderPage(w io.Writer, name string, data map[string]any) error {
	templateData := map[string]any{
		"Env": mode,
	}

	for k, v := range data {
		templateData[k] = v
	}

	return render(w, name, templateData)
}

func render(w io.Writer, name string, data any) error {
	var buf bytes.Buffer
	if err := tmpl().ExecuteTemplate(&buf, name, data); err != nil {
		return fmt.Errorf("render error: %w", err)
	}
	_, err := buf.WriteTo(w)
	return err
}

/* db funcs */

var errDBNil = errors.New("db is nil")

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
		{1, "Michaela Gormley", "", "https://replicate.delivery/yhqm/uZSufO1ULetrN00FXde5s7mjJJB5i0yh3Drfms5HeBb0ReS1E/out-0.webp"},
		{2, "Caroline Mullen ", "", "https://replicate.delivery/yhqm/kqyPgcRDPFZgAxqY5aQqgLtscg6x2zCal815fwTDDQZ15lqJA/out-0.webp"},
		{3, "Susan Fagan", "", "https://replicate.delivery/yhqm/sxYBygzKehzXHiJk2zdLF4sK3LeZ0jx6ss9qgmbJKcreoXqmA/out-0.webp"},
		{4, "Clare O'Shea", "", "https://replicate.delivery/yhqm/JqI1VW2t0f3lfkO4T7WfVciDf9BdsuzYcWljhcrFfGjfR9S1E/out-0.webp"},
		{5, "Aisling O'Reilly", "", "https://replicate.delivery/yhqm/nXCbkLusEIItOJf0UY6JCiXs1oaxrpKLXwV8dkhKMe791LVTA/out-0.webp"},
		{6, "Stacey Dowling", "", "https://replicate.delivery/yhqm/gMBf17FLmVQnGS9te4MWQJWlE6PdhjZd5Fhl2ycIZj1ftXqmA/out-0.webp"},
		{7, "Danielle Duffy", "", "https://replicate.delivery/yhqm/rvXUljqA3QqxCtGa64qnmJyn5jh571lvA8Dixinh6LK39S1E/out-0.webp"},
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

// get one random persona
func getRandomPersona() Persona {
	personas := getPersonas(0, 0)
	randomInt := mrand.Intn(len(personas))
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

type getPostParams struct {
	WebsiteID     int
	IDs           []int
	Limit         int
	SortBy        string
	SortAscending bool
}

func getPosts(db *sql.DB, params getPostParams) ([]Post, error) {
	if db == nil {
		return nil, errDBNil
	}

	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT id, website_id, src_url, author_id, description, timestamp FROM posts")

	args := make([]interface{}, 0)
	if params.WebsiteID != 0 || len(params.IDs) > 0 {
		queryBuilder.WriteString(" WHERE ")
		conditions := make([]string, 0)

		if params.WebsiteID != 0 {
			conditions = append(conditions, "website_id = ?")
			args = append(args, params.WebsiteID)
		}

		if len(params.IDs) > 0 {
			placeholders := make([]string, len(params.IDs))
			for i := range placeholders {
				placeholders[i] = "?"
			}
			conditions = append(conditions, fmt.Sprintf("id IN (%s)", strings.Join(placeholders, ",")))
			for _, id := range params.IDs {
				args = append(args, id)
			}
		}

		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}

	if params.SortBy != "" {
		direction := "DESC"
		if params.SortAscending {
			direction = "ASC"
		}
		sortString := fmt.Sprintf(" ORDER BY %s %s", params.SortBy, direction)

		if params.SortBy == "timestamp" {
			queryBuilder.WriteString(sortString)
		}

		if params.SortBy == "score" {
			queryBuilder.WriteString(sortString)
		}
	}

	if params.Limit > 0 {
		queryBuilder.WriteString(" LIMIT ?")
		args = append(args, params.Limit)
	}

	rows, err := db.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]Post, 0, params.Limit)
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.WebsiteID, &post.SrcURL, &post.AuthorID, &post.Description, &post.Timestamp); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
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
    preferences TEXT
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
	lenWebsites := len(websites)

	if limit == 0 || limit > lenWebsites {
		limit = lenWebsites
	}

	if offset >= lenWebsites {
		offset = 0
	}

	toReturn := make([]Website, 0, limit)
	for i := offset; i < limit; i++ {
		toReturn = append(toReturn, websites[i])
	}
	return toReturn
}

func getCategories(limit, offset int) []Category {
	lenCategories := len(categories)

	if limit == 0 || limit > lenCategories {
		limit = lenCategories
	}

	if offset >= lenCategories {
		offset = 0
	}

	toReturn := make([]Category, 0, limit)
	for i := offset; i < limit; i++ {
		toReturn = append(toReturn, categories[i])
	}
	return toReturn
}

// to replace existing method. supports 'supporting text'
// for websites like lookfantastic and cult beauty that have carousels with text not embedded in the image
// can be passed to the llm for additional context
func extractWebsiteBannerURLs(website Website) ([]BannerData, error) {
	res, err := http.Get(website.URL)
	if err != nil {
		return nil, fmt.Errorf("error sending get request to extract banner urls %w", err)
	}

	defer res.Body.Close()

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing document with go query %w", err)
	}

	re, err := regexp.Compile(`\s+`)
	if err != nil {
		return nil, err
	}

	bannerData := []BannerData{}

	switch website.WebsiteID {
	case 1:
		// beautyfeatures
		doc.Find(".som-carousel a").Each(func(i int, s *goquery.Selection) {
			bf := BannerData{}

			if text := s.Text(); text != "" {
				text := re.ReplaceAllString(text, " ")

				bf.SupportingText = text
			}

			if value, found := s.Attr("href"); found {
				if strings.HasPrefix(value, "/") {
					value = website.URL + value
				}

				bf.Href = value
			}

			// For each item found, get the title
			if value, found := s.Find("img").Attr("src"); found {
				if strings.HasPrefix(value, "/") {
					value = website.URL + value
				}

				bf.Src = value
			}

			bannerData = append(bannerData, bf)
		})
	case 2:
		// lookfantastic
		doc.Find(".responsiveSlider_slideContainer").Each(func(i int, s *goquery.Selection) {
			lf := BannerData{}
			// logic goes here
			if imgSrc, found := s.Find("img").Attr("src"); found {
				if strings.HasPrefix(imgSrc, "/") {
					imgSrc = website.URL + imgSrc
				}

				lf.Src = imgSrc
			}

			if text := strings.TrimSpace(s.Text()); text != "" {
				text := re.ReplaceAllString(text, " ")

				lf.SupportingText = text
			}

			if href, found := s.Find("a").Attr("href"); found {
				if strings.HasPrefix(href, "/") {
					href = website.URL + href
				}

				lf.Href = href
			}

			bannerData = append(bannerData, lf)
		})
	case 3:
		// millies
		doc.Find(".homepage-slider-parent .swiper-wrapper img[width='720']").Each(func(i int, s *goquery.Selection) {
			millies := BannerData{}
			// For each item found, get the title
			if value, found := s.Attr("data-src"); found {
				value = strings.ReplaceAll(value, "{width}", "800")

				if strings.HasPrefix(value, "//") {
					value = "https:" + value
				}

				millies.Src = value
			}
			bannerData = append(bannerData, millies)
		})
	case 4:
		// mcCauleys
		doc.Find("[data-content-type=slide] [data-background-images]").Each(func(i int, s *goquery.Selection) {
			mc := BannerData{}
			if value, found := s.Attr("data-background-images"); found {
				var result = map[string]any{}
				value = strings.ReplaceAll(value, "\\\"", "\"")
				if err := json.Unmarshal([]byte(value), &result); err != nil {
					return
				}
				// Extract the mobile_image value
				if mobileImage, ok := result["mobile_image"].(string); ok {
					mc.Src = mobileImage
				}
			}
			bannerData = append(bannerData, mc)
		})
	default:
		return nil, fmt.Errorf("could not find banner extraction rules for website %s", website.WebsiteName)
	}

	return bannerData, nil
}

/* chat service begins */
func generateOfferDescription(websiteName string, banner BannerData) (string, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return "", errors.New("OPENAI_API_KEY env var not set")
	}

	c := openai.NewClient(key)

	model := openai.GPT4VisionPreview

	content := []openai.ChatMessagePart{
		{
			Type: "text",
			Text: fmt.Sprintf(`You are a joyful and excited social media manager for a health and beauty magazine with the goal of motivating people to take advantage of today's available beauty offers. 
			Tell your audience what the beauty retailer %s is advertising today and highlight any coupons if available. Keep your response short, playful and suitable for a tweet or instagram caption. 
			Do not acknowledge that you are AI.`, websiteName),
		},
	}

	if banner.SupportingText != "" {
		content = append(content, openai.ChatMessagePart{
			Type: "text",
			Text: fmt.Sprintf("For some additional context regarding this promotion please see the quoted text '%s'", banner.SupportingText),
		})
	}

	content = append(content, openai.ChatMessagePart{
		Type:     "image_url",
		ImageURL: &openai.ChatMessageImageURL{URL: banner.Src},
	})

	res, err := c.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model:     model,
		MaxTokens: 1000,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:         "user",
				MultiContent: content,
			},
		},
	})

	if err != nil {
		return "", err
	}

	return res.Choices[0].Message.Content, nil

}

/* chat service ends */

func dbGetBrands(db *sql.DB, params getAllBrandsParams) ([]Brand, error) {
	var q string
	var rows *sql.Rows
	var err error
	if params.Limit == 0 {
		q = `SELECT id, name, path, score FROM brands`
		rows, err = db.Query(q)
	} else {
		q = `SELECT id, name, path, score FROM brands LIMIT ? OFFSET ?`
		rows, err = db.Query(q, params.Limit, params.Offset)
	}

	if err != nil {
		return nil, fmt.Errorf("could not query brands: %w", err)
	}
	defer rows.Close()

	brands := make([]Brand, 0, params.Limit) // Preallocate slice with capacity if limit is supplied
	for rows.Next() {
		var brand Brand
		if err := rows.Scan(&brand.ID, &brand.Name, &brand.Path, &brand.Score); err != nil {
			return nil, fmt.Errorf("could not scan brand: %w", err)
		}
		brands = append(brands, brand)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return brands, nil
}
