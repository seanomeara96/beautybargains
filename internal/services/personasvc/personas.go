package personasvc

import (
	"beautybargains/internal/models"
	"beautybargains/internal/repositories/personarepo"
	"fmt"
)

type repos struct {
	personas *personarepo.Repository
}

type Service struct {
	repos repos
}

func New(personas *personarepo.Repository) *Service {
	return &Service{
		repos{personas},
	}
}

func (s *Service) GetAll() ([]*models.Persona, error) {
	res, err := s.repos.personas.GetAll()
	if err != nil {
		return nil, fmt.Errorf("Issue calling persona repo for all personas. %w", err)
	}
	return res, nil
}
