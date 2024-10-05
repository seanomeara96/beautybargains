package main

import (
	"fmt"
	"strings"
)

type GetTopByPostCountResponse struct {
	HashtagID int
	PostCount int
}

type getPostParams struct {
	WebsiteID     int
	IDs           []int
	Limit         int
	SortBy        string
	SortAscending bool
}

func (s *Service) getPosts(params getPostParams) ([]Post, error) {
	if s.db == nil {
		return nil, errDBNil
	}

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
