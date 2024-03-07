package handlers

import (
	"beautybargains/internal/models"
	"beautybargains/internal/repositories/bannerpromotionrepo"
	"beautybargains/internal/services/bannerpromotionsvc"
	"beautybargains/internal/services/brandsvc"
	"beautybargains/internal/services/mailingsvc"
	"beautybargains/internal/services/personasvc"
	"beautybargains/internal/services/pricedatasvc"
	"beautybargains/internal/services/productsvc"
	"beautybargains/internal/services/websitesvc"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type services struct {
	personas         *personasvc.Service
	websites         *websitesvc.Service
	products         *productsvc.Service
	brands           *brandsvc.Service
	mailing          *mailingsvc.Service
	prices           *pricedatasvc.Service
	bannerpromotions *bannerpromotionsvc.Service
}

type Handler struct {
	services services
	tmpl     *template.Template
}

func NewHandler(
	tmpl *template.Template,
	personas *personasvc.Service,
	websites *websitesvc.Service,
	products *productsvc.Service,
	brands *brandsvc.Service,
	mailing *mailingsvc.Service,
	prices *pricedatasvc.Service,
	bannerpromotions *bannerpromotionsvc.Service,
) *Handler {
	h := Handler{}
	h.tmpl = tmpl
	h.services.personas = personas
	h.services.websites = websites
	h.services.products = products
	h.services.brands = brands
	h.services.mailing = mailing
	h.services.prices = prices
	h.services.bannerpromotions = bannerpromotions
	return &h
}

type MenuItem struct {
	Path string
	Name string
}

var menuItems = []MenuItem{
	MenuItem{"/", "Home"},
	MenuItem{"/promotions/", "Promotions"},
	MenuItem{"/websites/", "Websites"},
	MenuItem{"/products/", "Products"},
	MenuItem{"/brands/", "Brands"},
}

type BasePageData struct {
	Request   *http.Request
	MenuItems []MenuItem
}

func newBasePageData(r *http.Request) BasePageData {
	return BasePageData{
		Request:   r,
		MenuItems: menuItems,
	}
}

type Pagination struct {
	PageNumber int
	MaxPages   int
}

func (h *Handler) paginator(r *http.Request) (int, int, int) {
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

func (h *Handler) RobotsHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/robots.txt")
}

func (h *Handler) SitemapHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/sitemap.xml")
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {

	type HomePageData struct {
		BasePageData
	}

	b := newBasePageData(r)

	err := h.tmpl.ExecuteTemplate(w, "home", HomePageData{b})
	if err != nil {
		log.Fatal(err)
	}
}

func (h *Handler) InternalError(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (h *Handler) GetWebsiteProducts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	websiteID, err := strconv.Atoi(vars["website_id"])
	if err != nil {
		h.InternalError(w, r)
		return
	}

	limit, offset, _ := h.paginator(r)

	website, err := h.services.websites.Get(websiteID)
	if err != nil {
		h.InternalError(w, r)
		return
	}

	products, err := h.services.products.GetByWebsiteID(websiteID, limit, offset)
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}

	type WebsiteProductsPageData struct {
		BasePageData
		Website  *models.Website
		Products []models.Product
	}

	b := newBasePageData(r)

	err = h.tmpl.ExecuteTemplate(w, "products", WebsiteProductsPageData{b, website, products})
	if err != nil {
		log.Printf("Error: %v", err)
	}
}

func (h *Handler) GetBrands(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := h.paginator(r)

	brands, err := h.services.brands.GetAll(limit, offset)
	if err != nil {
		log.Printf("Error getting brands => %v", err)
		h.InternalError(w, r)
		return
	}

	brandCount, err := h.services.brands.Count()
	if err != nil {
		log.Printf("Error counting brands => %v", err)
		h.InternalError(w, r)
		return
	}

	maxPages := int(math.Ceil(float64(brandCount) / float64(limit)))

	b := newBasePageData(r)

	pagination := Pagination{page, maxPages}

	type BrandPageData struct {
		BasePageData
		Brands     []models.Brand
		Pagination Pagination
	}

	brandPageData := BrandPageData{b, brands, pagination}

	err = h.tmpl.ExecuteTemplate(w, "brands", brandPageData)
	if err != nil {
		log.Println(err)
	}
}

