package handlers

import (
	"beautybargains/models"
	"beautybargains/services"
	"encoding/json"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type Handler struct {
	s    *services.Service
	tmpl *template.Template
}

func NewHandler(s *services.Service, tmpl *template.Template) *Handler {
	return &Handler{s, tmpl}
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
	http.ServeFile(w, r, "robots.txt")
}

func (h *Handler) SitemapHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "sitemap.xml")
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

	products, err := h.s.GetWebsiteProducts(websiteID, limit, offset)
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}

	type WebsiteProductsPageData struct {
		BasePageData
		Products []models.Product
	}

	b := newBasePageData(r)

	err = h.tmpl.ExecuteTemplate(w, "products", WebsiteProductsPageData{b, products})
	if err != nil {
		log.Printf("Error: %v", err)
	}
}

func (h *Handler) GetBrands(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := h.paginator(r)

	brands, err := h.s.GetBrands(limit, offset)
	if err != nil {
		log.Printf("Error getting brands => %v", err)
		h.InternalError(w, r)
		return
	}

	brandCount, err := h.s.CountBrands()
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

	products, err := h.s.GetProductsWithBrandDetails(limit, offset)
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}

	productCount, err := h.s.CountProducts()
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
	brandName := vars["brand_path"]

	if brandName == "" {
		log.Printf("Warning: brandPath is empty")
	}

	limit, offset, page := h.paginator(r)

	brand, err := h.s.GetBrandByPath(brandName)
	if err != nil {
		log.Printf("Warning: could not find brand with path %s. Escaping and checking again...", brandName)
		// TODO cleaner system for generating paths for brands needed to avoid this mess
		brandName = url.QueryEscape(brandName)
		b, err := h.s.GetBrandByPath(brandName)
		if err != nil {
			log.Printf("Error getting brand by brand name %s => %v", brandName, err)
			h.InternalError(w, r)
			return
		}
		log.Printf("Info: found brand on second attempt id (%d) name (%s) path (%s)", b.ID, b.Name, b.Path)
		brand = b
	}

	products, err := h.s.GetProductsByBrand(brand.ID, limit, offset)
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}
	count, err := h.s.CountProductsByBrand(brand.ID)
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
	productErrors, err := h.s.GetProductErrors()
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

	product, prices, err := h.s.GetProductWithPrices(productID)
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

	website := models.Website{WebsiteName: name, URL: url, Country: country}

	err = h.s.CreateWebsite(website)
	if err != nil {
		h.InternalError(w, r)
		return
	}

	h.GetWebsites(w, r)
}

func (h *Handler) GetWebsites(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := h.paginator(r)
	websites, err := h.s.GetAllWebsites(limit, offset)
	if err != nil {
		h.InternalError(w, r)
		return
	}

	count, err := h.s.CountWebsites()
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

	website, err := h.s.GetWebsiteByID(websiteID)
	if err != nil {
		log.Printf("Error %v", err)
		h.InternalError(w, r)
		return
	}

	products, err := h.s.GetWebsiteProducts(websiteID, limit, offset)
	if err != nil {
		log.Printf("Error %v", err)
		h.InternalError(w, r)
		return
	}

	count, err := h.s.CountWebsiteProducts(websiteID)
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

	err = h.tmpl.ExecuteTemplate(w, "website", WebsitePageData{b, website, products, pagination})
	if err != nil {
		log.Printf("Error %v", err)
	}
}

func (h *Handler) GetUpdateProductsForm(w http.ResponseWriter, r *http.Request) {
	websites, err := h.s.GetAllWebsites(1000, 0)
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
		err = h.s.Subscribe(email)
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

	price, err := h.s.GetPriceData(productID, priceID)
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

	productURLs, err := h.s.NewProductURLs(urls)
	if err != nil {
		log.Printf("Error: %v", err)
		h.InternalError(w, r)
		return
	}

	log.Printf("%d new urls detected", len(productURLs))

	go (func() {
		_, err := h.s.CreateNewProducts(productURLs, websiteID)
		if err != nil {
			log.Printf("Error: %v", err)
		}
	})()

	website, err := h.s.GetWebsiteByID(websiteID)
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

	err = h.tmpl.ExecuteTemplate(w, "confirmupdateproducts", ConfirmUpdateProductsPageData{b, website, productURLs})
	if err != nil {
		log.Printf("Error: %v", err)
	}
}

func (h *Handler) GetPriceDrops(w http.ResponseWriter, r *http.Request) {
	limit, offset, _ := h.paginator(r)
	priceDrops, err := h.s.GetPriceDrops(limit, offset)
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

	err := h.s.VerifySubscription(token)
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

func (h *Handler) Promotions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	websiteName := vars["websiteName"]

	params := services.GetBannerPromotionsParams{}

	if websiteName != "" {
		params.WebsiteName = websiteName
	}

	promos, err := h.s.GetBannerPromotions(params)
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
