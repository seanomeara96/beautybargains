package main

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalCouponCode(t *testing.T) {
	data := `{ "code":"SAVE10", "description":"save €10", "valid_until":"2023-07-13T00:00:00Z" }`

	var c CouponCode

	if err := json.Unmarshal([]byte(data), &c); err != nil {
		t.Fatal(err)
	}

}
