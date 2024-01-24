package routers

import (
	"beautybargains/handlers"
	"beautybargains/models"
	"net/http"

	"github.com/gorilla/mux"
)

func NewRouter(mode models.Mode, handler *handlers.Handler) *mux.Router {
	r := mux.NewRouter()
	r.StrictSlash(true)

	assetsDir := http.Dir("assets/dist")
	assetsFileServer := http.FileServer(assetsDir)
	r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", assetsFileServer))

	/*
		Serve robots.txt & sitemap
	*/
	r.HandleFunc("/robots.txt", handler.RobotsHandler)
	r.HandleFunc("/sitemap.xml", handler.SitemapHandler)

	/*
		Home / Index Handler
	*/
	r.HandleFunc("/", handler.Home)

	/*
		Promotions Handlers
	*/
	r.HandleFunc("/promotions/", handler.Promotions).Methods(http.MethodGet)
	r.HandleFunc("/promotions/{websiteName}", handler.Promotions).Methods(http.MethodGet)
	r.HandleFunc("/{websiteName}/promotions/", handler.Promotions).Methods(http.MethodGet)

	if mode == models.Dev {
		r.HandleFunc("/websites/{website_id}/products/", handler.GetWebsiteProducts).Methods(http.MethodGet)
		r.HandleFunc("/products/update/", handler.GetUpdateProductsForm).Methods(http.MethodGet)
		r.HandleFunc("/products/update/", handler.ProcessUpdateProductsForm).Methods(http.MethodPost)
		r.HandleFunc("/products/errors/", handler.ListProductErrors).Methods(http.MethodGet)
		r.HandleFunc("/products/{product_id}/", handler.GetProductWithPrices).Methods(http.MethodGet)
		r.HandleFunc("/products/{product_id}/prices/{price_id}/", handler.GetPriceData).Methods(http.MethodGet)
		r.HandleFunc("/add-website/", handler.GetAddWebsiteForm).Methods(http.MethodGet)
		r.HandleFunc("/add-website/", handler.ProcessAddWebsiteFormSubmission).Methods(http.MethodPost)
		r.HandleFunc("/websites/", handler.GetWebsites).Methods(http.MethodGet)
		r.HandleFunc("/websites/{website_id}/", handler.GetWebsite).Methods(http.MethodGet)
		r.HandleFunc("/products/", handler.GetProducts).Methods(http.MethodGet)
		r.HandleFunc("/brands/", handler.GetBrands).Methods(http.MethodGet)
		r.HandleFunc("/brands/{brand_path}/", handler.GetProductsByBrand).Methods(http.MethodGet)
		r.HandleFunc("/price-drops/", handler.GetPriceDrops).Methods(http.MethodGet)
		r.HandleFunc("/subscribe/", handler.Subscribe).Methods(http.MethodPost)
		r.HandleFunc("/subscribe/", handler.Home).Methods(http.MethodGet)
		r.HandleFunc("/subscribe/verify", handler.VerifySubscription).Methods(http.MethodGet)
	}

	return r

}
