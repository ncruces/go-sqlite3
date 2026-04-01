//go:build !js

package pivot_test

import (
	"fmt"
	"log"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/ext/pivot"
)

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
