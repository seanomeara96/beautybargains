package main

import (
	"fmt"
	"testing"
)

func TestMain(t *testing.T) {
	bf, _ := getWebsiteByID(2)
	banners, err := extractWebsiteBannerURLs(bf)
	if err != nil {
		t.Error(err)
		return
	}
	for _, b := range banners {
		fmt.Println("b.Href", b.Href)
		fmt.Println("b.Src", b.Src)
		fmt.Println("b.SupportingText", b.SupportingText)
	}
	t.Error()
}
