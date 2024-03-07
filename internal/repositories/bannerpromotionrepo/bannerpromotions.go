package bannerpromotionrepo

import (
	"beautybargains/internal/models"
	"database/sql"
	"time"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db}
}

// TODO update these functions to addres websiteID column
func (s *Repository) UpdateBannerPromotion(p models.BannerPromotion) (*models.BannerPromotion, error) {
	q := `UPDATE banner_promotions SET author_id = ?, bannerURL = ?, websiteID = ?, description = ?, link = ? WHERE id = ?`
	_, err := s.db.Exec(q, p.AuthorID, p.BannerURL, p.WebsiteID, p.Description, p.Link, p.ID)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

type GetBannerPromotionsParams struct {
	WebsiteID           int
	SortByTimestampDesc bool
	Hashtag             string
}

func (s *Repository) GetBannerPromotions(params GetBannerPromotionsParams) ([]models.BannerPromotion, error) {
	promos := []models.BannerPromotion{}

	q := `SELECT id, websiteID, bannerURL, author_id, description, timestamp FROM banner_promotions`

	args := []any{}
	if params.WebsiteID != 0 {
		q += " WHERE websiteID  = ?"
		args = append(args, params.WebsiteID)
	}

	if params.SortByTimestampDesc {
		q += " ORDER BY timestamp DESC"
	}

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return promos, err
	}
	defer rows.Close()

	for rows.Next() {
		var promo models.BannerPromotion
		err = rows.Scan(&promo.ID, &promo.WebsiteID, &promo.BannerURL, &promo.AuthorID, &promo.Description, &promo.Timestamp)
		if err != nil {
			return promos, err
		}
		promos = append(promos, promo)
	}

	return promos, nil

}

func (s *Repository) DoesBannerPromotionExist(imgSrc string) (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT count(id) FROM banner_promotions WHERE bannerURL = ?`, imgSrc).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Repository) SaveBannerPromotion(websiteID int, url string, authorID int, description string, timestamp time.Time) error {
	_, err := s.db.Exec(
		"INSERT INTO banner_promotions(websiteID, bannerURL, author_id, description, timestamp) VALUES (? , ? , ?, ?, ?)",
		websiteID, url, authorID, description, time.Now())

	if err != nil {
		return err
	}

	return nil
}
