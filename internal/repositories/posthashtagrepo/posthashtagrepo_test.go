package posthashtagrepo

import (
	"database/sql"
	"math/rand"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestGetTopByPostCount(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Error(err)
		return
	}
	pr := New(db)

	if err := pr.Init(); err != nil {
		t.Errorf("Error init table. %v", err)
		return
	}

	for i := 0; i < 10; i++ {
		if err := pr.Insert(rand.Intn(11), rand.Intn(11)); err != nil {
			t.Errorf("Error inserting dummy data. %v", err)
			return
		}
	}

	res, err := pr.GetTopByPostCount(5)
	if err != nil {
		t.Errorf("Error getting top by post count. %v", err)
		return
	}

	if len(res) != 5 {
		t.Errorf("Expected 5 responses but only received %d.", len(res))
		t.Log(res)
		return
	}

}
