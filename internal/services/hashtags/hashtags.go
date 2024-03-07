package hashtagservice

import (
	"beautybargains/internal/repositories/hashtags"
	"beautybargains/internal/repositories/posthashtags"
)

type repos struct {
	hashtags     *hashtags.Repository
	posthashtags *posthashtags.Repository
}

type Service struct {
	repos repos
}