func (h *Handler) GetProducts(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := h.paginator(r)

	products, err := h.services.products.GetProductsWithBrandDetails(limit, offset)
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}

	productCount, err := h.services.products.Count()
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}

	maxPages := int(math.Ceil(float64(productCount) / float64(limit)))

	b := newBasePageData(r)
	pagination := Pagination{page, maxPages}

	type ProductsPageData struct {
		BasePageData
		Products   []models.Product
		Pagination Pagination
	}

	err = h.tmpl.ExecuteTemplate(w, "products", ProductsPageData{b, products, pagination})
	if err != nil {
		log.Printf("Error: %v", err)
	}
}
func (h *Handler) GetProductsByBrand(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	brandPath := vars["brand_path"]

	if brandPath == "" {
		log.Printf("Warning: brandPath is empty")
	}

	limit, offset, page := h.paginator(r)

	brand, err := h.services.brands.GetByPath(brandPath)
	if err != nil {
		log.Printf("Warning: could not find brand with path %s. Escaping and checking again...", brandPath)
		// TODO cleaner system for generating paths for brands needed to avoid this mess
		brandPath = url.QueryEscape(brandPath)
		b, err := h.services.brands.GetByPath(brandPath)
		if err != nil {
			log.Printf("Error getting brand by brand name %s => %v", brandPath, err)
			h.InternalError(w, r)
			return
		}
		log.Printf("Info: found brand on second attempt id (%d) name (%s) path (%s)", b.ID, b.Name, b.Path)
		brand = b
	}

	products, err := h.services.products.GetByBrandID(brand.ID, limit, offset)
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}
	count, err := h.services.products.CountByBrandID(brand.ID)
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}

	if len(products) < 1 {
		log.Printf("Warning: there should be products associated with this brand")
	}

	for i, _ := range products {
		products[i].Brand = brand
	}

	type ProductsPageData struct {
		BasePageData
		Products   []models.Product
		Pagination Pagination
	}

	b := newBasePageData(r)

	maxPages := int(math.Ceil(float64(count) / float64(limit)))
	pagination := Pagination{page, maxPages}

	err = h.tmpl.ExecuteTemplate(w, "products", ProductsPageData{b, products, pagination})
	if err != nil {
		log.Printf("Error rendering products by brand: %v", err)
	}
}

func (h *Handler) ListProductErrors(w http.ResponseWriter, r *http.Request) {
	productErrors, err := h.services.products.GetErrors()
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}

	err = h.tmpl.ExecuteTemplate(w, "producterrors", productErrors)
	if err != nil {
		log.Printf("Template Error: %v", err)
	}
}

func areOnSameDate(timestamp1, timestamp2 time.Time) bool {
	return timestamp1.Year() == timestamp2.Year() &&
		timestamp1.Month() == timestamp2.Month() &&
		timestamp1.Day() == timestamp2.Day()
}

type DatePrice struct {
	Date  string
	Price float64
}

func generateDateArray(minTime, maxTime models.PriceData) []DatePrice {
	var dateArray []DatePrice

	for current := minTime.Timestamp; current.Before(maxTime.Timestamp) || current.Equal(maxTime.Timestamp); current = current.Add(24 * time.Hour) {
		dateArray = append(dateArray, DatePrice{current.Format("2006-01-02"), 0})
	}

	return dateArray
}

func (h *Handler) GetProductWithPrices(w http.ResponseWriter, r *http.Request) {
	_productID := mux.Vars(r)["product_id"]

	productID, err := strconv.Atoi(_productID)
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}

	product, prices, err := h.services.products.GetWithPrices(productID)
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}

	type ProductData struct {
		Dates  []string  `json:"dates"`
		Prices []float64 `json:"prices"`
	}

	var productData ProductData
	for _, price := range prices {
		productData.Dates = append(productData.Dates, price.Timestamp.Format(time.DateOnly))
		productData.Prices = append(productData.Prices, price.Price)
	}

	productDataJSON, err := json.Marshal(productData)
	if err != nil {
		log.Printf("Error marshalling json %v", err)
		h.InternalError(w, r)
		return
	}

	type ProductPageData struct {
		BasePageData
		Product         models.Product
		Prices          []models.PriceData
		ProductDataJSON string
	}

	b := newBasePageData(r)

	err = h.tmpl.ExecuteTemplate(w, "product", ProductPageData{b, product, prices, string(productDataJSON)})
	if err != nil {
		log.Printf("Error: %v", err)
	}
}

func (h *Handler) GetAddWebsiteForm(w http.ResponseWriter, r *http.Request) {
	h.tmpl.ExecuteTemplate(w, "addwebsite", nil)
}
func (h *Handler) ProcessAddWebsiteFormSubmission(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}

	name := r.Form.Get("name")
	url := r.Form.Get("url")
	country := r.Form.Get("country")

	err = h.services.websites.Create(name, url, country)
	if err != nil {
		h.InternalError(w, r)
		return
	}

	h.GetWebsites(w, r)
}

