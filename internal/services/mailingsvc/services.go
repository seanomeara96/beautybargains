package mailingsvc

import (
	"beautybargains/internal/repositories/subscriberrepo"
	"fmt"
)

type repos struct {
	subscribers *subscriberrepo.Repository
}

type Service struct {
	repos repos
}

func New(s *subscriberrepo.Repository) *Service {
	return &Service{repos{s}}
}

func (s *Service) SendVerificationToken(email, token string) error {
	return fmt.Errorf("not yet implemented")
}

func (s *Service) Subscribe(email string) error {
	// TODO add verification and sanitation

	if err := s.repos.subscribers.Subscribe(email); err != nil {
		return err
	}
	return nil
}

func (s *Service) VerifySubscription(token string) error {
	err := s.repos.subscribers.VerifySubscription(token)
	if err != nil {
		return err
	}
	return nil
}
