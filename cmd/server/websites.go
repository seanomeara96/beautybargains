package main

import (
	"fmt"

	"github.com/gosimple/slug"
)

// Website represents a website entry in the Websites table.
type Website struct {
	WebsiteID   int     `json:"website_id"`   // Unique identifier for the website
	WebsiteName string  `json:"website_name"` // Name of the website
	URL         string  `json:"url"`          // URL of the website
	Country     string  `json:"country"`      // Country code of the website
	Score       float64 `json:"score"`        // Rating or score of the website
	Icon        string  `json:"icon"`         // Icon file name (if available)
	Screenshot  string  `json:"screenshot"`   // Screenshot file name
	Path        string  `json:"path"`         // URL-safe slug or path
}

const (
	_ = iota
	BeautyFeatures
	LookFantasticIE
	Millies
	McCauley
	SkinShop
	Cloud10
	BeautySavers
)

// Sample websites data
var websites = []Website{
	{
		WebsiteID:   BeautyFeatures,
		WebsiteName: "BeautyFeatures",
		URL:         "https://www.beautyfeatures.ie",
		Country:     "IE",
		Score:       10,
		Icon:        "https://cdn11.bigcommerce.com/s-63354/product_images/fav_bf.png?t=1712741168",
		Screenshot:  "www.beautyfeatures.ie_.png",
		Path:        slug.Make("BeautyFeatures"),
	},
	{
		WebsiteID:   LookFantasticIE,
		WebsiteName: "LookFantastic",
		URL:         "https://lookfantastic.ie",
		Country:     "IE",
		Score:       8,
		Icon:        "https://www.lookfantastic.ie/ssr-assets/lookfantastic/updated-favicon.png",
		Screenshot:  "www.lookfantastic.ie_.png",
		Path:        slug.Make("LookFantastic"),
	},
	{
		WebsiteID:   Millies,
		WebsiteName: "Millies",
		URL:         "https://millies.ie",
		Country:     "IE",
		Score:       9,
		Icon:        "https://millies.ie/cdn/shop/t/18/assets/favicon.png?v=116056058874015240761621352174",
		Screenshot:  "millies.ie_.png",
		Path:        slug.Make("Millies"),
	},
	{
		WebsiteID:   McCauley,
		WebsiteName: "McCauley Pharmacy",
		URL:         "https://www.mccauley.ie/",
		Country:     "IE",
		Score:       1,
		Icon:        "https://www.mccauley.ie/static/version1731482450/frontend/Uniphar/mccauleys/en_IE/images/favicons/favicon-196x196.png",
		Screenshot:  "www.mccauley.ie_.png",
		Path:        slug.Make("McCauley Pharmacy"),
	},
	{
		WebsiteID:   SkinShop,
		WebsiteName: "Skin Shop",
		URL:         "https://skinshop.ie/",
		Country:     "IE",
		Score:       5,
		Icon:        "https://skinshop.ie/cdn/shop/files/SkinShop_Log0.png",
		Screenshot:  "skinshop.ie_.png",
		Path:        slug.Make("skin shop"),
	},
	{
		WebsiteID:   Cloud10,
		WebsiteName: "Cloud10 Beauty",
		URL:         "https://www.cloud10beauty.com/",
		Country:     "IE",
		Score:       7,
		Icon:        "https://www.cloud10beauty.com/cdn/shop/files/cbjtd-pi83m_32x32_ba7b8fe2-486a-4697-98e6-fa087a4eeb65.webp",
		Screenshot:  "www.cloud10beauty.com__.png",
		Path:        slug.Make("cloud10 beauty"),
	},
	{
		WebsiteID:   BeautySavers,
		WebsiteName: "BeautySavers",
		URL:         "https://www.beautysavers.ie/",
		Country:     "IE",
		Score:       9,
		Icon:        "https://www.beautysavers.ie/favicon.ico",
		Screenshot:  "www.beautysavers.ie_.png",
		Path:        slug.Make("beauty savers"),
	},
}

/* website funcs*/

// Retrieve a website by its ID from the Websites table
func getWebsiteByID(website_id int) (Website, error) {
	for _, website := range getWebsites(0, 0) {
		if website.WebsiteID == website_id {
			return website, nil
		}
	}
	return Website{}, fmt.Errorf("no website with id %d", website_id)
}

func getWebsiteByPath(path string) (Website, error) {
	for _, website := range getWebsites(0, 0) {
		if website.Path == path {
			return website, nil
		}
	}
	return Website{}, fmt.Errorf("no website with name %s", path)
}

func getWebsites(limit, offset int) []Website {
	lenWebsites := len(websites)

	if limit == 0 || limit > lenWebsites {
		limit = lenWebsites
	}

	if offset >= lenWebsites {
		offset = 0
	}

	toReturn := make([]Website, 0, limit)
	for i := offset; i < limit; i++ {
		toReturn = append(toReturn, websites[i])
	}
	return toReturn
}
