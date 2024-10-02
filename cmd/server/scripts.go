package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

func processHashtags() error {

	/*
		Get all posts. At some point I will have to implement a way to filter for posts
		that have not already been processed
	*/
	posts, err := getPosts(db, getPostParams{})
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
			count, err := countHashtagsByPhrase(db, phrase)
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
				hashtagID, err := getHashtagIDByPhrase(db, phrase)
				if err != nil {
					return err
				}

				q := `SELECT count(*) FROM post_hashtags WHERE post_id = ? AND hashtag_id = ?`

				var count int
				if err := db.QueryRow(q, p.ID, hashtagID).Scan(&count); err != nil {
					return fmt.Errorf("could not count relationships between %d & %d. %v", p.ID, hashtagID, err)
				}

				noRelationShip := count < 1
				if noRelationShip {
					err = insertPostHashtagRelationship(db, p.ID, hashtagID)
					if err != nil {
						return err
					}
				}
			} else {
				newTagID, err := insertHashtag(db, &Hashtag{Phrase: phrase})
				if err != nil {
					return err
				}
				err = insertPostHashtagRelationship(db, p.ID, newTagID)
				if err != nil {
					return err
				}
			}
		}

	}
	return nil
}

func extractOffersFromBanners() error {

	websites := getWebsites(0, 0)

	for _, website := range websites {
		banners, err := extractWebsiteBannerURLs(website)
		if err != nil {
			continue
		}

		uniqueBanners := []BannerData{}
		for _, banner := range banners {

			var bannerCount int
			err := db.QueryRow(`SELECT count(id) FROM posts WHERE src_url = ?`, banner.Src).Scan(&bannerCount)
			if err != nil {
				return fmt.Errorf("error checking existance of banner %v", err)
			}

			bannerExists := bannerCount > 0

			if bannerExists {
				continue
			}

			uniqueBanners = append(uniqueBanners, banner)
		}

		for _, banner := range uniqueBanners {

			if banner.Src == "" {
				continue
			}

			description, err := generateOfferDescription(website.WebsiteName, banner)
			if err != nil {
				return fmt.Errorf(`error getting offer description from chatgpt: %v`, err)
			}

			author := getRandomPersona()

			// I picked 8 randomly for author id
			authorID := author.ID

			_, err = db.Exec(
				"INSERT INTO posts(website_id, src_url, author_id, description, timestamp) VALUES (? , ? , ?, ?, ?)",
				website.WebsiteID, banner.Src, authorID, description, time.Now())
			if err != nil {
				return fmt.Errorf(`error saving banner promotion: %w`, err)
			}
		}

	}

	if err := processHashtags(); err != nil {
		return err
	}
	return nil
}

func process() {
	fmt.Println("start processing")
	if err := extractOffersFromBanners(); err != nil {
		reportErr(err)
	}
	if err := scorePosts(); err != nil {
		reportErr(err)
	}
	fmt.Println("finished processing")
}

func scorePosts() error {
	posts, err := getPosts(db, getPostParams{})
	if err != nil {
		return err
	}

	for i := range posts {
		w, err := getWebsiteByID(posts[i].WebsiteID)
		if err != nil {
			return err
		}

		if posts[i].Score != float64(w.Score) {
			posts[i].Score = float64(w.Score)
			_, err := db.Exec("UPDATE posts SET score = ? WHERE id = ?", posts[i].Score, posts[i].ID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
