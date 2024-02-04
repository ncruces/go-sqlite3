package steampipe_test

import (
	"context"
	"log"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/steampipe"
	"github.com/turbot/steampipe-plugin-hackernews/hackernews"
)

func Example() {
	db, err := driver.Open(":memory:", func(c *sqlite3.Conn) error {
		ctx := context.TODO()
		return steampipe.Register(c, hackernews.Plugin(ctx))
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT * FROM hackernews_show_hn`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		// ...
	}

	// Output:
}
