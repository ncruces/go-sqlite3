package spellfix1_test

import (
	"testing"

	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/ext/spellfix1"
	"github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func Test(t *testing.T) {
	dsn := memdb.TestDB(t)

	ctx := testcfg.Context(t)
	db, err := driver.Open(dsn, spellfix1.Register)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, `
		CREATE TABLE cost1(iLang, cFrom, cTo, iCost);
		INSERT INTO cost1 VALUES
			(0, '', '?',  97),
			(0, '?', '',  98),
			(0, '?', '?', 99),
			(0, 'm', 'n', 50),
			(0, 'n', 'm', 50);
		SELECT editdist3('cost1');
	`)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		s1   string
		s2   string
		want int
	}{
		{"anchor", "amchor", 50},
		{"anchor", "anchoxr", 97},
		{"anchor", "anchorx", 97},
		{"anchor", "anchr", 98},
		{"anchor", "ancho", 98},
		{"anchor", "nchor", 98},
		{"anchor", "anchur", 99},
		{"anchor", "onchor", 99},
		{"anchor", "anchot", 99},
		{"anchor", "omchor", 149},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			var got int
			err := db.QueryRowContext(ctx, `SELECT editdist3(?, ?)`, tt.s1, tt.s2).Scan(&got)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}
