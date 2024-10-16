package main

import (
	"fmt"
	"strings"

	"github.com/gosimple/slug"
)

// Website struct matching the Websites table
type Website struct {
	WebsiteID   int     `json:"website_id"`
	WebsiteName string  `json:"website_name"`
	URL         string  `json:"url"`
	Country     string  `json:"country"`
	Score       float64 `json:"score"`
	Screenshot  string  `json:"screenshot"`
	Path        string  `json:"path"`
}

var websites = []Website{
	{1, "BeautyFeatures", "https://www.beautyfeatures.ie", "IE", 10, "www.beautyfeatures.ie_.png", slug.Make("BeautyFeatures")},
	{2, "LookFantastic", "https://lookfantastic.ie", "IE", 8, "www.lookfantastic.ie_.png", slug.Make("LookFantastic")},
	{3, "Millies", "https://millies.ie", "IE", 9, "millies.ie_.png", slug.Make("Millies")},
	{4, "McCauley Pharmacy", "https://www.mccauley.ie/", "IE", 1, "www.mccauley.ie_.png", slug.Make("McCauley Pharmacy")},
	{5, "Skin Shop", "https://skinshop.ie/", "IE", 5, "skinshop.ie_.png", slug.Make("skin shop")},
	{6, "Cloud10 Beauty", "https://www.cloud10beauty.com/", "IE", 7, "www.cloud10beauty.com__.png", slug.Make("cloud10 beauty")},
	{7, "BeautySavers", "https://www.beautysavers.ie/", "IE", 9, "www.beautysavers.ie_.png", slug.Make("beauty savers")},
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

func getWebsiteByName(websiteName string) (Website, error) {
	for _, website := range getWebsites(0, 0) {
		if strings.EqualFold(website.WebsiteName, websiteName) {
			return website, nil
		}
	}
	return Website{}, fmt.Errorf("no website with name %s", websiteName)
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
