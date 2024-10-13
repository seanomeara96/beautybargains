package main

import (
	"database/sql"
	"fmt"
	"strings"
)

type Brand struct {
	ID    int
	Name  string
	Path  string
	Score float64
}

/* CREATE TABLE brands (
    id INTEGER PRIMARY KEY,
    name TEXT,
    path TEXT,
    score float64 default 0
); */

type getAllBrandsParams struct {
	Limit, Offset int
}

func (s *Service) GetBrands(params getAllBrandsParams) ([]Brand, error) {
	var sb strings.Builder
	sb.WriteString("SELECT id, name, path, score FROM brands")
	if params.Limit != 0 {
		sb.WriteString(" LIMIT ? OFFSET ?")
	}

	var rows *sql.Rows
	var err error
	if params.Limit != 0 {
		rows, err = s.db.Query(sb.String(), params.Limit, params.Offset)
	} else {
		rows, err = s.db.Query(sb.String())
	}

	if err != nil {
		return nil, fmt.Errorf("could not query brands: %w", err)
	}
	defer rows.Close()

	brands := make([]Brand, 0, params.Limit)
	for rows.Next() {
		var brand Brand
		if err := rows.Scan(
			&brand.ID,
			&brand.Name,
			&brand.Path,
			&brand.Score,
		); err != nil {
			return nil, fmt.Errorf("could not scan brand: %w", err)
		}
		brands = append(brands, brand)
	}
	return brands, nil
}
func (s *Service) CreateBrand(brand Brand) error {
	_, err := s.db.Exec(`
	INSERT INTO
		brands (
			name,
			path,
			score
		)
	VALUES
		(?, ?, ?)`,
		brand.Name,
		brand.Path,
		brand.Score,
	)
	if err != nil {
		return fmt.Errorf("could not create brand: %w", err)
	}
	return nil
}

func (s *Service) BrandExists(brandName string) (bool, error) {
	var count int
	err := s.db.QueryRow(`
	SELECT
		COUNT(id)
	FROM
		brands
	WHERE
		name = ?`,
		brandName,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Service) GetBrandByID(id int) (Brand, error) {
	var brand Brand
	err := s.db.QueryRow(`
	SELECT
		id,
		name,
		path,
		score
	FROM
		brands
	WHERE
		id = ?`,
		id,
	).Scan(
		&brand.ID,
		&brand.Name,
		&brand.Path,
		&brand.Score,
	)
	if err != nil {
		return Brand{}, fmt.Errorf("could not get brand: %w", err)
	}
	return brand, nil
}

func (s *Service) GetBrandByName(name string) (Brand, error) {
	var brand Brand
	err := s.db.QueryRow(`
	SELECT
		id,
		name,
		path,
		score
	FROM
		brands
	WHERE
		name = ?`, name).Scan(
		&brand.ID,
		&brand.Name,
		&brand.Path,
		&brand.Score,
	)
	if err != nil {
		return brand, fmt.Errorf("could not get brand: %w", err)
	}
	return brand, nil
}

func (s *Service) UpdateBrand(brand Brand) error {
	_, err := s.db.Exec(`
	UPDATE
		brands
	SET
		name = ?,
		path = ?,
		score = ?
	WHERE
		id = ?`,
		brand.Name,
		brand.Path,
		brand.Score,
		brand.ID,
	)
	if err != nil {
		return fmt.Errorf("could not update brand: %w", err)
	}
	return nil
}

func (s *Service) DeleteBrand(id int) error {
	_, err := s.db.Exec(`
	DELETE FROM
		brands
	WHERE
		id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("could not delete brand: %w", err)
	}
	return nil
}
