package repositories

import (
	"beautybargains/models"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type HashtagsRepo struct {
	db *sql.DB
}

func DefaultHashtagRepoConnection() (*HashtagsRepo, *sql.DB, error) {
	db, err := sql.Open("sqlite", "hashtags.db")
	if err != nil {
		return nil, nil, err
	}
	repo := HashtagsRepo{db}

	return &repo, db, nil
}

func (r *HashtagsRepo) Insert(h *models.Hashtag) (*models.Hashtag, error) {
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

func (r *HashtagsRepo) DoesHashtagExist(phrase string) (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(id) FROM hashtags WHERE phrase = ?`, phrase).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
