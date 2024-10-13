package main

import "fmt"

type PostBrand struct {
	ID, PostID, BrandID int
}

// CREATE TABLE post_brands (
//     id INTEGER PRIMARY KEY AUTOINCREMENT,
//     post_id INTEGER,
//     brand_id INTEGER,
//     FOREIGN KEY (post_id) REFERENCES posts(id),
//     FOREIGN KEY (brand_id) REFERENCES brands(id)
// );

func (s *Service) CreatePostBrandRelationshipByBrandName(postID int, brandName string) error {
	brand, err := s.GetBrandByName(brandName)
	if err != nil {
		return fmt.Errorf("failed to get brand by name: %w", err)
	}
	_, err = s.db.Exec(`INSERT INTO post_brands(
		post_id,
		brand_id
	) VALUES (?, ?)`,
		postID,
		brand.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to insert into post_brands: %w", err)
	}
	return nil
}