func (h *Handler) GetWebsites(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := h.paginator(r)
	websites, err := h.services.websites.GetAll(limit, offset)
	if err != nil {
		h.InternalError(w, r)
		return
	}

	count, err := h.services.websites.Count()
	if err != nil {
		h.InternalError(w, r)
		return
	}

	maxPages := int(math.Ceil(float64(count) / float64(limit)))

	type WebsitesPageData struct {
		BasePageData
		Websites   []models.Website
		Pagination Pagination
	}

	b := newBasePageData(r)
	pagination := Pagination{page, maxPages}

	data := WebsitesPageData{b, websites, pagination}

	err = h.tmpl.ExecuteTemplate(w, "websites", data)
	if err != nil {
		log.Printf("Error: %v", err)
	}

}

func (h *Handler) GetWebsite(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	limit, offset, page := h.paginator(r)

	websiteID, err := strconv.Atoi(vars["website_id"])
	if err != nil {
		log.Printf("Error %v", err)
		h.InternalError(w, r)
		return
	}

	website, err := h.services.websites.Get(websiteID)
	if err != nil {
		log.Printf("Error %v", err)
		h.InternalError(w, r)
		return
	}

	products, err := h.services.products.GetByWebsiteID(websiteID, limit, offset)
	if err != nil {
		log.Printf("Error %v", err)
		h.InternalError(w, r)
		return
	}

	count, err := h.services.products.CountByWebsiteID(websiteID)
	if err != nil {
		log.Printf("Error %v", err)
		h.InternalError(w, r)
		return
	}

	maxPages := int(math.Ceil(float64(count) / float64(limit)))

	type WebsitePageData struct {
		BasePageData
		Website    models.Website
		Products   []models.Product
		Pagination Pagination
	}

	b := newBasePageData(r)

	pagination := Pagination{page, maxPages}

	err = h.tmpl.ExecuteTemplate(w, "website", WebsitePageData{b, *website, products, pagination})
	if err != nil {
		log.Printf("Error %v", err)
	}
}

func (h *Handler) GetUpdateProductsForm(w http.ResponseWriter, r *http.Request) {
	websites, err := h.services.websites.GetAll(1000, 0)
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}

	type UpdateProductFormData struct{ Websites []models.Website }

	err = h.tmpl.ExecuteTemplate(w, "updateproducts", UpdateProductFormData{websites})
	if err != nil {
		log.Printf("Error: %v", err)
	}
}

func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()

	email := r.FormValue("email")
	consent := r.FormValue("consent")

	if consent == "on" {
		err = h.services.mailing.Subscribe(email)
		if err != nil {
			log.Printf("Error subscribing user: %v", err)
		}

		err = h.tmpl.ExecuteTemplate(w, "subscriptionsuccess", nil)
		if err != nil {
			log.Printf("Error: %v", err)
		}
		return
	}

	err = h.tmpl.ExecuteTemplate(w, "subscriptionform", nil)
	if err != nil {
		log.Printf("Error: %v", err)
	}
}

func (h *Handler) GetPriceData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_productID := vars["product_id"]
	_priceID := vars["price_id"]

	productID, err := strconv.Atoi(_productID)
	if err != nil {
		h.InternalError(w, r)
		return
	}

	priceID, err := strconv.Atoi(_priceID)
	if err != nil {
		h.InternalError(w, r)
		return
	}

	price, err := h.services.prices.GetByProductID(productID, priceID)
	if err != nil {
		h.InternalError(w, r)
		return
	}

	err = h.tmpl.ExecuteTemplate(w, "pricedata", price)
	if err != nil {
		log.Printf("Error: %v", err)
	}
}

func (h *Handler) ProcessUpdateProductsForm(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}

	_websiteID := r.Form.Get("website")
	websiteID, err := strconv.Atoi(_websiteID)
	if err != nil {
		h.InternalError(w, r)
		return
	}

	_urls := r.Form.Get("product_urls")
	urls := strings.Split(_urls, "\n")

	log.Printf("%d urls submitted", len(urls))

	productURLs, err := h.services.products.FilterNewProductURLs(urls)
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}

	log.Printf("%d new urls detected", len(productURLs))

	go (func() {
		_, err := h.services.products.BatchCreate(productURLs, websiteID)
		if err != nil {
			log.Printf("Error: %v", err)
		}
	})()

	website, err := h.services.websites.Get(websiteID)
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}

	type ConfirmUpdateProductsPageData struct {
		BasePageData
		Website     models.Website
		ProductURLs []string
	}

	b := newBasePageData(r)

	err = h.tmpl.ExecuteTemplate(w, "confirmupdateproducts", ConfirmUpdateProductsPageData{b, *website, productURLs})
	if err != nil {
		log.Printf("Error: %v", err)
	}
}

