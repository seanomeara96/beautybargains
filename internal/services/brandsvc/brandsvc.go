package brandsvc

import (
	"beautybargains/internal/models"
	"beautybargains/internal/repositories/brandrepo"
	"fmt"
)

type repos struct {
	brands *brandrepo.Repository
}

type Service struct {
	repos repos
}

func New(b *brandrepo.Repository) *Service {
	return &Service{repos{b}}
}
func (s *Service) GetAll(limit, offset int) ([]models.Brand, error) {
	brands, err := s.repos.brands.GetBrands(limit, offset)
	if err != nil {
		return nil, err
	}
	return brands, nil
}

func (s *Service) Count() (int, error) {
	count, err := s.repos.brands.Count()
	if err != nil {
		return -1, fmt.Errorf("Could not ge brand count from repo. %w", err)
	}
	return count, nil
}

func (s *Service) GetByPath(path string) (models.Brand, error) {
	brand, err := s.repos.brands.GetBrandByPath(path)
	if err != nil {
		return models.Brand{}, err
	}
	return brand, nil
}
