package sqlite3_test

import (
	"bytes"
	"fmt"
	"log"
	"regexp"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func ExampleConn_CreateCollation() {
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}

	err = db.Exec(`CREATE TABLE IF NOT EXISTS words (word VARCHAR(10))`)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Exec(`INSERT INTO words (word) VALUES ('côte'), ('cote'), ('coter'), ('coté'), ('cotée'), ('côté')`)
	if err != nil {
		log.Fatal(err)
	}

	err = db.CreateCollation("french", collate.New(language.French).Compare)
	if err != nil {
		log.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT word FROM words ORDER BY word COLLATE french`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for stmt.Step() {
		fmt.Println(stmt.ColumnText(0))
	}
	if err := stmt.Err(); err != nil {
		log.Fatal(err)
	}

	err = stmt.Close()
	if err != nil {
		log.Fatal(err)
	}

	err = db.Close()
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// cote
	// coté
	// côte
	// côté
	// cotée
	// coter
}

func ExampleConn_CreateFunction() {
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}

	err = db.Exec(`CREATE TABLE IF NOT EXISTS words (word VARCHAR(10))`)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Exec(`INSERT INTO words (word) VALUES ('côte'), ('cote'), ('coter'), ('coté'), ('cotée'), ('côté')`)
	if err != nil {
		log.Fatal(err)
	}

	err = db.CreateFunction("upper", 1, sqlite3.DETERMINISTIC|sqlite3.INNOCUOUS, func(ctx sqlite3.Context, arg ...sqlite3.Value) {
		ctx.ResultBlob(bytes.ToUpper(arg[0].RawBlob()))
	})
	if err != nil {
		log.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT upper(word) FROM words`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for stmt.Step() {
		fmt.Println(stmt.ColumnText(0))
	}
	if err := stmt.Err(); err != nil {
		log.Fatal(err)
	}

	err = stmt.Close()
	if err != nil {
		log.Fatal(err)
	}

	err = db.Close()
	if err != nil {
		log.Fatal(err)
	}
	// Unordered output:
	// COTE
	// COTÉ
	// CÔTE
	// CÔTÉ
	// COTÉE
	// COTER
}

func ExampleContext_SetAuxData() {
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}

	err = db.Exec(`CREATE TABLE IF NOT EXISTS words (word VARCHAR(10))`)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Exec(`INSERT INTO words (word) VALUES ('côte'), ('cote'), ('coter'), ('coté'), ('cotée'), ('côté')`)
	if err != nil {
		log.Fatal(err)
	}

	err = db.CreateFunction("regexp", 2, sqlite3.DETERMINISTIC|sqlite3.INNOCUOUS, func(ctx sqlite3.Context, arg ...sqlite3.Value) {
		re, ok := ctx.GetAuxData(0).(*regexp.Regexp)
		if !ok {
			r, err := regexp.Compile(arg[0].Text())
			if err != nil {
				ctx.ResultError(err)
				return
			}
			ctx.SetAuxData(0, r)
			re = r
		}
		ctx.ResultBool(re.Match(arg[1].RawText()))
	})
	if err != nil {
		log.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT word FROM words WHERE word REGEXP '^\p{L}+e$'`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for stmt.Step() {
		fmt.Println(stmt.ColumnText(0))
	}
	if err := stmt.Err(); err != nil {
		log.Fatal(err)
	}

	err = stmt.Close()
	if err != nil {
		log.Fatal(err)
	}

	err = db.Close()
	if err != nil {
		log.Fatal(err)
	}
	// Unordered output:
	// cote
	// côte
	// cotée
}
