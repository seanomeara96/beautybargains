package posthashtagrepo

import (
	"database/sql"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db}
}

func (r *Repository) Insert(postID, hashtagID int) error {
	_, err := r.db.Exec("INSERT INTO post_hashtags(post_id, hashtag_id) VALUES (?, ?)", postID, hashtagID)
	if err != nil {
		return err
	}
	return nil
}

type GetTopByPostCountResponse struct {
	HashtagID int
	PostCount int
}

func (r *Repository) Init() error {
	q := `CREATE TABLE 
	post_hashtags(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id  INTEGER NOT NULL,
		hashtag_id INTEGER NOT NULL
	);`
	if _, err := r.db.Exec(q); err != nil {
		return err
	}
	return nil
}

func (r *Repository) GetTopByPostCount(limit int) ([]GetTopByPostCountResponse, error) {
	q := `SELECT 
		hashtag_id, 
		count(post_id) 
	FROM 
		post_hashtags 
	GROUP BY
		hashtag_id
	ORDER BY 
		count(post_id) 
	DESC 
	LIMIT ?`

	rows, err := r.db.Query(q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := []GetTopByPostCountResponse{}
	for rows.Next() {
		var row GetTopByPostCountResponse
		if err := rows.Scan(&row.HashtagID, &row.PostCount); err != nil {
			return nil, err
		}
		res = append(res, row)
	}
	return res, nil
}

func (r *Repository) CountRelationships(postID, hashtagID int) (int, error) {
	q := `SELECT 
		count(*) 
	FROM 
		post_hashtags 
	WHERE 
		post_id = ? 
	AND 
		hashtag_id = ?`

	var count int
	if err := r.db.QueryRow(q, postID, hashtagID).Scan(&count); err != nil {
		return 0, fmt.Errorf("Could not count relationships between %d & %d. %v", postID, hashtagID, err)
	}
	return count, nil
}

type GetPostsParams struct {
	HashtagID int
}

func (r *Repository) GetPostIDs(hashtagID int) ([]int, error) {
	rows, err := r.db.Query("SELECT post_id FROM post_hashtags WHERE hashtag_id = ?", hashtagID)
	if err != nil {
		return []int{}, fmt.Errorf("Error getting post_ids from post_hashtags db. %w", err)
	}
	defer rows.Close()

	res := []int{}
	for rows.Next() {
		var i int
		err := rows.Scan(&i)
		if err != nil {
			return []int{}, fmt.Errorf("Error scanning post_id in getposts. %w", err)
		}
		res = append(res, i)
	}

	return res, nil
}
