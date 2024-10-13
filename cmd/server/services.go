package main

import (
	"database/sql"
	"fmt"
)

type Service struct {
	db *sql.DB

	// category statemants
	// Prepared statements for reusing and improving performance
	createCategoryStmt *sql.Stmt
	getCategoryStmt    *sql.Stmt
	updateCategoryStmt *sql.Stmt
	deleteCategoryStmt *sql.Stmt
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
