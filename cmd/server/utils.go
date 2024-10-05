package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func reportErr(err error) {
	log.Print(err)
}

func newHandleFunc(r *http.ServeMux, globalMiddleware []middleware) func(path string, fn handleFunc) {
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

/* template functions end */

func renderPage(mode Mode, w io.Writer, name string, data map[string]any) error {
	templateData := map[string]any{
		"Env": mode,
	}

	for k, v := range data {
		templateData[k] = v
	}

	return render(w, name, templateData)
}

func render(mode Mode, w io.Writer, name string, data any) error {
	var buf bytes.Buffer
	if err := tmpl(mode).ExecuteTemplate(&buf, name, data); err != nil {
		return fmt.Errorf("render error: %w", err)
	}
	_, err := buf.WriteTo(w)
	return err
}

var _tmpl *template.Template

func tmpl(mode Mode) *template.Template {
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
