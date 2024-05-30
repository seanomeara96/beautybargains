package scripts

import (
	"beautybargains/internal/models"
	"beautybargains/internal/repositories/hashtagrepo"
	"beautybargains/internal/repositories/posthashtagrepo"
	"beautybargains/internal/repositories/postrepo"
	"database/sql"
	"errors"
	"regexp"
	"strings"
)

func ProcessHashtags() error {

	/*
		Establish DB Connections
	*/
	postDB, err := sql.Open("sqlite3", "data/posts.db")
	if err != nil {
		return err
	}
	defer postDB.Close()
	hashtagDB, err := sql.Open("sqlite3", "data/hashtags.db")
	if err != nil {
		return err
	}
	defer hashtagDB.Close()
	posthashtagDB, err := sql.Open("sqlite3", "data/posthashtags.db")
	if err != nil {
		return err
	}
	defer posthashtagDB.Close()
	/*
		Instantiate repos
	*/
	postRepo := postrepo.New(postDB)
	hashtagRepo := hashtagrepo.New(hashtagDB)
	postHashtagRepo := posthashtagrepo.New(posthashtagDB)

	/*
		Get all posts. At some point I will have to implement a way to filter for posts
		that have not already been processed
	*/
	posts, err := postRepo.GetAll(postrepo.GetPostParams{})
	if err != nil {
		return err
	}

	// Define a regular expression pattern for hashtags
	pattern := regexp.MustCompile(`#(\w+)`)

	for _, p := range posts {
		// Find all matches in the post
		matches := pattern.FindAllStringSubmatch(p.Description, -1)

		// Extract hashtags from the matches
		for _, match := range matches {
			if len(match) < 2 {
				return errors.New("match was less than 2")
			}
			phrase := strings.ToLower(match[1])
			count, err := hashtagRepo.CountByPhrase(phrase)
			if err != nil {
				return err
			}
			exists := count > 0

			/*
				If the phrase exists we want to check if it has a relationship to this post
				If it does not have a relationship we need to save the relationship
				If the phrase does not exist we need to save the phrase and the relationship.
			*/
			if exists {
				hashtagID, err := hashtagRepo.GetIDByPhrase(phrase)
				if err != nil {
					return err
				}

				count, err := postHashtagRepo.CountRelationships(p.ID, hashtagID)
				if err != nil {
					return err
				}

				noRelationShip := count < 1
				if noRelationShip {
					err = postHashtagRepo.Insert(p.ID, hashtagID)
					if err != nil {
						return err
					}
				}
			} else {
				newTag, err := hashtagRepo.Insert(&models.Hashtag{Phrase: phrase})
				if err != nil {
					return err
				}
				err = postHashtagRepo.Insert(p.ID, newTag.ID)
				if err != nil {
					return err
				}
			}
		}

	}
	return nil
}
