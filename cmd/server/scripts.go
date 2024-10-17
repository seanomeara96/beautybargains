package main

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/gosimple/slug"
)

func processHashtags(service *Service) error {

	/*
		Get all posts. At some point I will have to implement a way to filter for posts
		that have not already been processed
	*/
	posts, err := service.getPosts(getPostParams{})
	if err != nil {
		return fmt.Errorf("failed to get posts: %w", err)
	}

	// Define a regular expression pattern for hashtags
	pattern := regexp.MustCompile(`#(\w+)`)

	for _, p := range posts {
		// Find all matches in the post
		matches := pattern.FindAllStringSubmatch(p.Description, -1)

		// Extract hashtags from the matches
		for _, match := range matches {
			if len(match) < 2 {
				return fmt.Errorf("invalid hashtag match: %v", match)
			}
			phrase := strings.ToLower(match[1])
			count, err := service.countHashtagsByPhrase(phrase)
			if err != nil {
				return fmt.Errorf("failed to count hashtags for phrase '%s': %w", phrase, err)
			}
			exists := count > 0

			/*
				If the phrase exists we want to check if it has a relationship to this post
				If it does not have a relationship we need to save the relationship
				If the phrase does not exist we need to save the phrase and the relationship.
			*/
			if exists {
				hashtagID, err := service.getHashtagIDByPhrase(phrase)
				if err != nil {
					return fmt.Errorf("failed to get hashtag ID for phrase '%s': %w", phrase, err)
				}

				q := `SELECT count(*) FROM post_hashtags WHERE post_id = ? AND hashtag_id = ?`

				var count int
				if err := service.db.QueryRow(q, p.ID, hashtagID).Scan(&count); err != nil {
					return fmt.Errorf("could not count relationships between post %d & hashtag %d: %w", p.ID, hashtagID, err)
				}

				noRelationShip := count < 1
				if noRelationShip {
					err = service.insertPostHashtagRelationship(p.ID, hashtagID)
					if err != nil {
						return fmt.Errorf("failed to insert post-hashtag relationship for post %d and hashtag %d: %w", p.ID, hashtagID, err)
					}
				}
			} else {
				newTagID, err := service.insertHashtag(&Hashtag{Phrase: phrase})
				if err != nil {
					return fmt.Errorf("failed to insert new hashtag '%s': %w", phrase, err)
				}
				err = service.insertPostHashtagRelationship(p.ID, newTagID)
				if err != nil {
					return fmt.Errorf("failed to insert post-hashtag relationship for post %d and new hashtag %d: %w", p.ID, newTagID, err)
				}
			}
		}

	}
	return nil
}

func extractUniqueBanners(service *Service, website Website) ([]BannerData, error) {
	banners, err := extractWebsiteBannerURLs(website)
	if err != nil {
		return nil, fmt.Errorf("failed to extract banner URLs for website %s: %w", website.WebsiteName, err)
	}

	uniqueBanners := []BannerData{}
	for _, banner := range banners {

		var bannerCount int
		err := service.db.QueryRow(`SELECT count(id) FROM posts WHERE src_url = ?`, banner.Src).Scan(&bannerCount)
		if err != nil {
			return nil, fmt.Errorf("error checking existence of banner %s: %w", banner.Src, err)
		}

		bannerExists := bannerCount > 0

		if bannerExists {
			continue
		}

		uniqueBanners = append(uniqueBanners, banner)
	}

	return uniqueBanners, nil
}

