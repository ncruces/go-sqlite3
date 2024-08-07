//go:build (linux || darwin || windows || freebsd || openbsd || netbsd || dragonfly || illumos || sqlite3_flock) && !sqlite3_nosys

package driver_test

// Adapted from: https://go.dev/doc/tutorial/database-access

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

var db *sql.DB

type Album struct {
	ID     int64
	Title  string
	Artist string
	Price  float32
}

func Example() {
	// Get a database handle.
	var err error
	db, err = sql.Open("sqlite3", "./recordings.db")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove("./recordings.db")
	defer db.Close()

	// Create a table with some data in it.
	err = albumsSetup()
	if err != nil {
		log.Fatal(err)
	}

	albums, err := albumsByArtist("John Coltrane")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Albums found: %v\n", albums)

	// Hard-code ID 2 here to test the query.
	alb, err := albumByID(2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Album found: %v\n", alb)

	albID, err := addAlbum(Album{
		Title:  "The Modern Sound of Betty Carter",
		Artist: "Betty Carter",
		Price:  49.99,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("ID of added album: %v\n", albID)
	// Output:
	// Albums found: [{1 Blue Train John Coltrane 56.99} {2 Giant Steps John Coltrane 63.99}]
	// Album found: {2 Giant Steps John Coltrane 63.99}
	// ID of added album: 5
}

func albumsSetup() error {
	_, err := db.Exec(`
		DROP TABLE IF EXISTS album;
		CREATE TABLE album (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			title      VARCHAR(128) NOT NULL,
			artist     VARCHAR(255) NOT NULL,
			price      DECIMAL(5,2) NOT NULL
		);
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO album
			(title, artist, price)
		VALUES
			('Blue Train', 'John Coltrane', 56.99),
			('Giant Steps', 'John Coltrane', 63.99),
			('Jeru', 'Gerry Mulligan', 17.99),
			('Sarah Vaughan', 'Sarah Vaughan', 34.98)
	`)
	if err != nil {
		return err
	}

	return nil
}

// albumsByArtist queries for albums that have the specified artist name.
func albumsByArtist(name string) ([]Album, error) {
	// An albums slice to hold data from returned rows.
	var albums []Album

	rows, err := db.Query("SELECT * FROM album WHERE artist = ?", name)
	if err != nil {
		return nil, fmt.Errorf("albumsByArtist %q: %w", name, err)
	}
	defer rows.Close()
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var alb Album
		if err := rows.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
			return nil, fmt.Errorf("albumsByArtist %q: %w", name, err)
		}
		albums = append(albums, alb)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("albumsByArtist %q: %w", name, err)
	}
	return albums, nil
}

// albumByID queries for the album with the specified ID.
func albumByID(id int64) (Album, error) {
	// An album to hold data from the returned row.
	var alb Album

	row := db.QueryRow("SELECT * FROM album WHERE id = ?", id)
	if err := row.Scan(&alb.ID, &alb.Title, &alb.Artist, &alb.Price); err != nil {
		if err == sql.ErrNoRows {
			return alb, fmt.Errorf("albumsById %d: no such album", id)
		}
		return alb, fmt.Errorf("albumsById %d: %w", id, err)
	}
	return alb, nil
}

// addAlbum adds the specified album to the database,
// returning the album ID of the new entry
func addAlbum(alb Album) (int64, error) {
	result, err := db.Exec("INSERT INTO album (title, artist, price) VALUES (?, ?, ?)", alb.Title, alb.Artist, alb.Price)
	if err != nil {
		return 0, fmt.Errorf("addAlbum: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("addAlbum: %w", err)
	}
	return id, nil
}

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
	// representation so the driver considers the conversion unsuccesful.
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

var Layout = "2006-01-02T15:04:05.000Z07:00"

type CustomTime struct{ time.Time }

func (c CustomTime) Value() (driver.Value, error) {
	return c.UTC().Format(Layout), nil
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
		t, err := time.Parse(Layout, v)
		if err != nil {
			return err
		}
		*c = CustomTime{t}
	default:
		panic("unsupported value type")
	}
	return nil
}
