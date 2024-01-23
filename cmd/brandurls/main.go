package main

import (
	"beautybargains/services"
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "data")
	if err != nil {
		log.Println(err.Error())
		return
	}
	s := services.NewService(db)

	/*
		edit get products functions so that brand name is still included on product models
		edit getProducts by brand function
	*/

	brands, err := s.GetBrands(10000, 0)
	if err != nil {
		log.Println(err.Error())
		return
	}
	lenBrands := len(brands)
	for i, brand := range brands {
		log.Printf("%d/%d", i, lenBrands)
		products, err := s.GetProductsByBrand(brand.ID, 1000000, 0)
		if err != nil {
			log.Printf("Error: %v", err)
			return
		}
		for _, product := range products {
			_, err := db.Exec("UPDATE Products SET BrandID = ? WHERE ProductID = ?", brand.ID, product.ProductID)
			if err != nil {
				log.Printf("Error: %v", err)
				return
			}
		}
	}
	log.Println("Done")
}
