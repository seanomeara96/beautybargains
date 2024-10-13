package main

import (
	"fmt"
	"time"
)

type CouponCode struct {
	Code        string     `json:"code"`
	Description string     `json:"description"`
	ValidUntil  *time.Time `json:"valid_until"`
	WebsiteID   int
}

// CREATE TABLE coupon_codes (
//     code TEXT NOT NULL,
//     description TEXT NOT NULL,
//     valid_until DATETIME,
//     website_id INTEGER
// );

func (s *Service) CreateCouponCode(coupon CouponCode) error {
	if coupon.WebsiteID == 0 {
		return fmt.Errorf("expected a valid website ID got 0 instead")
	}
	_, err := s.db.Exec(`INSERT INTO coupon_codes(
		code,
		description,
		valid_until,
		website_id
	) VALUES (?,?,?,?)`,
		coupon.Code,
		coupon.Description,
		coupon.ValidUntil,
		coupon.WebsiteID,
	)
	if err != nil {
		return err
	}
	return nil
}
