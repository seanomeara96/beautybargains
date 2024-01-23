package scrapers

import (
	"beautybargains/models"
	"beautybargains/services"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"net/http"
	"net/url"
	"regexp"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/mattn/go-sqlite3"
)

type ProductDataMultipleOffers struct {
	Brand struct {
		Name string `json:"name"`
	} `json:"brand"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Image       string          `json:"image"`
	Offers      []ProductOffers `json:"offers"`
}

type ProductDataSingleOffer struct {
	Brand struct {
		Name string `json:"name"`
	} `json:"brand"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Image       string        `json:"image"`
	Offers      ProductOffers `json:"offers"`
}

type ProductOffers struct {
	Image        string `json:"image"`
	Name         string `json:"name"`
	Price        string `json:"price"`
	Sku          string `json:"sku"`
	Gtin14       string `json:"gtin14"`
	Gtin13       string `json:"gtin13"`
	Gtin12       string `json:"gtin12"`
	Availability string `json:"availability"`
	URL          string `json:"url"`
}

type Scraper struct {
	service *services.Service
}

func NewScraper(service *services.Service) *Scraper {
	return &Scraper{service}
}

func (s *Scraper) SaveProductData(websiteID int, url string) error {

	res, err := http.Get(url)
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return err
	}
	res.Body.Close()

	var productDataString string
	doc.Find("script[type='application/ld+json']").Each(func(i int, s *goquery.Selection) {
		script := strings.TrimSpace(s.Text())
		matched, _ := regexp.Match(`"@type":\s?"Product"`, []byte(script))
		if matched {
			reg := regexp.MustCompile(`"gtin1([0-9])":\s+([A-Za-z\d]+)`)
			productDataString = string(reg.ReplaceAll([]byte(script), []byte(`"gtin1$1":"$2"`)))
		}
	})

	if productDataString == "" {
		return fmt.Errorf("no product data for url %s", url)
	}

	containsMultipleOffers, _ := regexp.Match(`"offers"\s?:\s?\[`, []byte(productDataString))
	if containsMultipleOffers {
		var productData ProductDataMultipleOffers
		err = json.Unmarshal([]byte(productDataString), &productData)
		if err != nil {
			return err
		}
		for _, offer := range productData.Offers {
			err = s.UpdateDB(websiteID, url, productData.Name, productData.Brand.Name, productData.Description, productData.Image, offer)
			if err != nil {
				return err
			}
		}
	} else {
		var productData ProductDataSingleOffer
		err = json.Unmarshal([]byte(productDataString), &productData)
		if err != nil {
			fmt.Println(err.Error())
		}
		offer := productData.Offers
		err = s.UpdateDB(websiteID, url, productData.Name, productData.Brand.Name, productData.Description, productData.Image, offer)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Scraper) UpdateDB(websiteID int, webURL string, name string, brand string, description string, image string, offer ProductOffers) error {
	brandID := 0

	if brand != "" {
		brandExists, err := s.service.DoesBrandExist(strings.ToLower(brand))
		if err != nil {
			return err
		}
		//if brand exists get ID else create and return ID
		if brandExists {
			brand, err := s.service.GetBrandByName(strings.ToLower(brand))
			if err != nil {
				return err
			}
			brandID = brand.ID
		} else {
			brand := models.Brand{
				Name: strings.ToLower(brand),
				Path: url.QueryEscape(strings.ToLower(brand)),
			}
			id, err := s.service.InsertBrand(brand)
			if err != nil {
				return err
			}
			brandID = id
		}
	}

	productExists, err := s.service.DoesProductExist(webURL)
	if err != nil {
		return err
	}
	var productID int
	if productExists {
		product, err := s.service.GetProductByURL(webURL)
		if err != nil {
			return err
		}

		productID = product.ProductID

		productUpdates := product
		productUpdates.ProductName = name
		productUpdates.LastCrawled = time.Now()
		productUpdates.Description = description
		productUpdates.Image = image
		productUpdates.BrandID = brandID

		err = s.service.UpdateProduct(productUpdates)
		if err != nil {
			return err
		}
	} else {
		product := models.Product{
			BrandID:     brandID,
			Description: description,
			Image:       image,
			WebsiteID:   websiteID,
			ProductName: name,
			URL:         webURL,
			LastCrawled: time.Now(),
		}
		lastInsertID, err := s.service.CreateProduct(product)
		if err != nil {
			return err
		}
		productID = lastInsertID
	}

	if productID < 1 {
		return fmt.Errorf("expected a product id greater than 0. Instead got: %d", productID)
	}

	price, err := strconv.ParseFloat(offer.Price, 64)
	if err != nil {
		return err
	}

	priceData := models.PriceData{
		ProductID: productID,
		Price:     price,
		Name:      offer.Name,
		SKU:       offer.Sku,
		Gtin12:    offer.Gtin12,
		Gtin13:    offer.Gtin13,
		Gtin14:    offer.Gtin14,
		Image:     offer.Image,
		Timestamp: time.Now(),
		Currency:  "EUR",
	}

	err = s.service.CreatePriceData(priceData)
	if err != nil {
		return err
	}
	return nil
}
