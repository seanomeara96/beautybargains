package main

import "github.com/gosimple/slug"

type Category struct {
	ID       int
	ParentID int
	Name     string
	URL      string
}

func getCategories(limit, offset int) []Category {
	var categories = []Category{
		{ID: 1, ParentID: 0, Name: "Haircare"},
		{ID: 2, ParentID: 0, Name: "Skincare"},
		{ID: 3, ParentID: 0, Name: "Makeup"},
		{ID: 4, ParentID: 0, Name: "Fragrance"},
		{ID: 5, ParentID: 0, Name: "Body Care"},
		{ID: 6, ParentID: 0, Name: "Nail Care"},
		{ID: 7, ParentID: 0, Name: "Men's Grooming"},
		{ID: 8, ParentID: 0, Name: "Beauty Tools"},
		{ID: 9, ParentID: 0, Name: "Bath & Shower"},
		{ID: 10, ParentID: 0, Name: "Sun Care"},
		{ID: 11, ParentID: 0, Name: "Oral Care"},
		{ID: 12, ParentID: 0, Name: "Wellness"},
	}

	for i := range categories {
		categories[i].URL = slug.Make(categories[i].Name)
	}

	lenCategories := len(categories)

	if limit == 0 || limit > lenCategories {
		limit = lenCategories
	}

	if offset >= lenCategories {
		offset = 0
	}

	toReturn := make([]Category, 0, limit)
	for i := offset; i < limit; i++ {
		toReturn = append(toReturn, categories[i])
	}
	return toReturn
}
