package pivot_test

import (
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/pivot"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
)

// https://antonz.org/sqlite-pivot-table/
func Example() {
	sqlite3.AutoExtension(pivot.Register)

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`
		CREATE TABLE sales(product TEXT, year INT, income DECIMAL);
		INSERT INTO  sales(product, year, income) VALUES
			('alpha', 2020, 100),
			('alpha', 2021, 120),
			('alpha', 2022, 130),
			('alpha', 2023, 140),
			('beta',  2020, 10),
			('beta',  2021, 20),
			('beta',  2022, 40),
			('beta',  2023, 80),
			('gamma', 2020, 80),
			('gamma', 2021, 75),
			('gamma', 2022, 78),
			('gamma', 2023, 80);
	`)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Exec(`
		CREATE VIRTUAL TABLE v_sales USING pivot(
			-- rows
			(SELECT DISTINCT product FROM sales),
			-- columns
			(SELECT DISTINCT year, year FROM sales),
			-- cells
			(SELECT sum(income) FROM sales WHERE product = ? AND year = ?)
		)`)
	if err != nil {
		log.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT * FROM v_sales`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	cols := make([]string, stmt.ColumnCount())
	for i := range cols {
		cols[i] = stmt.ColumnName(i)
	}
	fmt.Println(pretty(cols))
	for stmt.Step() {
		for i := range cols {
			cols[i] = stmt.ColumnText(i)
		}
		fmt.Println(pretty(cols))
	}
	if err := stmt.Reset(); err != nil {
		log.Fatal(err)
	}

	// Output:
	// product 2020    2021    2022    2023
	// alpha   100     120     130     140
	// beta    10      20      40      80
	// gamma   80      75      78      80
}

func TestMain(m *testing.M) {
	sqlite3.AutoExtension(pivot.Register)
	m.Run()
}

func TestRegister(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
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

	if stmt.Step() {
		if got := stmt.ColumnInt(0); got != 3 {
			t.Errorf("got %d, want 3", got)
		}
	}

	err = db.Exec(`ALTER TABLE v_x RENAME TO v_y`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRegister_errors(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
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
