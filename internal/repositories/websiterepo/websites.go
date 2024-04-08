package websiterepo

import (
	"beautybargains/internal/models"
	"database/sql"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{
		db,
	}
}

// Create a new website and insert it into the Websites table
func (s *Repository) CreateWebsite(website models.Website) error {
	_, err := s.db.Exec("INSERT INTO Websites (WebsiteName, URL, Country) VALUES (?, ?, ?)",
		website.WebsiteName, website.URL, website.Country)
	if err != nil {
		return err
	}
	return nil
}

// Retrieve a website by its ID from the Websites table
func (s *Repository) GetByID(websiteID int) (*models.Website, error) {
	var website models.Website
	err := s.db.QueryRow("SELECT WebsiteID, WebsiteName, URL, Country FROM Websites WHERE WebsiteID = ?", websiteID).
		Scan(&website.WebsiteID, &website.WebsiteName, &website.URL, &website.Country)
	if err != nil {
		return nil, err
	}
	return &website, nil
}

// Retrieve a website by its ID from the Websites table
func (s *Repository) GetByName(websiteName string) (*models.Website, error) {
	var website models.Website
	err := s.db.QueryRow("SELECT WebsiteID, WebsiteName, URL, Country FROM Websites WHERE LOWER(WebsiteName) = ?", websiteName).
		Scan(&website.WebsiteID, &website.WebsiteName, &website.URL, &website.Country)
	if err != nil {
		return nil, err
	}
	return &website, nil
}
func (s *Repository) CountWebsites() (int, error) {
	q := `SELECT count(WebsiteID) FROM Websites`
	var count int
	err := s.db.QueryRow(q).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Repository) GetAllWebsites(limit, offset int) ([]models.Website, error) {
	websites := []models.Website{}
	rows, err := s.db.Query("SELECT WebsiteID, WebsiteName, URL, Country FROM websites LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return websites, err
	}
	defer rows.Close()

	for rows.Next() {
		website := models.Website{}
		err = rows.Scan(&website.WebsiteID, &website.WebsiteName, &website.URL, &website.Country)
		if err != nil {
			return websites, err
		}
		websites = append(websites, website)
	}

	return websites, nil
}

// Update an existing website in the Websites table
func (s *Repository) UpdateWebsite(website models.Website) error {
	_, err := s.db.Exec("UPDATE Websites SET WebsiteName = ?, URL = ?, Country = ? WHERE WebsiteID = ?",
		website.WebsiteName, website.URL, website.Country, website.WebsiteID)
	if err != nil {
		return err
	}
	return nil
}

// Delete a website by its ID from the Websites table
func (s *Repository) DeleteWebsite(websiteID int) error {
	_, err := s.db.Exec("DELETE FROM Websites WHERE WebsiteID = ?", websiteID)
	if err != nil {
		return err
	}
	return nil
}
