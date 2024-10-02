package main

import (
	"database/sql"
	"errors"
)

func insertHashtag(db *sql.DB, h *Hashtag) (lastInsertID int, err error) {
	if db == nil {
		return -1, errDBNil
	}

	if h == nil {
		return -1, errors.New("hashtag passed to insert hashtag is nil")
	}

	res, err := db.Exec(`INSERT INTO hashtags (phrase) VALUES (?)`, h.Phrase)
	if err != nil {
		return -1, err
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		return -1, err
	}

	return int(lastID), nil
}

func countHashtagsByPhrase(db *sql.DB, phrase string) (int, error) {
	if db == nil {
		return -1, errDBNil
	}

	var count int
	err := db.QueryRow(`SELECT COUNT(id) FROM hashtags WHERE phrase = ?`, phrase).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func getHashtagIDByPhrase(db *sql.DB, phrase string) (int, error) {
	if db == nil {
		return -1, errDBNil
	}

	q := `SELECT id FROM hashtags WHERE phrase = ?`
	var h int
	err := db.QueryRow(q, phrase).Scan(&h)
	if err != nil {
		return 0, err
	}
	return h, nil
}

func getHashtagByID(db *sql.DB, id int) (*Hashtag, error) {
	if db == nil {
		return nil, errDBNil
	}

	q := `SELECT id, phrase from hashtags WHERE id = ?`
	var h Hashtag
	if err := db.QueryRow(q, id).Scan(&h.ID, &h.Phrase); err != nil {
		return nil, err
	}
	return &h, nil
}
