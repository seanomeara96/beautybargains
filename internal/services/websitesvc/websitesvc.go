package websitesvc

import (
	"beautybargains/internal/models"
	"beautybargains/internal/repositories/websiterepo"
	"fmt"
)

type repos struct {
	websites *websiterepo.Repository
}

type Service struct {
	repos repos
}

func New(w *websiterepo.Repository) *Service {
	return &Service{
		repos{w},
	}
}

func (s *Service) Get(id int) (*models.Website, error) {
	website, err := s.repos.websites.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("Could not get website by id. %w", err)
	}
	return website, nil
}

func (s *Service) GetByName(name string) (*models.Website, error) {
	website, err := s.repos.websites.GetByName(name)
	if err != nil {
		return nil, err
	}
	return website, nil
}

func (s *Service) Create(name, url, country string) error {
	website := models.Website{WebsiteName: name, URL: url, Country: country}
	// TODO check by name or ulr to make sure website is unique
	if err := s.repos.websites.CreateWebsite(website); err != nil {
		return err
	}
	return nil
}

func (s *Service) GetAll(limit, offset int) ([]models.Website, error) {
	websites, err := s.repos.websites.GetAllWebsites(limit, offset)
	if err != nil {
		return nil, err
	}
	return websites, nil
}

func (s *Service) Count() (int, error) {
	count, err := s.repos.websites.CountWebsites()
	if err != nil {
		return -1, err
	}

	return count, nil
}
