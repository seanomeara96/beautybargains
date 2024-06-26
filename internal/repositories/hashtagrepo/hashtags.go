package hashtagrepo

import (
	"beautybargains/internal/models"
	"database/sql"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db}
}

func (r *Repository) Insert(h *models.Hashtag) (*models.Hashtag, error) {
	res, err := r.db.Exec(`INSERT INTO hashtags (phrase) VALUES (?)`, h.Phrase)
	if err != nil {
		return nil, err
	}
	lastID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	h.ID = int(lastID)
	return h, nil
}

func (r *Repository) CountByPhrase(phrase string) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(id) FROM hashtags WHERE phrase = ?`, phrase).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) GetIDByPhrase(phrase string) (int, error) {
	q := `SELECT id FROM hashtags WHERE phrase = ?`
	var h int
	err := r.db.QueryRow(q, phrase).Scan(&h)
	if err != nil {
		return 0, err
	}
	return h, nil
}
func (r *Repository) Get(id int) (models.Hashtag, error) {
	q := `SELECT id, phrase from hashtags WHERE id = ?`
	var h models.Hashtag
	if err := r.db.QueryRow(q, id).Scan(&h.ID, &h.Phrase); err != nil {
		return h, err
	}
	return h, nil
}
