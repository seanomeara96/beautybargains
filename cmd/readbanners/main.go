package main

import (
	"beautybargains/internal/models"
	"beautybargains/internal/repositories/personarepo"
	"beautybargains/internal/repositories/postrepo"
	"beautybargains/internal/repositories/websiterepo"
	"beautybargains/internal/scripts"
	"beautybargains/internal/services/chatsvc"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"

	_ "github.com/mattn/go-sqlite3"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file. %v", err)
		return
	}

	chat := chatsvc.InitChat()

	personaDB, err := sql.Open("sqlite3", "data/models.db")
	if err != nil {
		log.Fatalf("Could not connect to persona repo. %v", err)
	}
	defer personaDB.Close()
	personaRepo := personarepo.New(personaDB)

	websiteDB, err := sql.Open("sqlite3", "data/websites.db")
	if err != nil {
		log.Fatalf("Could not connect to website database. %v", err)
	}
	defer websiteDB.Close()
	websiteRepo := websiterepo.New(websiteDB)

	postDB, err := sql.Open("sqlite3", "data/posts.db")
	if err != nil {
		log.Fatalf("Could not connect to post db. %v", err)
	}
	defer postDB.Close()
	postRepo := postrepo.New(postDB)

	websites, err := websiteRepo.GetAllWebsites(1000, 0)
	if err != nil {
		log.Fatalf("Error getting all websites. %v", err)
		return
	}

	for _, website := range websites {
		bannerURLs, err := ExtractBannerURLs(website)
		if err != nil {
			log.Printf("Error extracting banner urls from website. %v", err)
			continue
		}

		uniqueBanners := []string{}
		for _, u := range bannerURLs {

			bannerCount, err := postRepo.CountBySrc(u)
			if err != nil {
				log.Printf("Error checking existance of banner %v", err)
				return
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

			description, err := chat.GetOfferDescription(website.WebsiteName, url)
			if err != nil {
				log.Fatalf(`Error getting offer description from chatgpt. 
				WebsiteName: %s,
				URL: %s,
				%v`, website.WebsiteName, url, err)
				return
			}

			author, err := personaRepo.GetRandom()
			if err != nil {
				log.Printf("Warning: could not get author from repo. %v", err)
			}

			// I picked 8 randomly for author id
			authorID := 8
			if author != nil {
				authorID = author.ID
			}

			err = postRepo.Insert(website.WebsiteID, url, authorID, description, time.Now())
			if err != nil {
				log.Fatalf(`Error saving banner promotion. 
				Website id: %d,
				URL: %s,
				AuthorID: %d,
				Description: %s,
				%v`, website.WebsiteID, url, authorID, description, err)
				return
			}
		}

	}

	if err := scripts.ProcessHashtags(); err != nil {
		log.Fatal(err)
	}
}

/*
For a known website, retreive the banner urls and supply them in a string slice
*/
func ExtractBannerURLs(website models.Website) ([]string, error) {
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
