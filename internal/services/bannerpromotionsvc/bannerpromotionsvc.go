package bannerpromotionsvc

import (
	"beautybargains/internal/models"
	"beautybargains/internal/repositories/bannerpromotionrepo"
)

type repos struct {
	bannerpromotions *bannerpromotionrepo.Repository
}

type Service struct {
	repos repos
}

func New(bp *bannerpromotionrepo.Repository) *Service {
	return &Service{repos{bp}}
}

func (s *Service) GetAll(params bannerpromotionrepo.GetBannerPromotionsParams) ([]models.BannerPromotion, error) {
	p, err := s.repos.bannerpromotions.GetBannerPromotions(params)
	if err != nil {
		return nil, err
	}
	return p, nil
}
