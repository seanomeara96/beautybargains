package services

import (
	"beautybargains/models"
	"fmt"
	"strings"
)

// Create a new product and insert it into the Products table
func (s *Service) CreateProduct(product models.Product) (int, error) {
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

func (s *Service) CreateNewProducts(uniqueURLs []string, websiteID int) ([]models.Product, error) {
	newProducts := []models.Product{}

	for i := range uniqueURLs {
		newProduct := models.NewProduct(websiteID, uniqueURLs[i])
		newProducts = append(newProducts, newProduct)
	}

	stmt, err := s.db.Prepare("INSERT INTO Products (ProductName, WebsiteID, Description, URL, BrandID, Image) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return newProducts, err
	}
	defer stmt.Close()

	for _, product := range newProducts {
		_, err := stmt.Exec(product.ProductName, product.WebsiteID, product.Description, product.URL, product.BrandID, product.Image)
		if err != nil {
			return []models.Product{}, err
		}
	}

	return newProducts, nil
}

// Retrieve a product by its ID from the Products table
func (s *Service) GetProductByID(productID int) (models.Product, error) {
	var product models.Product
	err := s.db.QueryRow("SELECT ProductID, WebsiteID, ProductName, Description, URL, BrandID, Image FROM Products WHERE ProductID = ?", productID).
		Scan(&product.ProductID, &product.WebsiteID, &product.ProductName, &product.Description, &product.URL, &product.BrandID, &product.Image)
	if err != nil {
		return models.Product{}, err
	}
	return product, nil
}

func (s *Service) GetProductWithPrices(productID int) (models.Product, []models.PriceData, error) {
	product, err := s.GetProductByID(productID)
	if err != nil {
		return models.Product{}, []models.PriceData{}, err
	}
	prices, err := s.GetPriceDataByProductID(product.ProductID)
	if err != nil {
		return models.Product{}, []models.PriceData{}, err
	}
	return product, prices, nil
}
func (s *Service) GetProductByURL(url string) (models.Product, error) {
	var product models.Product
	err := s.db.QueryRow("SELECT ProductID, WebsiteID, ProductName, Description, URL, BrandID, Image FROM Products WHERE URL = ?", url).
		Scan(&product.ProductID, &product.WebsiteID, &product.ProductName, &product.Description, &product.URL, &product.BrandID, &product.Image)
	if err != nil {
		return models.Product{}, err
	}
	return product, nil
}

func (s *Service) CountProducts() (int, error) {
	q := `SELECT count(ProductID) FROM Products`
	var count int
	err := s.db.QueryRow(q).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
func (s *Service) CountProductsByBrand(brandID int) (int, error) {
	q := `SELECT count(ProductID) FROM Products WHERE BrandID = ?`
	var count int
	err := s.db.QueryRow(q, brandID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
func (s *Service) GetWebsiteProducts(websiteID, limit, offset int) ([]models.Product, error) {
	var products []models.Product
	rows, err := s.db.Query("SELECT ProductID, WebsiteID, ProductName, Description, URL, BrandID, Image, b.Name, b.Path FROM Products INNER JOIN brands b on BrandID = b.id WHERE WebsiteID = ? LIMIT ? OFFSET ?", websiteID, limit, offset)
	if err != nil {
		return products, err
	}
	defer rows.Close()

	for rows.Next() {
		var product models.Product
		err := rows.Scan(&product.ProductID, &product.WebsiteID, &product.ProductName, &product.Description, &product.URL, &product.BrandID, &product.Image, &product.Brand.Name, &product.Brand.Path)
		if err != nil {
			return products, err
		}
		products = append(products, product)
	}

	return products, nil
}

func (s *Service) CountWebsiteProducts(websiteID int) (int, error) {
	q := `SELECT count(ProductID) FROM Products WHERE WebsiteID = ?`
	var count int
	err := s.db.QueryRow(q, websiteID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Service) GetProducts(limit, offset int) ([]models.Product, error) {
	products := []models.Product{}
	rows, err := s.db.Query("SELECT ProductID, WebsiteID, ProductName, Description, URL, BrandID, Image FROM Products LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return products, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var product models.Product
		err := rows.Scan(&product.ProductID, &product.WebsiteID, &product.ProductName, &product.Description, &product.URL, &product.BrandID, &product.Image)
		if err != nil {
			return products, err
		}
		products = append(products, product)
		count++
	}
	return products, nil
}

func (s *Service) GetProductsWithBrandDetails(limit, offset int) ([]models.Product, error) {
	products := []models.Product{}
	rows, err := s.db.Query("SELECT ProductID, WebsiteID, ProductName, Description, URL, BrandID, Image, b.Name, b.Path FROM Products INNER JOIN Brands b ON BrandID = b.id LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return products, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var product models.Product
		err := rows.Scan(&product.ProductID, &product.WebsiteID, &product.ProductName, &product.Description, &product.URL, &product.BrandID, &product.Image, &product.Brand.Name, &product.Brand.Path)
		if err != nil {
			return products, err
		}
		products = append(products, product)
		count++
	}
	return products, nil
}

func (s *Service) GetProductsByBrand(brandID, limit, offset int) ([]models.Product, error) {
	products := []models.Product{}
	rows, err := s.db.Query("SELECT ProductID, WebsiteID, ProductName, Description, URL, BrandID, Image FROM Products WHERE BrandID = ? LIMIT ? OFFSET ?", brandID, limit, offset)
	if err != nil {
		return products, fmt.Errorf("could not get products for brand id %d => %v", brandID, err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var product models.Product
		err := rows.Scan(&product.ProductID, &product.WebsiteID, &product.ProductName, &product.Description, &product.URL, &product.BrandID, &product.Image)
		if err != nil {
			return products, fmt.Errorf("could not scan product in getProductsByBrand => %w", err)
		}
		products = append(products, product)
		count++
	}
	return products, nil
}

// Update an existing product in the Products table
func (s *Service) UpdateProduct(product models.Product) error {
	_, err := s.db.Exec("UPDATE Products SET ProductName = ?, WebsiteID = ?, Description = ?, URL = ?, BrandID = ?, Image = ? WHERE ProductID = ?",
		product.ProductName, product.WebsiteID, product.Description, product.URL, product.BrandID, product.Image, product.ProductID)
	if err != nil {
		return err
	}
	return nil
}

// Delete a product by its ID from the Products table
func (s *Service) DeleteProduct(productID int) error {
	_, err := s.db.Exec("DELETE FROM Products WHERE ProductID = ?", productID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) NewProductURLs(urls []string) ([]string, error) {

	for i := range urls {
		urls[i] = strings.TrimSpace(urls[i])
	}

	newURLs := []string{}

	for i := range urls {
		url := urls[i]
		productExists, err := s.DoesProductExist(url)
		if err != nil {
			return newURLs, err
		}
		if !productExists {
			newURLs = append(newURLs, url)
		}
	}
	return newURLs, nil
}

func (s *Service) DoesProductExist(url string) (bool, error) {
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

func (s *Service) CountProductsWithNoPriceDataByWebsiteID(websiteID int) (int, error) {
	q := `SELECT count(*) FROM Products p LEFT JOIN PriceData pd ON p.ProductID = pd.ProductID WHERE pd.ProductID IS NULL AND Error = false AND p.WebsiteID = ?`
	var count int
	err := s.db.QueryRow(q, websiteID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Service) GetProductsWithNoPriceDataByWebsiteID(websiteID, limit int) ([]models.Product, error) {
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

func (s *Service) SaveProductError(pID int, errored bool, msg string) error {
	_, err := s.db.Exec("UPDATE Products SET Error = ?, ErrorReason = ? WHERE ProductID = ?", errored, msg, pID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) GetProductErrors() ([]models.ProductError, error) {
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
