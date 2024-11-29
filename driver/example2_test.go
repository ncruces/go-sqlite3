//go:build linux || darwin || windows || freebsd || openbsd || netbsd || dragonfly || illumos || sqlite3_flock || sqlite3_dotlk

package driver_test

// Adapted from: https://go.dev/doc/tutorial/database-access

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"time"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

func Example_customTime() {
	db, err := sql.Open("sqlite3", "file:/time.db?vfs=memdb")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE data (
			id INTEGER PRIMARY KEY,
			date_time TEXT
		) STRICT;
	`)
	if err != nil {
		log.Fatal(err)
	}

	// This one will be returned as string to [sql.Scanner] because it doesn't
	// pass the driver's round-trip test when it tries to figure out if it's
	// a time. 2009-11-17T20:34:58.650Z goes in, but parsing and formatting
	// it with [time.RFC3338Nano] results in 2009-11-17T20:34:58.65Z. Though
	// the times are identical, the trailing zero is lost in the string
	// representation so the driver considers the conversion unsuccessful.
	c1 := CustomTime{time.Date(
		2009, 11, 17, 20, 34, 58, 650000000, time.UTC)}

	// Store our custom time in the database.
	_, err = db.Exec(`INSERT INTO data (date_time) VALUES(?)`, c1)
	if err != nil {
		log.Fatal(err)
	}

	var strc1 string
	// Retrieve it as a string, the result of Value().
	err = db.QueryRow(`
		SELECT date_time
		FROM data
		WHERE id = last_insert_rowid()
	`).Scan(&strc1)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("in db:", strc1)

	var resc1 CustomTime
	// Retrieve it as our custom time type, going through Scan().
	err = db.QueryRow(`
		SELECT date_time
		FROM data
		WHERE id = last_insert_rowid()
	`).Scan(&resc1)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("custom time:", resc1)

	// This one will be returned as [time.Time] to [sql.Scanner] because it does
	// pass the driver's round-trip test when it tries to figure out if it's
	// a time. 2009-11-17T20:34:58.651Z goes in, and parsing and formatting
	// it with [time.RFC3339Nano] results in 2009-11-17T20:34:58.651Z.
	c2 := CustomTime{time.Date(
		2009, 11, 17, 20, 34, 58, 651000000, time.UTC)}
	// Store our custom time in the database.
	_, err = db.Exec(`INSERT INTO data (date_time) VALUES(?)`, c2)
	if err != nil {
		log.Fatal(err)
	}

	var strc2 string
	// Retrieve it as a string, the result of Value().
	err = db.QueryRow(`
		SELECT date_time
		FROM data
		WHERE id = last_insert_rowid()
	`).Scan(&strc2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("in db:", strc2)

	var resc2 CustomTime
	// Retrieve it as our custom time type, going through Scan().
	err = db.QueryRow(`
		SELECT date_time
		FROM data
		WHERE id = last_insert_rowid()
	`).Scan(&resc2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("custom time:", resc2)
	// Output:
	// in db: 2009-11-17T20:34:58.650Z
	// scan type string: 2009-11-17T20:34:58.650Z
	// custom time: 2009-11-17 20:34:58.65 +0000 UTC
	// in db: 2009-11-17T20:34:58.651Z
	// scan type time: 2009-11-17 20:34:58.651 +0000 UTC
	// custom time: 2009-11-17 20:34:58.651 +0000 UTC
}

type CustomTime struct{ time.Time }

func (c CustomTime) Value() (driver.Value, error) {
	return sqlite3.TimeFormat7TZ.Encode(c.UTC()), nil
}

func (c *CustomTime) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*c = CustomTime{time.Time{}}
	case time.Time:
		fmt.Println("scan type time:", v)
		*c = CustomTime{v}
	case string:
		fmt.Println("scan type string:", v)
		t, err := sqlite3.TimeFormat7TZ.Decode(v)
		if err != nil {
			return err
		}
		*c = CustomTime{t}
	default:
		panic("unsupported value type")
	}
	return nil
}
