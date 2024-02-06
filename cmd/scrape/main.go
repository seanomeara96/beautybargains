package main

import (
	"beautybargains/logger"
	"beautybargains/models"
	"beautybargains/scrapers"
	"beautybargains/services"
	"database/sql"
	"flag"
	"fmt"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func CountProductsToCrawl(db *sql.DB, websiteID int) (int, error) {
	var count int
	err := db.QueryRow("SELECT count(p.ProductID) FROM Products p LEFT JOIN (SELECT ProductID, MAX(timestamp) AS LatestTimestamp FROM PriceData GROUP BY ProductID) pd ON p.ProductID = pd.ProductID WHERE p.Error = false AND websiteID = ? AND pd.LatestTimestamp < datetime('now', '-12 hours')", websiteID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetNextProducts(db *sql.DB, websiteID int) ([]models.Product, error) {
	var products []models.Product
	rows, err := db.Query("SELECT p.ProductID, ProductName, WebsiteID, BrandID, Description, URL FROM Products p LEFT JOIN (SELECT ProductID, MAX(timestamp) AS LatestTimestamp FROM PriceData GROUP BY ProductID) pd ON p.ProductID = pd.ProductID WHERE p.Error = false AND websiteID = ? AND pd.LatestTimestamp < datetime('now', '-12 hours') LIMIT 1", websiteID)
	if err != nil {
		return products, err
	}
	defer rows.Close()

	for rows.Next() {
		var product models.Product
		err = rows.Scan(&product.ProductID, &product.ProductName, &product.WebsiteID, &product.BrandID, &product.Description, &product.URL)
		if err != nil {
			return products, err
		}
		products = append(products, product)
	}

	return products, nil

}

func main() {

	log := logger.NewLogger(logger.LogLevelError)

	_websiteID := flag.String("wid", "", "website id")

	_outFile := flag.String("file", "", "log output method")

	_level := flag.String("level", "error", "log level")

	flag.Parse()

	if *_websiteID == "" {
		log.Error("must supply a website ID")
		return
	}

	if *_outFile != "true" && *_outFile != "false" {
		log.Error("must supply a valid file flag value (true|false)")
		return
	}

	if *_level != "info" && *_level != "error" {
		log.Error("invalid log level supplied: " + *_level)
		return
	}

	level := *_level
	if level == "info" {
		log = logger.NewLogger(logger.LogLevelInfo)
	}

	outFile := *_outFile
	if outFile == "true" {
		_, err := logger.SetOutputToFile()
		if err != nil {
			log.Error(fmt.Sprintf("could not set output to file: %v", err))
			return
		}
	}

	websiteID, err := strconv.Atoi(*_websiteID)
	if err != nil {
		log.Error("invalid id format")
		return
	}

	db, err := sql.Open("sqlite3", "data")
	if err != nil {
		log.Error(fmt.Sprintf("could not connect to db: %v", err))
		return
	}
	defer db.Close()

	srv := services.NewService(db)
	scraper := scrapers.NewScraper(srv)

	/*
		Identify products that havent been crawled in x amount of time
	*/
	count, err := CountProductsToCrawl(db, websiteID)
	if err != nil {
		log.Error(fmt.Sprintf("could not count products left to crawl: %v", err))
		return
	}

	/*
		While there are products retrieve the necessary info
	*/
	for count > 0 {
		/*
			Get all the products that need crawling
		*/
		products, err := GetNextProducts(db, websiteID)
		if err != nil {
			log.Error(fmt.Sprintf("could not get next product to crawl: %v", err))
			return
		}
		/*
			For each product save the product data
		*/
		for _, product := range products {
			err = scraper.SaveProductData(product.WebsiteID, product.URL)
			if err != nil {
				/*
					If fail to save product data, save reason
				*/
				log.Error(fmt.Sprintf("could not save product data %v", err))
				/*
					If saving error fails, log the error
				*/
				serr := srv.SaveProductError(product.ProductID, true, err.Error())
				if serr != nil {
					log.Error(fmt.Sprintf("could not save error msg to product: %v", err))
					return
				}
			}
			log.Info(fmt.Sprintf("Successfully saved product: %d - %s", product.ProductID, product.ProductName))
		}
		/*
			Update count of products that need to be crawled
		*/
		count, err = CountProductsToCrawl(db, websiteID)
		if err != nil {
			panic(err)
		}
		/*
			Precautionary sleep to help avoid rate limits
		*/
		time.Sleep(1500 * time.Millisecond)
	}
}
