package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Post struct {
	WebsiteID   int
	ID          int
	Description string
	SrcURL      string
	Link        sql.NullString
	Timestamp   time.Time
	AuthorID    int // supposed to correspond with a persona id
	Score       float64
}

type getPostParams struct {
	WebsiteID     int
	IDs           []int
	Limit         int
	Offset        int
	SortBy        string
	SortAscending bool
}

func (s *Service) getPosts(params getPostParams) ([]Post, error) {

	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT id, website_id, src_url, author_id, description, timestamp FROM posts")

	args := make([]any, 0)
	if params.WebsiteID != 0 || len(params.IDs) > 0 {
		queryBuilder.WriteString(" WHERE ")
		conditions := make([]string, 0)

		if params.WebsiteID != 0 {
			conditions = append(conditions, "website_id = ?")
			args = append(args, params.WebsiteID)
		}

		if len(params.IDs) > 0 {
			placeholders := make([]string, len(params.IDs))
			for i := range placeholders {
				placeholders[i] = "?"
			}
			conditions = append(conditions, fmt.Sprintf("id IN (%s)", strings.Join(placeholders, ",")))
			for _, id := range params.IDs {
				args = append(args, id)
			}
		}

		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}

	if params.SortBy != "" {
		direction := "DESC"
		if params.SortAscending {
			direction = "ASC"
		}
		sortString := fmt.Sprintf(" ORDER BY %s %s", params.SortBy, direction)

		if params.SortBy == "timestamp" {
			queryBuilder.WriteString(sortString)
		}

		if params.SortBy == "score" {
			queryBuilder.WriteString(sortString)
		}
	}

	if params.Limit > 0 {
		queryBuilder.WriteString(" LIMIT ?")
		args = append(args, params.Limit)
	}

	if params.Offset > 0 {
		queryBuilder.WriteString(" OFFSET ?")
		args = append(args, params.Offset)
	}

	rows, err := s.db.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]Post, 0, params.Limit)
	for rows.Next() {
		var post Post
		if err := rows.Scan(
			&post.ID,
			&post.WebsiteID,
			&post.SrcURL,
			&post.AuthorID,
			&post.Description,
			&post.Timestamp,
		); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func (s *Service) GetPreviewPosts(website Website, postIDs []int) ([]Post, error) {
	args := []any{}
	var q strings.Builder
	q.WriteString(`WITH orderedPosts AS (
	SELECT id, description, author_id, Score, src_url, timestamp, website_id
	FROM posts p `)

	if len(postIDs) > 0 {
		q.WriteString(`WHERE p.id IN (`)
		q.WriteString(strings.Repeat("?,", len(postIDs)-1) + "?)")
		for _, id := range postIDs {
			args = append(args, id)
		}
	} else if website.WebsiteID != 0 {
		q.WriteString(`WHERE website_id = ?`)
		args = append(args, website.WebsiteID)
	}

	q.WriteString(` ORDER BY timestamp DESC LIMIT 6)
	SELECT * FROM orderedPosts ORDER BY score DESC LIMIT 6`)

	rows, err := s.db.Query(q.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(
			&post.ID,
			&post.Description,
			&post.AuthorID,
			&post.Score,
			&post.SrcURL,
			&post.Timestamp,
			&post.WebsiteID,
		); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, err
}
