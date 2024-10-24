package main

type PostHashtag struct {
	ID        int
	PostID    int
	HashtagID int
}

/* posthashtag db funcs*/
func (s *Service) insertPostHashtagRelationship(postID, hashtagID int) error {

	_, err := s.db.Exec("INSERT INTO post_hashtags(post_id, hashtag_id) VALUES (?, ?)", postID, hashtagID)
	if err != nil {
		return err
	}
	return nil
}
