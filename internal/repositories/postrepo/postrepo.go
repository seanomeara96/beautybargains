package postrepo

import (
	"beautybargains/internal/models"
	"database/sql"
	"strings"
	"time"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db}
}

// TODO update these functions to addres websiteID column
func (s *Repository) Update(p models.Post) (*models.Post, error) {
	q := `UPDATE 
		posts 
	SET 
		author_id = ?, 
		srcURL = ?, 
		websiteID = ?, 
		description = ?, 
		link = ? 
	WHERE 
		id = ?`

	if _, err := s.db.Exec(
		q,
		p.AuthorID,
		p.SrcURL,
		p.WebsiteID,
		p.Description,
		p.Link,
		p.ID,
	); err != nil {
		return nil, err
	}

	return &p, nil
}

type GetPostParams struct {
	WebsiteID           int
	SortByTimestampDesc bool
	IDs                 []int
}

func (s *Repository) GetAll(params GetPostParams) ([]models.Post, error) {
	promos := []models.Post{}

	q := `SELECT id, websiteID, srcURL, author_id, description, timestamp FROM posts`

	args := []any{}
	if params.WebsiteID != 0 || len(params.IDs) > 0 {
		subQ := []string{}
		if params.WebsiteID != 0 {
			subQ = append(subQ, "websiteID = ?")
			args = append(args, params.WebsiteID)
		}

		if len(params.IDs) > 0 {
			idQuery := "id IN ("
			idQuery += strings.Repeat("?, ", len(params.IDs)-1)
			idQuery += "?)"
			subQ = append(subQ, idQuery)
			for _, id := range params.IDs {
				args = append(args, id)
			}
		}
		q += " WHERE " + strings.Join(subQ, " AND ")
	}

	if params.SortByTimestampDesc {
		q += " ORDER BY timestamp DESC"
	}

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return promos, err
	}
	defer rows.Close()

	for rows.Next() {
		var promo models.Post
		err = rows.Scan(&promo.ID, &promo.WebsiteID, &promo.SrcURL, &promo.AuthorID, &promo.Description, &promo.Timestamp)
		if err != nil {
			return promos, err
		}
		promos = append(promos, promo)
	}

	return promos, nil

}

func (s *Repository) CountBySrc(imgSrc string) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT count(id) FROM posts WHERE srcURL = ?`, imgSrc).Scan(&count)
	if err != nil {
		return -1, err
	}
	return count, nil
}

func (s *Repository) Insert(websiteID int, url string, authorID int, description string, timestamp time.Time) error {
	_, err := s.db.Exec(
		"INSERT INTO posts(websiteID, srcURL, author_id, description, timestamp) VALUES (? , ? , ?, ?, ?)",
		websiteID, url, authorID, description, time.Now())

	if err != nil {
		return err
	}

	return nil
}
