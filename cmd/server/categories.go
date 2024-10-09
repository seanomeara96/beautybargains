package main

import (
	"database/sql"
	"fmt"
)

type Category struct {
	ID       int
	ParentID int
	Name     string
	URL      string
}

// NewService initializes the Service with prepared statements
func NewService(db *sql.DB) (*Service, error) {
	s := &Service{db: db}

	var err error

	// Prepare statements on initialization
	s.createCategoryStmt, err = db.Prepare(`INSERT INTO categories (parent_id, name, url) VALUES (?, ?, ?)`)
	if err != nil {
		return nil, fmt.Errorf("error preparing createCategoryStmt: %v", err)
	}

	s.getCategoryStmt, err = db.Prepare(`SELECT id, parent_id, name, url FROM categories WHERE id = ?`)
	if err != nil {
		return nil, fmt.Errorf("error preparing getCategoryStmt: %v", err)
	}

	s.updateCategoryStmt, err = db.Prepare(`UPDATE categories SET parent_id = ?, name = ?, url = ? WHERE id = ?`)
	if err != nil {
		return nil, fmt.Errorf("error preparing updateCategoryStmt: %v", err)
	}

	s.deleteCategoryStmt, err = db.Prepare(`DELETE FROM categories WHERE id = ?`)
	if err != nil {
		return nil, fmt.Errorf("error preparing deleteCategoryStmt: %v", err)
	}

	return s, nil
}

// Close closes all prepared statements
func (s *Service) Close() error {
	if s.createCategoryStmt != nil {
		s.createCategoryStmt.Close()
	}
	if s.getCategoryStmt != nil {
		s.getCategoryStmt.Close()
	}
	if s.updateCategoryStmt != nil {
		s.updateCategoryStmt.Close()
	}
	if s.deleteCategoryStmt != nil {
		s.deleteCategoryStmt.Close()
	}
	return nil
}

// CreateCategory inserts a new category into the database
func (s *Service) CreateCategory(c *Category) error {
	// Use the prepared statement to improve performance
	result, err := s.createCategoryStmt.Exec(c.ParentID, c.Name, c.URL)
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
	c := &Category{}
	// Use the prepared statement to avoid re-parsing the query
	err := s.getCategoryStmt.QueryRow(id).Scan(&c.ID, &c.ParentID, &c.Name, &c.URL)
	if err != nil {
		return nil, fmt.Errorf("error getting category: %v", err)
	}
	return c, nil
}

// UpdateCategory updates an existing category in the database
func (s *Service) UpdateCategory(c *Category) error {
	// Use the prepared statement to improve performance
	_, err := s.updateCategoryStmt.Exec(c.ParentID, c.Name, c.URL, c.ID)
	if err != nil {
		return fmt.Errorf("error updating category: %v", err)
	}
	return nil
}

// DeleteCategory removes a category from the database
func (s *Service) DeleteCategory(id int) error {
	// Use the prepared statement to avoid re-parsing the query
	_, err := s.deleteCategoryStmt.Exec(id)
	if err != nil {
		return fmt.Errorf("error deleting category: %v", err)
	}
	return nil
}

// GetCategories retrieves a list of categories with pagination
func (s *Service) GetCategories(limit, offset int) ([]Category, error) {
	// Build the query string dynamically because parameters cannot be used for LIMIT and OFFSET
	query := fmt.Sprintf(`SELECT id, parent_id, name, url FROM categories LIMIT %d OFFSET %d`, limit, offset)
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error listing categories: %v", err)
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
