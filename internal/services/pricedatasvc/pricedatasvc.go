package pricedatasvc

import (
	"beautybargains/internal/models"
	"beautybargains/internal/repositories/pricedatarepo"
)

type repos struct {
	pricedata *pricedatarepo.Repository
}

type Service struct {
	repos repos
}

func New(pricedata *pricedatarepo.Repository) *Service {
	return &Service{repos{pricedata}}
}

func (s *Service) GetByProductID(productID, priceID int) (models.PriceData, error) {
	pricedata, err := s.repos.pricedata.GetPriceData(productID, priceID)
	if err != nil {
		return models.PriceData{}, err
	}
	return pricedata, nil
}

func (s *Service) GetPriceDrops(limit, offset int) ([]models.PriceChange, error) {
	priceDrops, err := s.repos.pricedata.GetPriceDrops(limit, offset)
	if err != nil {
		return nil, err
	}

	return priceDrops, nil
}
