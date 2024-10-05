package main

import (
	"errors"
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
