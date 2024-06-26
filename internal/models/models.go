package models

import "time"

type Mode string

const (
	Dev  Mode = "dev"
	Prod Mode = "prod"
)

// Product struct matching the Products table
type Product struct {
	ProductID   int       `json:"product_id"`
	WebsiteID   int       `json:"website_id"`
	ProductName string    `json:"product_name"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	BrandID     int       `json:"brand_id"`
	Image       string    `json:"image"`
	LastCrawled time.Time `json:"last_crawled"`
	Brand       Brand
}

type ProductError struct {
	Product
	ErrorReason string
}

func NewProduct(websiteID int, url string) Product {
	return Product{
		WebsiteID:   websiteID,
		ProductName: url,
		URL:         url,
		Description: url,
	}
}

// Website struct matching the Websites table
type Website struct {
	WebsiteID   int    `json:"website_id"`
	WebsiteName string `json:"website_name"`
	URL         string `json:"url"`
	Country     string `json:"country"`
}

// PriceData struct matching the PriceData table
type PriceData struct {
	PriceID   int       `json:"price_id"`
	Name      string    `json:"name"`
	ProductID int       `json:"product_id"`
	SKU       string    `json:"sku"`
	Gtin12    string    `json:"gtin12"`
	Gtin13    string    `json:"gtin13"`
	Gtin14    string    `json:"gtin14"`
	Price     float64   `json:"price"`
	Currency  string    `json:"currency"`
	InStock   bool      `json:"in_stock"`
	Timestamp time.Time `json:"timestamp"`
	Image     string    `json:"image"`
}

type PriceAtTime struct {
	Price float64
	Date  time.Time
}

type PriceChange struct {
	ProductID         int
	CurrentPrice      float64
	CurrentTimeStamp  time.Time
	PreviousPrice     float64
	PreviousTimestamp time.Time
}

type ProductWithPrice struct {
	Product
	PriceData PriceData
}

type Brand struct {
	ID   int
	Name string
	Path string
}

type Post struct {
	WebsiteID   int
	ID          int
	Description string
	SrcURL      string
	Link        string
	Timestamp   time.Time
	AuthorID    int // supposed to correspond with a persona id
}

type Hashtag struct {
	ID     int
	Phrase string
}

type PostHashtag struct {
	ID        int
	PostID    int
	HashtagID int
}

type Trending struct {
	Category  string
	Phrase    string
	PostCount int
}

var DummyTrending = []Trending{
	Trending{"Topic", "#this", 2},
	Trending{"Topic", "#that", 2},
	Trending{"Topic", "#the", 2},
	Trending{"Topic", "#other", 2},
	Trending{"Topic", "#thing", 2},
}
