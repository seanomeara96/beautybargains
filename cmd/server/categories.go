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
	// Use the prepared statement to improve performance
	result, err := s.createCategoryStmt.Exec(c.ParentID, c.Name, c.URL)
	if err != nil {
		return fmt.Errorf("error creating category (ParentID: %d, Name: %s, URL: %s): %v", c.ParentID, c.Name, c.URL, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("error getting last insert ID for category (ParentID: %d, Name: %s, URL: %s): %v", c.ParentID, c.Name, c.URL, err)
	}
	c.ID = int(id)
	return nil
}

func (s *Service) CategoryExists(name string) (bool, error) {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(id) FROM categories WHERE name = ?`, name).Scan(&count); err != nil {
		return false, fmt.Errorf("error checking if category exists (Name: %s): %v", name, err)
	}
	return count > 0, nil
}

// GetCategory retrieves a category by ID
func (s *Service) GetCategory(id int) (*Category, error) {
	c := &Category{}
	// Use the prepared statement to avoid re-parsing the query
	err := s.getCategoryStmt.QueryRow(id).Scan(&c.ID, &c.ParentID, &c.Name, &c.URL)
	if err != nil {
		return nil, fmt.Errorf("error getting category (ID: %d): %v", id, err)
	}
	return c, nil
}

func (s *Service) GetCategoryByName(catName string) (*Category, error) {
	var c Category
	err := s.db.QueryRow(`SELECT
		id,
		parent_id,
		name,
		url
	FROM
		categories
	WHERE
		name = ?`,
		catName,
	).Scan(&c.ID, &c.ParentID, &c.Name, &c.URL)
	if err != nil {
		return nil, fmt.Errorf("error getting category by name (Name: %s): %v", catName, err)
	}
	return &c, nil
}

// UpdateCategory updates an existing category in the database
func (s *Service) UpdateCategory(c *Category) error {
	// Use the prepared statement to improve performance
	_, err := s.updateCategoryStmt.Exec(c.ParentID, c.Name, c.URL, c.ID)
	if err != nil {
		return fmt.Errorf("error updating category (ID: %d, ParentID: %d, Name: %s, URL: %s): %v", c.ID, c.ParentID, c.Name, c.URL, err)
	}
	return nil
}

// DeleteCategory removes a category from the database
func (s *Service) DeleteCategory(id int) error {
	// Use the prepared statement to avoid re-parsing the query
	_, err := s.deleteCategoryStmt.Exec(id)
	if err != nil {
		return fmt.Errorf("error deleting category (ID: %d): %v", id, err)
	}
	return nil
}

// GetCategories retrieves a list of categories with pagination
func (s *Service) GetCategories(limit, offset int) ([]Category, error) {
	// Build the query string dynamically because parameters cannot be used for LIMIT and OFFSET
	query := fmt.Sprintf(`SELECT id, parent_id, name, url FROM categories LIMIT %d OFFSET %d`, limit, offset)
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error listing categories (Limit: %d, Offset: %d): %v", limit, offset, err)
	}
	defer rows.Close()

	// Preallocate the slice with capacity 'limit' to reduce allocations
	categories := make([]Category, 0, limit)
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
