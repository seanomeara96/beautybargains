package main

import (
	"fmt"
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

func saveOfferDescriptionAsPost(service *Service, website Website, banner BannerData, description string) (int, error) {
	// I picked 8 randomly for author id
	res, err := service.db.Exec(
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
				return fmt.Errorf("error getting offer description from chatgpt for website %s and banner %s: %w", website.WebsiteName, banner.Src, err)
			}

			postID, err := saveOfferDescriptionAsPost(service, website, banner, offer.Description)
			if err != nil {
				return fmt.Errorf("error saving offer description as post for website %s: %w", website.WebsiteName, err)
			}

			if err := savePostCategories(service, postID, offer.Categories); err != nil {
				return fmt.Errorf("error saving post categories for post %d: %w", postID, err)
			}

			if err := savePostBrands(service, postID, offer.Brands); err != nil {
				return fmt.Errorf("error saving post brands for post %d: %w", postID, err)
			}

			if err := saveOfferCouponCodes(service, website, offer.CouponCodes); err != nil {
				return fmt.Errorf("error saving offer coupon codes for website %s: %w", website.WebsiteName, err)
			}
		}

	}

	return nil
}

func savePostCategories(service *Service, postID int, offerCategories []string) error {
	for _, catName := range offerCategories {
		exists, err := service.CategoryExists(catName)
		if err != nil {
			return fmt.Errorf("error checking if category '%s' exists: %w", catName, err)
		}
		if !exists {
			if err := service.CreateCategory(&Category{0, 0, catName, slug.Make(catName)}); err != nil {
				return fmt.Errorf("error creating category '%s': %w", catName, err)
			}
		}
		if err := service.CreatePostCategoryRelationshipByCategoryName(postID, catName); err != nil {
			return fmt.Errorf("error creating post-category relationship for post %d and category '%s': %w", postID, catName, err)
		}
	}
	return nil
}

func savePostBrands(service *Service, postID int, offerBrands []string) error {
	for _, brandName := range offerBrands {
		exists, err := service.BrandExists(brandName)
		if err != nil {
			return fmt.Errorf("error checking if brand '%s' exists: %w", brandName, err)
		}
		if !exists {
			if err := service.CreateBrand(Brand{0, brandName, slug.Make(brandName), 0}); err != nil {
				return fmt.Errorf("error creating brand '%s': %w", brandName, err)
			}
		}
		if err := service.CreatePostBrandRelationshipByBrandName(postID, brandName); err != nil {
			return fmt.Errorf("error creating post-brand relationship for post %d and brand '%s': %w", postID, brandName, err)
		}
	}
	return nil
}

func saveOfferCouponCodes(service *Service, website Website, offerCouponCodes []CouponCode) error {
	for _, coupon := range offerCouponCodes {
		coupon.WebsiteID = website.WebsiteID
		if err := service.CreateCouponCode(coupon); err != nil {
			return fmt.Errorf("error creating coupon code for website %s: %w", website.WebsiteName, err)
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
