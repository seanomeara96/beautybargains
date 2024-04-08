package main

import (
	"beautybargains/internal/logger"
	"beautybargains/internal/repositories/brandrepo"
	"beautybargains/internal/repositories/pricedatarepo"
	"beautybargains/internal/repositories/productrepo"
	"beautybargains/internal/scrapers"
	"database/sql"
	"flag"
	"fmt"
	"math"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

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

	productDB, err := sql.Open("sqlite3", "data/products.db")
	if err != nil {
		panic(err)
	}
	defer productDB.Close()
	productRepo := productrepo.New(productDB)

	pricedataDB, err := sql.Open("sqlite3", "data/pricedata.db")
	if err != nil {
		panic(err)
	}
	defer pricedataDB.Close()
	pricedataRepo := pricedatarepo.New(pricedataDB)

	brandDB, err := sql.Open("sqlite3", "data/brands.db")
	if err != nil {
		panic(err)
	}
	defer brandDB.Close()
	brandRepo := brandrepo.New(brandDB)

	scraper := scrapers.NewScraper(productRepo, pricedataRepo, brandRepo)

	limit := 50
	offset := 1
	productCount, err := productRepo.CountByWebsiteID(websiteID)
	if err != nil {
		panic(err)
	}

	maxPages := RoundUpToInt(float64(productCount) / float64(limit))
	for i := 0; i < maxPages; i++ {
		offset = (i * limit)
		products, err := productRepo.GetWebsiteProducts(websiteID, limit, offset)
		if err != nil {
			panic(fmt.Errorf("Could not get website products. %w", err))
		}

		for _, product := range products {
			countPrices, err := pricedataRepo.CountByProductID(product.ProductID)
			if err != nil {
				panic(err)
			}
			hasPrices := countPrices > 0
			if hasPrices {
				price, err := pricedataRepo.GetLatestPrice(product.ProductID)
				if err != nil {
					panic(err)
				}
				if MoreThanNDaysOld(12, price.Timestamp) {
					err = scraper.SaveProductData(product.WebsiteID, product.URL)
					if err != nil {
						/*
							If fail to save product data, save reason
						*/
						log.Warning(fmt.Sprintf("could not save product data %v", err))
						/*
							If saving error fails, log the error
						*/
						serr := productRepo.SaveProductError(product.ProductID, true, err.Error())
						if serr != nil {
							log.Error(fmt.Sprintf("could not save error msg to product: %v", err))
						}
					}
				}
			} else {
				err = scraper.SaveProductData(product.WebsiteID, product.URL)
				if err != nil {
					/*
						Cant save product and has ne previous prices, just delete the product
					*/
					err := productRepo.DeleteProduct(product.ProductID)
					if err != nil {
						panic(err)
					}
				}
			}
		}
	}
}

func RoundUpToInt(n float64) int {
	return int(math.Round(n/2) * 2)
}

func MoreThanNDaysOld(n time.Duration, recordedAt time.Time) bool {
	nDays := n * 24 * time.Hour            // 27 now
	duration := time.Now().Sub(recordedAt) // 15 - 1 = 13
	return duration > nDays
}
