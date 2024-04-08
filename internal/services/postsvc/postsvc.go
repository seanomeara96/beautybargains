package postsvc

import (
	"beautybargains/internal/models"
	"beautybargains/internal/repositories/hashtagrepo"
	"beautybargains/internal/repositories/posthashtagrepo"
	"beautybargains/internal/repositories/postrepo"
	"fmt"
)

type repos struct {
	posts        *postrepo.Repository
	hashtags     *hashtagrepo.Repository
	postHashtags *posthashtagrepo.Repository
}

type Service struct {
	repos repos
}

func New(posts *postrepo.Repository, hashtags *hashtagrepo.Repository, postHashtags *posthashtagrepo.Repository) *Service {
	return &Service{repos{posts, hashtags, postHashtags}}
}

type GetPostParams struct {
	WebsiteID           int
	SortByTimestampDesc bool
	Hashtag             string
}

func (s *Service) GetAll(params GetPostParams) ([]models.Post, error) {

	var postIDs []int
	if params.Hashtag != "" {
		hashtagID, err := s.repos.hashtags.GetIDByPhrase(params.Hashtag)
		if err != nil {
			return nil, fmt.Errorf("Could not get hashtag id in get by phrase. %w", err)
		}
		ids, err := s.repos.postHashtags.GetPostIDs(hashtagID)
		if err != nil {
			return nil, fmt.Errorf("Could not get post ids for hashtag id, %d phrase: %s. %w", hashtagID, params.Hashtag, err)
		}
		postIDs = ids
	}

	repoParams := postrepo.GetPostParams{}
	repoParams.IDs = postIDs
	repoParams.SortByTimestampDesc = params.SortByTimestampDesc
	repoParams.WebsiteID = params.WebsiteID
	p, err := s.repos.posts.GetAll(repoParams)
	if err != nil {
		return nil, fmt.Errorf("Error with postrepo GetAll func at postsvc.GetAll. %w", err)
	}
	return p, nil
}
