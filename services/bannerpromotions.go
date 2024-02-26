package services

import (
	"beautybargains/models"
	"time"
)

type GetBannerPromotionsParams struct {
	WebsiteName         string
	SortByTimestampDesc bool
}

func (s *Service) GetBannerPromotions(params GetBannerPromotionsParams) ([]models.BannerPromotion, error) {
	promos := []models.BannerPromotion{}

	q := `SELECT 
		bp.id, bp.bannerURL, bp.author_id, bp.description, bp.Timestamp,
		w.WebsiteID, w.WebsiteName, w.URL, w.Country
		FROM banner_promotions bp INNER JOIN Websites w ON w.WebsiteID = bp.WebsiteID`

	args := []any{}
	if params.WebsiteName != "" {
		q += " WHERE LOWER(w.WebsiteName) = ?"
		args = append(args, params.WebsiteName)
	}

	if params.SortByTimestampDesc {
		q += " ORDER BY Timestamp DESC"
	}

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return promos, err
	}
	defer rows.Close()

	for rows.Next() {
		var promo models.BannerPromotion
		err = rows.Scan(&promo.ID, &promo.BannerURL, &promo.AuthorID, &promo.Description, &promo.Timestamp, &promo.WebsiteID, &promo.WebsiteName, &promo.URL, &promo.Country)
		if err != nil {
			return promos, err
		}
		promos = append(promos, promo)
	}

	return promos, nil

}

func (s *Service) DoesBannerPromotionExist(imgSrc string) (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT count(id) FROM banner_promotions WHERE bannerURL = ?`, imgSrc).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Service) SaveBannerPromotion(websiteID int, url string, authorID int, description string, timestamp time.Time) error {
	_, err := s.db.Exec(
		"INSERT INTO banner_promotions(websiteID, bannerURL, author_id, description, timestamp) VALUES (? , ? , ?, ?, ?)",
		websiteID, url, authorID, description, time.Now())

	if err != nil {
		return err
	}

	return nil
}
