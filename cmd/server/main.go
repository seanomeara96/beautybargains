package main

import (
	"beautybargains/handlers"
	"beautybargains/services"
	"database/sql"
	"flag"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	portValue := flag.String("port", "", "http port")
	flag.Parse()
	if *portValue == "" {
		log.Println("port is required via -port flag")
		return
	}
	port := *portValue

	db, err := sql.Open("sqlite3", "data")
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("web").Funcs(funcMap).ParseGlob("templates/**/*.tmpl")
	if err != nil {
		panic(err)
	}

	service := services.NewService(db)

	handler := handlers.NewHandler(service, tmpl)

	r := mux.NewRouter()
	r.StrictSlash(true)

	assetsDir := http.Dir("assets/dist")

	assetsFileServer := http.FileServer(assetsDir)
	r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", assetsFileServer))
	r.HandleFunc("/", handler.Home)
	r.HandleFunc("/websites/{website_id}/products", handler.GetWebsiteProducts).Methods(http.MethodGet)
	r.HandleFunc("/products/update", handler.GetUpdateProductsForm).Methods(http.MethodGet)
	r.HandleFunc("/products/update", handler.ProcessUpdateProductsForm).Methods(http.MethodPost)
	r.HandleFunc("/products/errors", handler.ListProductErrors).Methods(http.MethodGet)
	r.HandleFunc("/products/{product_id}", handler.GetProductWithPrices).Methods(http.MethodGet)
	r.HandleFunc("/products/{product_id}/prices/{price_id}", handler.GetPriceData).Methods(http.MethodGet)
	r.HandleFunc("/add-website", handler.GetAddWebsiteForm).Methods(http.MethodGet)
	r.HandleFunc("/add-website", handler.ProcessAddWebsiteFormSubmission).Methods(http.MethodPost)
	r.HandleFunc("/websites", handler.GetWebsites).Methods(http.MethodGet)
	r.HandleFunc("/websites/{website_id}", handler.GetWebsite).Methods(http.MethodGet)
	r.HandleFunc("/products", handler.GetProducts).Methods(http.MethodGet)
	r.HandleFunc("/brands", handler.GetBrands).Methods(http.MethodGet)
	r.HandleFunc("/brands/{brand_path}", handler.GetProductsByBrand).Methods(http.MethodGet)
	r.HandleFunc("/price-drops", handler.GetPriceDrops).Methods(http.MethodGet)
	r.HandleFunc("/subscribe", handler.Subscribe).Methods(http.MethodPost)
	r.HandleFunc("/subscribe", handler.Home).Methods(http.MethodGet)
	r.HandleFunc("/subscribe/verify", handler.VerifySubscription).Methods(http.MethodGet)
	log.Println("listening on " + port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
