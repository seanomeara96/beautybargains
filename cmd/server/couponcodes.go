package main

import (
	"database/sql"
	"fmt"
	"time"
)

type CouponCode struct {
	ID          int
	Code        string       `json:"code"`
	Description string       `json:"description"`
	ValidUntil  sql.NullTime `json:"valid_until"`
	FirstSeen   time.Time
	WebsiteID   int
}

// CREATE TABLE coupon_codes (
// 	   id INTEGER PRIMARY KEY AUTOINCREMENT,
//     code TEXT NOT NULL,
//     description TEXT NOT NULL,
//     valid_until DATETIME,
// 	   first_seen DATETIME,
//     website_id INTEGER
// );

func (s *Service) CreateCouponCode(coupon CouponCode) error {
	if coupon.WebsiteID == 0 {
		return fmt.Errorf("expected a valid website ID got 0 instead")
	}
	_, err := s.db.Exec(`
	INSERT INTO coupon_codes(
		code,
		description,
		valid_until,
		first_seen,
		website_id
	) VALUES (?,?,?,?,?)`,
		coupon.Code,
		coupon.Description,
		coupon.ValidUntil,
		time.Now(),
		coupon.WebsiteID,
	)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) GetCoupons() ([]CouponCode, error) {
	rows, err := s.db.Query(`SELECT
		code,
		description,
		valid_until,
		first_seen,
		website_id
	FROM coupon_codes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var coupons []CouponCode
	for rows.Next() {
		var coupon CouponCode
		err := rows.Scan(
			&coupon.Code,
			&coupon.Description,
			&coupon.ValidUntil,
			&coupon.FirstSeen,
			&coupon.WebsiteID,
		)
		if err != nil {
			return nil, err
		}
		coupons = append(coupons, coupon)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return coupons, nil
}