func (h *Handler) GetPriceDrops(w http.ResponseWriter, r *http.Request) {
	limit, offset, _ := h.paginator(r)
	priceDrops, err := h.services.prices.GetPriceDrops(limit, offset)
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}

	type PriceChangesData struct{ PriceChanges []models.PriceChange }

	err = h.tmpl.ExecuteTemplate(w, "pricechanges", PriceChangesData{priceDrops})
	if err != nil {
		log.Printf("Error: %v", err)
	}

}

func (h *Handler) VerifySubscription(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	token := vars.Get("token")

	if token == "" {
		log.Println("Warning: subscription verification attempted with no token")
		h.Home(w, r)
		return
	}

	err := h.services.mailing.VerifySubscription(token)
	if err != nil {
		log.Printf("Error: could not verify subscription => %v", err)
		h.InternalError(w, r)
		return
	}

	// subscription confirmed

	type SubscriptionVerificationPageData struct {
		BasePageData
	}

	b := newBasePageData(r)

	err = h.tmpl.ExecuteTemplate(w, "subscriptionverification", SubscriptionVerificationPageData{b})
	if err != nil {
		log.Printf("Error: could not render subscription verification page => %v", err)
	}

}

func (h *Handler) Feed(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	websiteName := vars["websiteName"]

	hashtagQuery := r.URL.Query().Get("hashtag")

	params := bannerpromotionrepo.GetBannerPromotionsParams{
		SortByTimestampDesc: true,
	}

	if websiteName != "" {
		website, err := h.services.websites.GetByName(websiteName)
		if err == nil {
			params.WebsiteID = website.WebsiteID
		}
	}

	if hashtagQuery != "" {
		params.Hashtag = hashtagQuery
	}

	personas, err := h.services.personas.GetAll()
	if err != nil {
		log.Printf("Could not get all personas feed page. %v", err)
		h.InternalError(w, r)
		return
	}

	promos, err := h.services.bannerpromotions.GetAll(params)
	if err != nil {
		log.Printf("Could not get banner promotions for feed page.  %v", err)
		h.InternalError(w, r)
		return
	}

	events := []models.Event{}
	for i := 0; i < len(promos); i++ {
		e := models.Event{}

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
			unit = "Hours"
			magnitude = hours
		}

		// Step 3: Format String
		e.Content.TimeElapsed = fmt.Sprintf("%d %s ago", magnitude, unit)
		e.Meta.Src = &promos[i].BannerURL

		pattern := regexp.MustCompile(`#(\w+)`)

		extraText := promos[i].Description

		matches := pattern.FindAllStringSubmatch(extraText, -1)

		for _, match := range matches {
			phrase := strings.ToLower(match[1])
			extraText = strings.Replace(extraText, match[0], fmt.Sprintf("<a class='text-blue-500' href='/feed/?hashtag=%s'>%s</a>", phrase, match[0]), 1)
		}

		extraTextHTML := template.HTML(extraText)

		e.Content.ExtraText = &extraTextHTML
		website, err := h.services.websites.Get(promos[i].WebsiteID)
		if err != nil {
			log.Printf("Could not get website by id %d. %v", promos[i].WebsiteID, err)
			h.InternalError(w, r)
			return
		}
		e.Content.Summary = fmt.Sprintf("posted an update about %s", website.WebsiteName)
		e.Content.ExtraImages = nil
		events = append(events, e)
	}

	websites, err := h.services.websites.GetAll(10, 0)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	type FeedPageData struct {
		BasePageData
		Events   []models.Event
		Websites []models.Website
		Trending []models.Trending
	}

	b := newBasePageData(r)

	err = h.tmpl.ExecuteTemplate(w, "feedpage", FeedPageData{b, events, websites, models.DummyTrending})
	if err != nil {
		log.Printf("Error: %v", err)
	}

}

func (h *Handler) Promotions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	websiteName := vars["websiteName"]

	params := bannerpromotionrepo.GetBannerPromotionsParams{
		SortByTimestampDesc: true,
	}

	if websiteName != "" {
		website, err := h.services.websites.GetByName(websiteName)
		if err == nil {
			params.WebsiteID = website.WebsiteID
		}
	}

	promos, err := h.services.bannerpromotions.GetAll(params)
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}

	type PromoPageData struct {
		BasePageData
		Promotions []models.BannerPromotion
	}

	b := newBasePageData(r)

	err = h.tmpl.ExecuteTemplate(w, "promotionspage", PromoPageData{b, promos})
	if err != nil {
		log.Printf("Error: %v", err)
	}

}
