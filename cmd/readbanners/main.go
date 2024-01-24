package main

import (
	"beautybargains/models"
	"beautybargains/services"
	"database/sql"
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
		log.Fatal(err)
		return
	}

	db, err := sql.Open("sqlite3", "data")
	if err != nil {
		log.Fatal(err)
		return
	}
	defer db.Close()

	chat := services.InitChat()
	srv := services.NewService(db)

	websites, err := srv.GetAllWebsites(250, 0)
	if err != nil {
		log.Fatal(err)
		return
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS banner_promotions(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		websiteID INTEGER NOT NULL,
		bannerURL INTEGER NOT NULL,
		description TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		link TEXT
	)`)

	for _, website := range websites {
		bannerURLs, err := ExtractBannerURLs(website)
		if err != nil {
			log.Println(err)
			continue
		}

		uniqueBanners := []string{}
		for _, u := range bannerURLs {

			bannerExists, err := srv.DoesBannerPromotionExist(u)
			if err != nil {
				log.Printf("Error checking existance of banner %v", err)
				return
			}

			if bannerExists {
				continue
			}

			uniqueBanners = append(uniqueBanners, u)

		}

		for _, url := range uniqueBanners {
			description, err := chat.GetOfferDescription(website.WebsiteName, url)
			if err != nil {
				log.Fatal(err)
				return
			}
			err = srv.SaveBannerPromotion(website.WebsiteID, url, description, time.Now())

			if err != nil {
				log.Fatal(err)
				return
			}
		}

	}
}

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
		doc.Find("picture source").Each(func(i int, s *goquery.Selection) {
			// For each item found, get the title
			value, found := s.Attr("srcset")
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
	default:
		return []string{}, fmt.Errorf("Could not find banner extraction rules for website %s", website.WebsiteName)
	}

	return bannerURLs, nil
}
