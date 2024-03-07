package personarepo

import (
	"beautybargains/internal/models"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db}
}

func DefaultRepositoryConnection() (*Repository, *sql.DB, error) {
	db, err := sql.Open("sqlite3", "images.db")
	if err != nil {
		return nil, nil, err
	}
	repo := NewRepository(db)
	return repo, db, nil
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db}
}

// get all
func (r *Repository) GetAll() ([]*models.Persona, error) {
	q := `SELECT id, name, description, profile_photo FROM models`
	rows, err := r.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []*models.Persona
	for rows.Next() {
		var p models.Persona
		err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.ProfilePhoto)
		if err != nil {
			return nil, err
		}
		res = append(res, &p)
	}
	return res, nil
}

// get one
func (r *Repository) Get(id int) (*models.Persona, error) {
	return nil, fmt.Errorf("Not yet implmented")
}

// get on random
func (r *Repository) GetRandom() (*models.Persona, error) {
	q := `SELECT id, name, description, profile_photo FROM models ORDER BY RANDOM() LIMIT 1`
	var p models.Persona
	err := r.db.QueryRow(q).Scan(&p.ID, &p.Name, &p.Description, &p.ProfilePhoto)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving random persona. %w", err)
	}
	return &p, nil
}

// update one
func (r *Repository) Update(p models.Persona) (*models.Persona, error) {
	q := `UPDATE models SET name = ?, description = ?, profile_photo = ? WHERE id = ?`
	_, err := r.db.Exec(q, p.Name, p.Description, p.ProfilePhoto, p.ID)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// delete one
func (r *Repository) Delete(id int) error {
	q := `DELETE FROM models WHERE id = ?`
	_, err := r.db.Exec(q, id)
	if err != nil {
		return err
	}
	return nil
}
