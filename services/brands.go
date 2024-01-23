package services

import (
	"beautybargains/models"
	"errors"
	"fmt"
)

func (s *Service) GetBrands(limit, offset int) ([]models.Brand, error) {
	q := `SELECT id, name, path FROM brands ORDER BY name ASC LIMIT ? OFFSET ?`

	rows, err := s.db.Query(q, limit, offset)
	if err != nil {
		return []models.Brand{}, fmt.Errorf("could not get brands from db %v", err)
	}
	defer rows.Close()

	brands := []models.Brand{}
	for rows.Next() {
		var brand models.Brand
		err = rows.Scan(&brand.ID, &brand.Name, &brand.Path)
		if err != nil {
			return brands, err
		}
		brands = append(brands, brand)
	}

	return brands, nil

}

func (s *Service) CountBrands() (int, error) {
	q := `SELECT count(id) as brand_count FROM Brands`
	var count int
	err := s.db.QueryRow(q).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("could not count brands => %w", err)
	}
	return count, nil
}
func (s *Service) InsertBrand(brand models.Brand) (int, error) {
	q := `INSERT INTO brands(name, path) VALUES (?, ?)`
	res, err := s.db.Exec(q, brand.Name, brand.Path)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (s *Service) DoesBrandExist(brandName string) (bool, error) {
	q := `SELECT count(id) FROM Brands WHERE name = ?`
	var count int
	err := s.db.QueryRow(q, brandName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Service) GetBrandByName(brandName string) (models.Brand, error) {
	q := `SELECT id, name, path FROM brands WHERE name = ?`
	var brand models.Brand
	err := s.db.QueryRow(q, brandName).Scan(&brand.ID, &brand.Name, &brand.Path)
	if err != nil {
		return models.Brand{}, err
	}
	return brand, nil
}
func (s *Service) GetBrandByPath(brandPath string) (models.Brand, error) {
	q := `SELECT id, name, path FROM brands WHERE path = ?`
	var brand models.Brand
	err := s.db.QueryRow(q, brandPath).Scan(&brand.ID, &brand.Name, &brand.Path)
	if err != nil {
		return models.Brand{}, fmt.Errorf("could not get brand for path %s => %w", brandPath, err)
	}
	return brand, nil
}
func (s *Service) UpdateBrand(brand models.Brand) error {
	if brand.ID < 1 {
		return errors.New("Need to supply a brand id to update")
	}
	q := `UPDATE brands SET name = ?, path = ? WHERE id = ?`
	_, err := s.db.Exec(q, brand.Name, brand.Path, brand.ID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) DeleteBrand(brandID int) error {
	if brandID < 1 {
		return errors.New("Need to supply a valid brand id to update")
	}
	q := `DELETE FROM brands WHERE id = ?`
	_, err := s.db.Exec(q, brandID)
	if err != nil {
		return err
	}
	return nil
}
