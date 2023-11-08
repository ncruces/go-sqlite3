package driver_test

import (
	"fmt"
	"log"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

func Example_json() {
	db, err := driver.Open("file:/test.db?vfs=memdb", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE orders (
			cart_id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			cart    TEXT
		);
	`)
	if err != nil {
		log.Fatal(err)
	}

	type CartItem struct {
		ItemID   string `json:"id"`
		Name     string `json:"name"`
		Quantity int    `json:"quantity,omitempty"`
		Price    int    `json:"price,omitempty"`
	}

	type Cart struct {
		Items []CartItem `json:"items"`
	}

	_, err = db.Exec(`INSERT INTO orders (user_id, cart) VALUES (?, ?)`, 123, sqlite3.JSON(Cart{
		[]CartItem{
			{ItemID: "111", Name: "T-shirt", Quantity: 1, Price: 250},
			{ItemID: "222", Name: "Trousers", Quantity: 1, Price: 600},
		},
	}))
	if err != nil {
		log.Fatal(err)
	}

	var total string
	err = db.QueryRow(`
		SELECT total(json_each.value -> 'price')
		FROM orders, json_each(cart -> 'items')
		WHERE cart_id = last_insert_rowid()
	`).Scan(&total)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("total:", total)
	// Output:
	// total: 850
}
