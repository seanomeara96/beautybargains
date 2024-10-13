package main

import (
	"fmt"
)

// SQLite create table query:
// CREATE TABLE post_categories (
//
//	id INTEGER PRIMARY KEY AUTOINCREMENT,
//	post_id INTEGER,
//	category_id INTEGER,
//	FOREIGN KEY (post_id) REFERENCES posts(id),
//	FOREIGN KEY (category_id) REFERENCES categories(id)
//
// );
type PostCategory struct {
	ID, PostID, CategoryID int
}

func (s *Service) CreatePostCategoryRelationshipByCategoryName(postID int, catName string) error {
	cat, err := s.GetCategoryByName(catName)
	if err != nil {
		return fmt.Errorf("failed to get category by name: %w", err)
	}
	if _, err := s.db.Exec(`INSERT INTO post_categories (
		post_id,
		category_id
	) VALUES (?, ?)`, postID, cat.ID); err != nil {
		return fmt.Errorf("failed to insert post category relationship: %w", err)
	}
	return nil
}
