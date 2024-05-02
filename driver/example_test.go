//go:build (linux || darwin || windows || freebsd || illumos) && !sqlite3_nosys

package driver_test

// Adapted from: https://go.dev/doc/tutorial/database-access

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
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
