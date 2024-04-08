package main

import (
	"fmt"
	"testing"
	"time"
)

func TestTimestampDuration(t *testing.T) {

	time1 := time.Now().Add(-13 * 24 * time.Hour)

	fmt.Printf(time1.String())

	isMoreThan12DaysOld := MoreThanNDaysOld(12, time1)

	if !isMoreThan12DaysOld {
		t.Error("time 1 should be more than 12 days old")
	}

	time2 := time.Now().Add(-11 * 24 * time.Hour)

	isMoreThan12DaysOld = MoreThanNDaysOld(12, time2)

	if isMoreThan12DaysOld {
		t.Error("time 2 should not be older than 12 days")
	}

}

func TestRoundUpToInt(t *testing.T) {
	res := RoundUpToInt(11.5)
	if res != 12 {
		t.Error("Expected 12")
	}
	res = RoundUpToInt(11.25)
	if res != 12 {
		t.Error("Expected 12")
	}
	res = RoundUpToInt(11.75)
	if res != 12 {
		t.Error("Expected 12")
	}
}
