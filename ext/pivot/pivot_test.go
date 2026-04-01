package pivot_test

import (
	"os"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/ext/pivot"
	"github.com/ncruces/go-sqlite3/internal/testcfg"
)

func TestMain(m *testing.M) {
	sqlite3.AutoExtension(pivot.Register)
	os.Exit(m.Run())
}

func TestRegister(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`
		CREATE TABLE r AS
			SELECT 1 id UNION SELECT 2 UNION SELECT 3;

		CREATE TABLE c(
			id   INTEGER PRIMARY KEY,
			name TEXT
		);
		INSERT INTO c (name) VALUES
			('a'),('b'),('c'),('d');

		CREATE TABLE x(
			r_id INT,
			c_id INT,
			val  TEXT
		);
		INSERT INTO x (r_id, c_id, val)
			SELECT r.id, c.id, c.name || r.id
			FROM c, r;				
	`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`
		CREATE VIRTUAL TABLE v_x USING pivot(
			-- rows
			(SELECT id r_id FROM r),
			-- columns
			(SELECT id c_id, name FROM c),
			-- cells
			(SELECT val FROM x WHERE r_id = ?1 AND c_id = ?2)
		)`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT * FROM v_x WHERE rowid <> 0 AND r_id <> 1 ORDER BY rowid, r_id DESC LIMIT 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	if !stmt.Step() {
		t.Fatal(stmt.Err())
	} else if got := stmt.ColumnInt(0); got != 3 {
		t.Errorf("got %d, want 3", got)
	}

	err = db.Exec(`ALTER TABLE v_x RENAME TO v_y`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRegister_errors(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE VIRTUAL TABLE pivot USING pivot()`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`CREATE VIRTUAL TABLE split_date USING pivot(SELECT 1, SELECT 2, SELECT 3)`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`CREATE VIRTUAL TABLE split_date USING pivot((SELECT 1), SELECT 2, SELECT 3)`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`CREATE VIRTUAL TABLE split_date USING pivot((SELECT 1), (SELECT 2), SELECT 3)`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`CREATE VIRTUAL TABLE split_date USING pivot((SELECT 1), (SELECT 1, 2), SELECT 3)`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`CREATE VIRTUAL TABLE split_date USING pivot((SELECT 1), (SELECT 1, 2), (SELECT 3, 4))`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`CREATE VIRTUAL TABLE split_date USING pivot((SELECT 1), (SELECT 1, 2), (SELECT 3))`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}
}

func pretty(cols []string) string {
	var buf strings.Builder
	for i, s := range cols {
		if i != 0 {
			buf.WriteByte(' ')
		}
		for buf.Len()%8 != 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(s)
	}
	return buf.String()
}
