package main

import (
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"time"
)

/* Event structure is inspred by the event html element in the semantic ui library */

type Event struct {
	ID      int
	Profile Profile
	Content Content
	Meta    EventMeta
}

type Profile struct {
	Photo    string
	Username string
}

type Content struct {
	Summary     template.HTML
	TimeElapsed string
	ExtraImages *[]ExtraImage  // optional
	ExtraText   *template.HTML // optional
}

type ExtraImage struct {
	Src string
	Alt string
}

type EventMeta struct {
	CTALink *string
	Src     *string
	Likes   int
}

func (s *Service) ConvertPostsToEvents(posts []Post) ([]Event, error) {
	events := make([]Event, 0, len(posts))
	for _, post := range posts {
		e, err := s.ConvertPostToEvent(post)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func (s *Service) ConvertPostToEvent(post Post) (Event, error) {
	e := Event{}
	for _, persona := range getPersonas(0, 0) {
		if persona.ID == post.AuthorID {
			e.Profile.Username = persona.Name
			e.Profile.Photo = persona.ProfilePhoto
			break
		}
	}

	timeDiff := time.Since(post.Timestamp)
	hours := int(timeDiff.Hours())
	days := hours / 24

	var unit string
	var magnitude int
	if days > 0 {
		unit, magnitude = "Days", days
	} else {
		unit, magnitude = "Hours", hours
		if hours == 1 {
			unit = "Hour"
		}
	}

	e.Content.TimeElapsed = fmt.Sprintf("%d %s ago", magnitude, unit)
	e.Meta.Src = &post.SrcURL

	if post.Link.Valid {
		e.Meta.CTALink = &post.Link.String
	}

	extraText := post.Description
	pattern := regexp.MustCompile(`#(\w+)`)
	matches := pattern.FindAllStringSubmatch(extraText, -1)

	for _, match := range matches {
		phrase := strings.ToLower(match[1])
		extraText = strings.Replace(extraText, match[0], fmt.Sprintf("<a class='text-blue-500' href='/?hashtag=%s'>%s</a>", phrase, match[0]), 1)
	}

	e.Content.ExtraText = (*template.HTML)(&extraText)
	website, err := getWebsiteByID(post.WebsiteID)
	if err != nil {
		return Event{}, fmt.Errorf("could not get website by id %d: %v", post.WebsiteID, err)
	}
	e.Content.Summary = template.HTML(fmt.Sprintf("posted an update about <a href='%s'>%s</a>", website.URL, website.WebsiteName))
	e.Content.ExtraImages = &[]ExtraImage{{post.SrcURL, ""}}
	return e, nil
}
