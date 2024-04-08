package hashtagsvc

import (
	"beautybargains/internal/models"
	"beautybargains/internal/repositories/hashtagrepo"
	"beautybargains/internal/repositories/posthashtagrepo"
	"fmt"
)

type Service struct {
	hashtags     *hashtagrepo.Repository
	postHashtags *posthashtagrepo.Repository
}

func New(h *hashtagrepo.Repository, p *posthashtagrepo.Repository) *Service {
	return &Service{h, p}
}

func (s *Service) GetTrending(limit int) ([]models.Trending, error) {
	top, err := s.postHashtags.GetTopByPostCount(limit) // should expect an array like {hashtag, postcount}
	if err != nil {
		return nil, fmt.Errorf("Could not get postHashtags at GetTrending. %v", err)
	}
	var trending []models.Trending
	for _, row := range top {
		hashtag, err := s.hashtags.Get(row.HashtagID)
		if err != nil {
			return nil, fmt.Errorf("Could not get hashtag by id at GetTrending in hashtagsvc. %w", err)
		}
		trending = append(trending, models.Trending{Category: "Topic", Phrase: hashtag.Phrase, PostCount: row.PostCount})
	}
	return trending, nil
}
