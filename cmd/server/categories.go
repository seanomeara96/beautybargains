package main

import (
	"fmt"
)

type Category struct {
	ID       int
	ParentID int
	Name     string
	URL      string
}

// CreateCategory inserts a new category into the database
func (s *Service) CreateCategory(c *Category) error {
	query := `INSERT INTO categories (parent_id, name, url) VALUES (?, ?, ?)`
	result, err := s.db.Exec(query, c.ParentID, c.Name, c.URL)
	if err != nil {
		return fmt.Errorf("error creating category: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("error getting last insert ID: %v", err)
	}
	c.ID = int(id)
	return nil
}

// GetCategory retrieves a category by ID
func (s *Service) GetCategory(id int) (*Category, error) {
	query := `SELECT id, parent_id, name, url FROM categories WHERE id = ?`
	c := &Category{}
	err := s.db.QueryRow(query, id).Scan(&c.ID, &c.ParentID, &c.Name, &c.URL)
	if err != nil {
		return nil, fmt.Errorf("error getting category: %v", err)
	}
	return c, nil
}

// UpdateCategory updates an existing category in the database
func (s *Service) UpdateCategory(c *Category) error {
	query := `UPDATE categories SET parent_id = ?, name = ?, url = ? WHERE id = ?`
	_, err := s.db.Exec(query, c.ParentID, c.Name, c.URL, c.ID)
	if err != nil {
		return fmt.Errorf("error updating category: %v", err)
	}
	return nil
}

// DeleteCategory removes a category from the database
func (s *Service) DeleteCategory(id int) error {
	query := `DELETE FROM categories WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("error deleting category: %v", err)
	}
	return nil
}

// ListCategories retrieves a list of categories with pagination
func (s *Service) GetCategories(limit, offset int) ([]Category, error) {
	query := `SELECT id, parent_id, name, url FROM categories LIMIT ? OFFSET ?`
	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error listing categories: %v", err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.ParentID, &c.Name, &c.URL); err != nil {
			return nil, fmt.Errorf("error scanning category row: %v", err)
		}
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating category rows: %v", err)
	}
	return categories, nil
}
