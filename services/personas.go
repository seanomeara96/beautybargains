package services

import (
	"beautybargains/models"
	"beautybargains/repositories"
)

type PersonaService struct {
	r *repositories.PersonaRepo
}

func NewPersonaService(r *repositories.PersonaRepo) *PersonaService {
	return &PersonaService{r}
}

func (s *PersonaService) GetAll() ([]*models.Persona, error) {
	res, err := s.r.GetAll()
	if err != nil {
		return nil, err
	}
	return res, nil
}
