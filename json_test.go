package sqlite3_test

import (
	"fmt"
	"log"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

func Example_json() {
	db, err := sqlite3.Open("file:/test.db?vfs=memdb")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`
		CREATE TABLE orders (
			cart_id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			cart    BLOB
		) STRICT;
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

	stmt, _, err := db.Prepare(`INSERT INTO orders (user_id, cart) VALUES (?, jsonb(?))`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	if err := stmt.BindInt(1, 123); err != nil {
		log.Fatal(err)
	}

	if err := stmt.BindJSON(2, Cart{
		[]CartItem{
			{ItemID: "111", Name: "T-shirt", Quantity: 1, Price: 250},
			{ItemID: "222", Name: "Trousers", Quantity: 1, Price: 600},
		},
	}); err != nil {
		log.Fatal(err)
	}

	if err := stmt.Exec(); err != nil {
		log.Fatal(err)
	}

	sl1, _, err := db.Prepare(`
		SELECT total(json_each.value -> 'price')
		FROM orders, json_each(cart -> 'items')
		WHERE cart_id = last_insert_rowid()
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer sl1.Close()

	for sl1.Step() {
		fmt.Println("total:", sl1.ColumnInt(0))
	}

	if err := sl1.Err(); err != nil {
		log.Fatal(err)
	}

	sl2, _, err := db.Prepare(`
		SELECT json(cart)
		FROM orders
		WHERE cart_id = last_insert_rowid()
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer sl2.Close()

	for sl2.Step() {
		var cart Cart
		if err := sl2.ColumnJSON(0, &cart); err != nil {
			log.Fatal(err)
		}
		for _, item := range cart.Items {
			fmt.Printf("id: %s, name: %s, quantity: %d, price: %d\n",
				item.ItemID, item.Name, item.Quantity, item.Price)
		}
	}

	if err := sl2.Err(); err != nil {
		log.Fatal(err)
	}

	// Output:
	// total: 850
	// id: 111, name: T-shirt, quantity: 1, price: 250
	// id: 222, name: Trousers, quantity: 1, price: 600
}
