package main

import (
	"database/sql"
	"fmt"
)

type getAllBrandsParams struct {
	Limit, Offset int
}

func dbGetBrands(db *sql.DB, params getAllBrandsParams) ([]Brand, error) {
	var q string
	var rows *sql.Rows
	var err error
	if params.Limit == 0 {
		q = `SELECT id, name, path, score FROM brands`
		rows, err = db.Query(q)
	} else {
		q = `SELECT id, name, path, score FROM brands LIMIT ? OFFSET ?`
		rows, err = db.Query(q, params.Limit, params.Offset)
	}

	if err != nil {
		return nil, fmt.Errorf("could not query brands: %w", err)
	}
	defer rows.Close()

	brands := make([]Brand, 0, params.Limit) // Preallocate slice with capacity if limit is supplied
	for rows.Next() {
		var brand Brand
		if err := rows.Scan(&brand.ID, &brand.Name, &brand.Path, &brand.Score); err != nil {
			return nil, fmt.Errorf("could not scan brand: %w", err)
		}
		brands = append(brands, brand)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return brands, nil
}
