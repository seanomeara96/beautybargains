package posthashtagrepo

import (
	"database/sql"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func (r *Repository) Insert(postID, hashtagID int) error {
	_, err := r.db.Exec("INSERT INTO post_hashtags(post_id, hashtag_id) VALUES (?, ?)", postID, hashtagID)
	if err != nil {
		return err
	}
	return nil
}

type GetPostsParams struct {
	HashtagID int
}

func (r *Repository) GetPosts(params GetPostsParams) ([]int, error) {
	rows, err := r.db.Query("SELECT post_id FROM post_hashtags WHERE hashtag_id = ?", params.HashtagID)
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
