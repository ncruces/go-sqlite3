package litestream_test

import (
	"log"
	"log/slog"
	"time"

	"github.com/benbjohnson/litestream/s3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/litestream"
)

func ExampleNewReplica() {
	client := s3.NewReplicaClient()
	client.Bucket = "test-bucket"
	client.Path = "fruits.db"

	litestream.NewReplica("fruits.db", client, slog.Default())

	db, err := driver.Open("file:fruits.db?vfs=litestream&_poll_interval=5s")
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	for {
		time.Sleep(time.Second)
		rows, err := db.Query("SELECT * FROM fruits")
		if err != nil {
			log.Fatalln(err)
		}

		for rows.Next() {
			var name, color string
			err := rows.Scan(&name, &color)
			if err != nil {
				log.Fatalln(err)
			}
			log.Println(name, color)
		}

		log.Println("===")
		rows.Close()
	}
}