func saveOfferDescriptionAsPost(tx *sql.Tx, website Website, banner BannerData, description string) (int, error) {
	// I picked 8 randomly for author id
	res, err := tx.Exec(
		`INSERT INTO posts(
			website_id,
			src_url,
			author_id,
			description,
			timestamp
		) VALUES (? , ? , ?, ?, ?)`,
		website.WebsiteID,
		banner.Src,
		getRandomPersona().ID,
		description,
		time.Now(),
	)
	if err != nil {
		return -1, fmt.Errorf("error saving banner promotion for website %s: %w", website.WebsiteName, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return -1, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return int(id), nil
}

func extractOffersFromBanners(service *Service) error {
	websites := getWebsites(0, 0)
	for _, website := range websites {
		banners, err := extractUniqueBanners(service, website)
		if err != nil {
			return fmt.Errorf("error extracting unique banners for website %s: %w", website.WebsiteName, err)
		}
		for _, banner := range banners {
			if banner.Src == "" {
				continue
			}

			offer, err := analyzeOffer(website.WebsiteName, banner)
			if err != nil {
				log.Printf("error getting offer description from chatgpt for website %s and banner %s: %w", website.WebsiteName, banner.Src, err)
				continue
			}

			tx, err := service.db.Begin()
			if err != nil {
				return err
			}

			postID, err := saveOfferDescriptionAsPost(tx, website, banner, offer.Description)
			if err != nil {
				tx.Rollback()
				log.Printf("error saving offer description as post for website %s: %w", website.WebsiteName, err)
				continue
			}

			if err := savePostCategories(tx, postID, offer.Categories); err != nil {
				tx.Rollback()
				log.Printf("error saving post categories for post %d: %w", postID, err)
				continue
			}

			if err := savePostBrands(tx, postID, offer.Brands); err != nil {
				tx.Rollback()
				log.Printf("error saving post brands for post %d: %w", postID, err)
				continue
			}

			if err := saveOfferCouponCodes(tx, website, offer.CouponCodes); err != nil {
				tx.Rollback()
				log.Printf("error saving offer coupon codes for website %s: %w", website.WebsiteName, err)
				continue
			}

			if err := tx.Commit(); err != nil {
				return err
			}
		}
	}
	return nil
}

func savePostCategories(tx *sql.Tx, postID int, offerCategories []string) error {
	for _, catName := range offerCategories {

		var count int
		if err := tx.QueryRow(`
		SELECT 
			COUNT(id) 
		FROM 
			categories 
		WHERE 
			name = ?
		`, catName).Scan(&count); err != nil {
			return fmt.Errorf(
				"error checking if category exists (Name: %s): %v",
				catName,
				err,
			)
		}

		exists := count > 0

		if !exists {
			// Use the prepared statement to improve performance
			_, err := tx.Exec(`
			INSERT INTO categories (
				parent_id, 
				name, 
				url,
			) VALUES (?, ?, ?)`,
				0,
				catName,
				slug.Make(catName),
			)
			if err != nil {
				return fmt.Errorf(
					"error creating category (ParentID: %d, Name: %s, URL: %s): %v",
					0,
					catName,
					slug.Make(catName),
					err,
				)
			}
		}

		var c Category
		if err := tx.QueryRow(`
		SELECT
			id,
			parent_id,
			name,
			url
		FROM
			categories
		WHERE
			name = ?`,
			catName,
		).Scan(
			&c.ID,
			&c.ParentID,
			&c.Name,
			&c.URL,
		); err != nil {
			return fmt.Errorf(
				"error getting category by name (Name: %s): %v",
				catName,
				err,
			)
		}

		if _, err := tx.Exec(`
		INSERT INTO post_categories (
			post_id,
			category_id
		) VALUES (?, ?)`,
			postID, c.ID,
		); err != nil {
			return fmt.Errorf(
				"failed to insert post category relationship: %w",
				err,
			)
		}
	}
	return nil
}

func savePostBrands(tx *sql.Tx, postID int, offerBrands []string) error {
	for _, brandName := range offerBrands {
		var count int
		err := tx.QueryRow(`
		SELECT
			COUNT(id)
		FROM
			brands
		WHERE
			name = ?`,
			brandName,
		).Scan(&count)
		if err != nil {
			return err
		}
		exists := count > 0
		if !exists {
			_, err := tx.Exec(`
			INSERT INTO brands (
				name,
				path,
				score )
			VALUES
				(?, ?, ?)`,
				brandName, slug.Make(brandName), 0,
			)
			if err != nil {
				return fmt.Errorf("could not create brand: %w", err)
			}

		}
		var brand Brand
		err = tx.QueryRow(`
		SELECT
			id,
			name,
			path,
			score
		FROM
			brands
		WHERE
			name = ?`, brandName).Scan(
			&brand.ID,
			&brand.Name,
			&brand.Path,
			&brand.Score,
		)

		if err != nil {
			return fmt.Errorf("failed to get brand by name: %w", err)
		}
		_, err = tx.Exec(`INSERT INTO post_brands(
				post_id,
				brand_id
			) VALUES (?, ?)`,
			postID,
			brand.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert into post_brands: %w", err)
		}
	}
	return nil
}

func saveOfferCouponCodes(tx *sql.Tx, website Website, offerCouponCodes []CouponCode) error {
	for _, coupon := range offerCouponCodes {
		coupon.WebsiteID = website.WebsiteID
		if coupon.WebsiteID == 0 {
			return fmt.Errorf("expected a valid website ID got 0 instead")
		}
		_, err := tx.Exec(`
		INSERT INTO coupon_codes(
			code,
			description,
			valid_until,
			first_seen,
			website_id
		) VALUES (?,?,?,?,?)`,
			coupon.Code,
			coupon.Description,
			coupon.ValidUntil,
			time.Now(),
			coupon.WebsiteID,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func scorePosts(service *Service) error {
	posts, err := service.getPosts(getPostParams{})
	if err != nil {
		return fmt.Errorf("error getting posts: %w", err)
	}

	for i := range posts {
		w, err := getWebsiteByID(posts[i].WebsiteID)
		if err != nil {
			return fmt.Errorf("error getting website for post %d: %w", posts[i].ID, err)
		}

		if posts[i].Score != float64(w.Score) {
			posts[i].Score = float64(w.Score)
			_, err := service.db.Exec("UPDATE posts SET score = ? WHERE id = ?", posts[i].Score, posts[i].ID)
			if err != nil {
				return fmt.Errorf("error updating score for post %d: %w", posts[i].ID, err)
			}
		}
	}

	return nil
}
