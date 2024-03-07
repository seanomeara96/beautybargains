package main

import (
	"beautybargains/internal/models"
	"beautybargains/internal/repositories/bannerpromotionrepo"
	"beautybargains/internal/repositories/brandrepo"
	"beautybargains/internal/repositories/personarepo"
	"beautybargains/internal/repositories/pricedatarepo"
	"beautybargains/internal/repositories/productrepo"
	"beautybargains/internal/repositories/subscriberrepo"
	"beautybargains/internal/repositories/websiterepo"
	"beautybargains/internal/services/bannerpromotionsvc"
	"beautybargains/internal/services/brandsvc"
	"beautybargains/internal/services/mailingsvc"
	"beautybargains/internal/services/personasvc"
	"beautybargains/internal/services/pricedatasvc"
	"beautybargains/internal/services/productsvc"
	"beautybargains/internal/services/websitesvc"
	"beautybargains/web/handlers"
	"beautybargains/web/routers"
	"database/sql"
	"flag"
	"html/template"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	_port := flag.String("port", "", "http port")
	_mode := flag.String("mode", "", "deployment mode")

	flag.Parse()

	port := *_port
	mode := models.Mode(*_mode)

	if port == "" {
		log.Println("Port is required via -port flag.")
		return
	}

	if mode == "" {
		log.Println("No mode was supplied, starting server in development mode.")
		mode = models.Dev
	}

	tmpl, err := template.New("web").Funcs(funcMap).ParseGlob("web/templates/**/*.tmpl")
	if err != nil {
		log.Printf("Error parsing templates. %v", err)
		return
	}

	personaDB := dbConnect("data/models.db")
	personaRepo := personarepo.New(personaDB)
	personaService := personasvc.New(personaRepo)
	defer personaDB.Close()

	websiteDB := dbConnect("data/websites.db")
	websiteRepo := websiterepo.New(websiteDB)
	websiteService := websitesvc.New(websiteRepo)
	defer websiteDB.Close()

	brandDB := dbConnect("data/brands.db")
	brandRepo := brandrepo.New(brandDB)
	brandService := brandsvc.New(brandRepo)
	defer brandDB.Close()

	pricedataDB := dbConnect("data/pricedata.db")
	pricedataRepo := pricedatarepo.New(pricedataDB)
	pricedataService := pricedatasvc.New(pricedataRepo)
	defer pricedataDB.Close()

	productDB := dbConnect("data/products.db")
	productRepo := productrepo.New(productDB)
	// requires brands and prices
	productService := productsvc.New(productRepo, brandRepo, pricedataRepo)
	defer productDB.Close()

	subscriberDB := dbConnect("data/subscribers.db")
	subscriberRepo := subscriberrepo.New(subscriberDB)
	mailingService := mailingsvc.New(subscriberRepo)
	defer subscriberDB.Close()

	bannerpromotionDB := dbConnect("data/bannerpromotions.db")
	bannerpromotionRepo := bannerpromotionrepo.New(bannerpromotionDB)
	bannerpromotionService := bannerpromotionsvc.New(bannerpromotionRepo)
	defer bannerpromotionDB.Close()

	handler := handlers.NewHandler(
		tmpl,
		personaService,
		websiteService,
		productService,
		brandService,
		mailingService,
		pricedataService,
		bannerpromotionService,
	)
	router := routers.NewRouter(mode, handler)

	log.Println("Server listening on http://localhost:" + port)
	if err = http.ListenAndServe(":"+port, router); err != nil {
		log.Printf("Error during listen and serve. %v", err)
	}
}

func dbConnect(path string) *sql.DB {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		panic(err)
	}
	return db
}
