package main

import (
	"fmt"
	"strings"
	"time"
)

type CouponCode struct {
	ID          int
	Code        string     `json:"code"`
	Description string     `json:"description"`
	ValidUntil  *time.Time `json:"valid_until"`
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
	INSERT INTO
		coupon_codes(
			code,
			description,
			valid_until,
			first_seen,
			website_id
		)
	VALUES
		(?, ?, ?, ?, ?)`,
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

type getCouponParams struct {
	Limit, Offset int
}

func (s *Service) GetCoupons(params getCouponParams) ([]CouponCode, error) {

	var query strings.Builder

	query.WriteString(`
	SELECT
		code,
		description,
		valid_until,
		first_seen,
		website_id
	FROM
		coupon_codes
	ORDER BY
		id DESC`)

	args := []any{}
	if params.Limit > 0 {
		query.WriteString(` LIMIT ?`)
		args = append(args, params.Limit)
	}

	if params.Offset > 0 {
		query.WriteString(` OFFSET ?`)
		args = append(args, params.Offset)
	}

	rows, err := s.db.Query(query.String(), args...)
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
