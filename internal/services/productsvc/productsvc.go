package productsvc

import (
	"beautybargains/internal/models"
	"beautybargains/internal/repositories/brandrepo"
	"beautybargains/internal/repositories/pricedatarepo"
	"beautybargains/internal/repositories/productrepo"
	"fmt"
	"strings"
)

type repos struct {
	products *productrepo.Repository
	brands   *brandrepo.Repository
	prices   *pricedatarepo.Repository
}

type Service struct {
	repos repos
}

func New(products *productrepo.Repository, brands *brandrepo.Repository, prices *pricedatarepo.Repository) *Service {
	return &Service{repos{products: products, brands: brands, prices: prices}}
}

func (s *Service) Count() (int, error) {
	count, err := s.repos.products.CountProducts()
	if err != nil {
		return -1, err
	}
	return count, nil
}

func (s *Service) GetByWebsiteID(id, limit, offset int) ([]models.Product, error) {
	products, err := s.repos.products.GetWebsiteProducts(id, limit, offset)
	if err != nil {
		return []models.Product{}, fmt.Errorf("Could not get products from repo by website id %w", err)
	}
	return products, nil
}

func (s *Service) GetProductsWithBrandDetails(limit, offset int) ([]models.Product, error) {
	products, err := s.repos.products.GetProducts(limit, offset)
	if err != nil {
		return []models.Product{}, err
	}
	seen := []models.Brand{}
	for i, _ := range products {
		found := false
		for _, brand := range seen {
			if brand.ID == products[i].BrandID {
				found = true
				products[i].Brand = brand
				break
			}
		}
		if found {
			continue
		}
		brand, err := s.repos.brands.Get(products[i].BrandID)
		if err != nil {
			return []models.Product{}, err
		}

		seen = append(seen, brand)
	}
	return products, nil
}

func (s *Service) GetByBrandID(id, limit, offset int) ([]models.Product, error) {
	products, err := s.repos.products.GetProductsByBrand(id, limit, offset)
	if err != nil {
		return []models.Product{}, err
	}
	return products, nil
}

func (s *Service) CountByWebsiteID(id int) (int, error) {
	count, err := s.repos.products.CountByWebsiteID(id)
	if err != nil {
		return -1, err
	}
	return count, nil
}

func (s *Service) CountByBrandID(id int) (int, error) {
	count, err := s.repos.products.CountProductsByBrand(id)
	if err != nil {
		return -1, err
	}
	return count, nil
}

func (s *Service) GetErrors() ([]models.ProductError, error) {
	productErrs, err := s.repos.products.GetProductErrors()
	if err != nil {
		return []models.ProductError{}, err
	}
	return productErrs, nil
}

func (s *Service) GetWithPrices(id int) (models.Product, []models.PriceData, error) {
	product, err := s.repos.products.Get(id)
	if err != nil {
		return models.Product{}, []models.PriceData{}, err
	}

	priceData, err := s.repos.prices.GetByProductID(product.ProductID)
	if err != nil {
		return models.Product{}, []models.PriceData{}, err
	}

	return product, priceData, nil
}

func (s *Service) FilterNewProductURLs(urls []string) ([]string, error) {

	for i := range urls {
		urls[i] = strings.TrimSpace(urls[i])
	}

	newURLs := []string{}

	for i := range urls {
		url := urls[i]
		count, err := s.repos.products.CountByURL(url)
		if err != nil {
			return newURLs, err
		}
		productExists := count > 1
		if !productExists {
			newURLs = append(newURLs, url)
		}
	}
	return newURLs, nil
}

func (s *Service) BatchCreate(uniqueURLs []string, websiteID int) ([]models.Product, error) {
	newProducts := []models.Product{}
	for i := range uniqueURLs {
		newProduct := models.NewProduct(websiteID, uniqueURLs[i])
		id, err := s.repos.products.Create(newProduct)
		if err != nil {
			return nil, err
		}
		newProduct.ProductID = id
		newProducts = append(newProducts, newProduct)
	}
	return newProducts, nil
}
