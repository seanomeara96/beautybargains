package hashtagservice

import (
	"beautybargains/internal/repositories/hashtagrepo"
	"beautybargains/internal/repositories/posthashtagrepo"
)

type repos struct {
	hashtags     *hashtagrepo.Repository
	posthashtags *posthashtagrepo.Repository
}

type Service struct {
	repos repos
}
