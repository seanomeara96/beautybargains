package main

import (
	"errors"
	"fmt"
)

func (s *Service) insertHashtag(h *Hashtag) (lastInsertID int, err error) {
	if h == nil {
		return -1, errors.New("hashtag passed to insert hashtag is nil")
	}

	res, err := s.db.Exec(`INSERT INTO hashtags (phrase) VALUES (?)`, h.Phrase)
	if err != nil {
		return -1, err
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		return -1, err
	}

	return int(lastID), nil
}

func (s *Service) countHashtagsByPhrase(phrase string) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(id) FROM hashtags WHERE phrase = ?`, phrase).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Service) getHashtagIDByPhrase(phrase string) (int, error) {
	q := `SELECT id FROM hashtags WHERE phrase = ?`
	var h int
	err := s.db.QueryRow(q, phrase).Scan(&h)
	if err != nil {
		return 0, err
	}
	return h, nil
}

func (s *Service) getHashtagByID(id int) (*Hashtag, error) {
	q := `SELECT id, phrase from hashtags WHERE id = ?`
	var h Hashtag
	if err := s.db.QueryRow(q, id).Scan(&h.ID, &h.Phrase); err != nil {
		return nil, err
	}
	return &h, nil
}

func (s *Service) getPostIDsByHashtagQuery(hashtagQuery string, postIDs []int) error {
	hashtagID, err := s.getHashtagIDByPhrase(hashtagQuery)
	if err != nil {
		return fmt.Errorf("could not get hashtag id: %w", err)
	}

	postIdRows, err := s.db.Query("SELECT post_id FROM post_hashtags WHERE hashtag_id = ?", hashtagID)
	if err != nil {
		return fmt.Errorf("error getting post_ids: %w", err)
	}
	defer postIdRows.Close()

	for postIdRows.Next() {
		var id int
		if err := postIdRows.Scan(&id); err != nil {
			return fmt.Errorf("error scanning post_id: %w", err)
		}
		postIDs = append(postIDs, id)
	}

	return nil
}
func (s *Service) GetTrendingHashtags() ([]Hashtag, error) {
	rows, err := s.db.Query(`SELECT hashtag_id, count(post_id) FROM post_hashtags GROUP BY hashtag_id ORDER BY count(post_id) DESC LIMIT 5`)
	if err != nil {
		return nil, fmt.Errorf("could not count hashtag mentions: %w", err)
	}
	defer rows.Close()
	type hashtagCount struct {
		HashtagID int
		PostCount int
	}
	top := make([]hashtagCount, 0, 5)
	for rows.Next() {
		var row hashtagCount
		if err := rows.Scan(&row.HashtagID, &row.PostCount); err != nil {
			return nil, err
		}
		top = append(top, row)
	}
	var trendingHashtags []Hashtag
	for _, row := range top {
		hashtag, err := s.getHashtagByID(row.HashtagID)
		if err != nil {
			return nil, fmt.Errorf("could not get hashtag by id: %w", err)
		}
		trendingHashtags = append(trendingHashtags, Hashtag{ID: hashtag.ID, Phrase: hashtag.Phrase})
	}
	return trendingHashtags, nil
}
