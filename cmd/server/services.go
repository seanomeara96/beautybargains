package main

import "database/sql"

type Service struct {
	db *sql.DB

	// category statemants
	// Prepared statements for reusing and improving performance
	createCategoryStmt *sql.Stmt
	getCategoryStmt    *sql.Stmt
	updateCategoryStmt *sql.Stmt
	deleteCategoryStmt *sql.Stmt
}
