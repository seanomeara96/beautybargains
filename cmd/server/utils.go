package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func newHandleFunc(
	r *http.ServeMux, globalMiddleware []middleware, reportErr func(error) error,
) func(path string, fn handleFunc) {
	return func(path string, fn handleFunc) {
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
}

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

// to replace existing method. supports 'supporting text'
// for websites like lookfantastic and cult beauty that have carousels with text not embedded in the image
// can be passed to the llm for additional context

func getGoQueryPageDocument(url string) (*goquery.Document, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error sending get request to extract banner urls %w", err)
	}
	defer res.Body.Close()
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing document with go query %w", err)
	}
	return doc, nil
}

type BannerData struct {
	Src            string
	SupportingText string
	Href           string
}

func extractWebsiteBannerURLs(website Website) ([]BannerData, error) {
	doc, err := getGoQueryPageDocument(website.URL)
	if err != nil {
		return nil, err
	}

	bannerData := []BannerData{}

	switch website.WebsiteID {
	case 1:
		// beautyfeatures
		doc.Find(".som-carousel a").Each(func(i int, s *goquery.Selection) {
			bf := BannerData{}

			regex, err := regexp.Compile(`\s+`)
			if err != nil {
				log.Println("Warning regex for beautyfeatures could not compile")
				return
			}

			if text := s.Text(); text != "" {
				text := regex.ReplaceAllString(text, " ")

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
		doc.Find(".carousel-item").Each(func(i int, selection *goquery.Selection) {
			lf := BannerData{}
			// logic goes here
			if imgSrc, found := selection.Find("[media='(max-width: 640px)']").Attr("srcset"); found {
				if strings.HasPrefix(imgSrc, "/") {
					imgSrc = website.URL + imgSrc
				}

				if strings.Contains(imgSrc, " ") {
					imgSrc = strings.Split(imgSrc, " ")[0]
				}

				lf.Src = imgSrc
			}
			regex, err := regexp.Compile(`\s+`)
			if err != nil {
				log.Println("Warning regex for lookfantastic could not compile")
				return
			}

			if text := strings.TrimSpace(selection.Text()); text != "" {
				text := regex.ReplaceAllString(text, " ")

				lf.SupportingText = text
			}

			if href, found := selection.Find("a").Attr("href"); found {
				if strings.HasPrefix(href, "/") {
					href = website.URL + href
				}

				lf.Href = href
			}

			bannerData = append(bannerData, lf)
		})
	case 3:
		// millies
		doc.Find(".swiper-wrapper .slide-img.md\\:hidden").Each(func(i int, s *goquery.Selection) {
			millies := BannerData{}
			// For each item found, get the title
			if value, found := s.Attr("src"); found {
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

	case 5:
		// skin shop
		doc.Find(`.slideshow-slide .background-image--mobile img`).Each(func(i int, s *goquery.Selection) {

			if imgSrc, found := s.Attr("src"); found {
				if strings.HasPrefix(imgSrc, "//") {
					imgSrc = "https:" + imgSrc
				}

				bannerData = append(bannerData, BannerData{Src: imgSrc})
			}
		})
	case 6:
		// cloud 10?
		doc.Find(".homepage-slider img.desktop-hide").Each(func(i int, s *goquery.Selection) {

			if imgSrc, found := s.Attr("src"); found {
				if strings.HasPrefix(imgSrc, "//") {
					imgSrc = "https:" + imgSrc
				}

				bannerData = append(bannerData, BannerData{Src: imgSrc})
			}
		})

	case 7:
		doc.Find("body > div.home-main > div:nth-child(5) > div .slide-image").Each(func(i int, s *goquery.Selection) {
			if imgSrc, found := s.Attr("src"); found {
				if strings.HasPrefix(imgSrc, "/") {
					imgSrc = website.URL + imgSrc
				}

				bannerData = append(bannerData, BannerData{Src: imgSrc})
			}
		})
	default:
		return nil, fmt.Errorf("could not find banner extraction rules for website %s", website.WebsiteName)
	}
	return bannerData, nil
}
