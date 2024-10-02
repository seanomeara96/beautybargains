package main

import "database/sql"

/* posthashtag db funcs*/

func insertPostHashtagRelationship(db *sql.DB, postID, hashtagID int) error {

	if db == nil {
		return errDBNil
	}

	_, err := db.Exec("INSERT INTO post_hashtags(post_id, hashtag_id) VALUES (?, ?)", postID, hashtagID)
	if err != nil {
		return err
	}
	return nil
}
