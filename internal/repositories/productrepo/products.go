package productrepo

import (
	"beautybargains/internal/models"
	"database/sql"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{
		db,
	}
}

// Create a new product and insert it into the Products table
func (s *Repository) Create(product models.Product) (int, error) {
	res, err := s.db.Exec("INSERT INTO Products (ProductName, WebsiteID, Description, URL, BrandID, Image) VALUES (?, ?, ?, ?, ?, ?, ?)",
		product.ProductName, product.WebsiteID, product.Description, product.URL, product.BrandID, product.Image)
	if err != nil {
		return 0, err
	}
	_id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	id := int(_id)
	return id, err
}

// Retrieve a product by its ID from the Products table
func (s *Repository) Get(productID int) (models.Product, error) {
	q := `SELECT 
	ProductID, WebsiteID, ProductName, Description, URL, BrandID, Image 
	FROM Products 
	WHERE ProductID = ?`

	var product models.Product
	if err := s.db.QueryRow(q, productID).
		Scan(
			&product.ProductID,
			&product.WebsiteID,
			&product.ProductName,
			&product.Description,
			&product.URL,
			&product.BrandID,
			&product.Image,
		); err != nil {
		return models.Product{}, err
	}

	return product, nil
}

func (s *Repository) CountProducts() (int, error) {
	q := `SELECT count(ProductID) FROM Products`

	var count int
	if err := s.db.QueryRow(q).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Repository) CountProductsByBrand(brandID int) (int, error) {
	q := `SELECT count(ProductID) FROM Products WHERE BrandID = ?`

	var count int
	if err := s.db.QueryRow(q, brandID).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Repository) GetWebsiteProducts(websiteID, limit, offset int) ([]models.Product, error) {
	q := `SELECT 
	ProductID, WebsiteID, ProductName, Description, URL, BrandID, Image, 
	FROM Products 
	WHERE WebsiteID = ? 
	LIMIT ? OFFSET ?`

	var products []models.Product

	rows, err := s.db.Query(q, websiteID, limit, offset)
	if err != nil {
		return products, err
	}
	defer rows.Close()

	for rows.Next() {
		var product models.Product

		if err := rows.Scan(
			&product.ProductID,
			&product.WebsiteID,
			&product.ProductName,
			&product.Description,
			&product.URL,
			&product.BrandID,
			&product.Image,
		); err != nil {
			return products, err
		}

		products = append(products, product)
	}

	return products, nil
}

func (s *Repository) CountByWebsiteID(websiteID int) (int, error) {
	q := `SELECT count(ProductID) FROM Products WHERE WebsiteID = ?`

	var count int
	if err := s.db.QueryRow(q, websiteID).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Repository) GetProducts(limit, offset int) ([]models.Product, error) {
	q := `
	SELECT 
		ProductID, 
		WebsiteID, 
		ProductName, 
		Description, 
		URL, 
		BrandID, 
		Image 
	FROM Products 
	LIMIT ? OFFSET ?`

	products := []models.Product{}
	rows, err := s.db.Query(q, limit, offset)
	if err != nil {
		return products, err
	}
	defer rows.Close()

	for rows.Next() {
		var product models.Product

		if err := rows.Scan(
			&product.ProductID,
			&product.WebsiteID,
			&product.ProductName,
			&product.Description,
			&product.URL,
			&product.BrandID,
			&product.Image,
		); err != nil {
			return products, err
		}

		products = append(products, product)
	}

	return products, nil
}

func (s *Repository) GetProductsByBrand(brandID, limit, offset int) ([]models.Product, error) {
	q := `
	SELECT 
		ProductID, 
		WebsiteID, 
		ProductName, 
		Description, 
		URL, 
		BrandID, 
		Image 
	FROM Products 
	WHERE BrandID = ? 
	LIMIT ? OFFSET ?`

	products := []models.Product{}

	rows, err := s.db.Query(q, brandID, limit, offset)
	if err != nil {
		return products, fmt.Errorf("could not get products for brand id %d => %v", brandID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var product models.Product

		if err := rows.Scan(
			&product.ProductID,
			&product.WebsiteID,
			&product.ProductName,
			&product.Description,
			&product.URL,
			&product.BrandID,
			&product.Image,
		); err != nil {
			return products, fmt.Errorf("could not scan product in getProductsByBrand => %w", err)
		}

		products = append(products, product)
	}

	return products, nil
}

// Update an existing product in the Products table
func (s *Repository) UpdateProduct(product models.Product) error {
	q := `UPDATE Products 
	SET ProductName = ?, 
		WebsiteID = ?, 
		Description = ?, 
		URL = ?, BrandID = ?, 
		Image = ? 
	WHERE 
		ProductID = ?`

	if _, err := s.db.Exec(q,
		product.ProductName,
		product.WebsiteID,
		product.Description,
		product.URL,
		product.BrandID,
		product.Image,
		product.ProductID,
	); err != nil {
		return err
	}

	return nil
}

// Delete a product by its ID from the Products table
func (s *Repository) DeleteProduct(productID int) error {
	_, err := s.db.Exec("DELETE FROM Products WHERE ProductID = ?", productID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Repository) DoesProductExist(url string) (bool, error) {
	stmt, err := s.db.Prepare(`SELECT count(URL) FROM Products WHERE URL = ?`)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	var count int
	err = stmt.QueryRow(url).Scan(&count)
	if err != nil {
		return false, err
	}

	if count > 0 {
		return true, nil
	}

	return false, nil

}

func (s *Repository) CountProductsWithNoPriceDataByWebsiteID(websiteID int) (int, error) {
	q := `SELECT count(*) FROM Products p LEFT JOIN PriceData pd ON p.ProductID = pd.ProductID WHERE pd.ProductID IS NULL AND Error = false AND p.WebsiteID = ?`
	var count int
	err := s.db.QueryRow(q, websiteID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Repository) GetProductsWithNoPriceDataByWebsiteID(websiteID, limit int) ([]models.Product, error) {
	q := `SELECT  p.ProductID, WebsiteID, ProductName, Description, BrandID, URL FROM Products p LEFT JOIN PriceData pd ON p.ProductID = pd.ProductID WHERE pd.ProductID IS NULL AND p.WebsiteID = ? AND Error = false LIMIT 1`
	var products []models.Product
	rows, err := s.db.Query(q, websiteID, limit)
	if err != nil {
		return products, err
	}
	defer rows.Close()

	for rows.Next() {
		var product models.Product
		err = rows.Scan(&product.ProductID, &product.WebsiteID, &product.ProductName, &product.Description, &product.BrandID, &product.URL)
		if err != nil {
			return products, err
		}
		products = append(products, product)
	}

	return products, nil
}

func (s *Repository) SaveProductError(pID int, errored bool, msg string) error {
	_, err := s.db.Exec("UPDATE Products SET Error = ?, ErrorReason = ? WHERE ProductID = ?", errored, msg, pID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Repository) GetProductErrors() ([]models.ProductError, error) {
	q := `SELECT ProductID, WebsiteID, ProductName, Description, brandID, URL, ErrorReason FROM Products WHERE Error = true`
	rows, err := s.db.Query(q)
	if err != nil {
		return []models.ProductError{}, err
	}
	defer rows.Close()

	var productErrors []models.ProductError
	for rows.Next() {
		var productError models.ProductError
		err = rows.Scan(&productError.ProductID, &productError.WebsiteID, &productError.ProductName, &productError.Description, &productError.BrandID, &productError.URL, &productError.ErrorReason)
		if err != nil {
			return []models.ProductError{}, err
		}
		productErrors = append(productErrors, productError)
	}
	return productErrors, nil
}
