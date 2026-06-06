package rtree_test

import (
	"fmt"
	"log"

	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/ext/rtree"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

func Example_rtree() {
	db, err := driver.Open("file:/test.db?vfs=memdb", rtree.Register)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE VIRTUAL TABLE locations USING rtree(id, minX, maxX, minY, maxY);
		INSERT INTO locations VALUES (1, -10.0, -5.0, 30.0, 35.0);
		INSERT INTO locations VALUES (2, 100.0, 105.0, 0.0, 5.0);
	`)
	if err != nil {
		log.Fatal(err)
	}

	var id int
	err = db.QueryRow(`SELECT id FROM locations WHERE 
		minX >= -12.0 AND maxX <= -4.0 AND 
		minY >= 28.0 AND maxY <= 36.0`).Scan(&id)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(id)
	// Output: 1
}

func Example_geopoly() {
	db, err := driver.Open("file:/test.db?vfs=memdb", rtree.Register)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE VIRTUAL TABLE areas USING geopoly(name);
		INSERT INTO areas(name, _shape) VALUES ('Triangle Area', '[[0,0],[2,0],[1,2],[0,0]]');
	`)
	if err != nil {
		log.Fatal(err)
	}

	var name string
	err = db.QueryRow("SELECT name FROM areas WHERE geopoly_contains_point(_shape, 1.0, 1.0)").Scan(&name)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(name)
	// Output: Triangle Area
}
