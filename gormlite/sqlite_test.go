package gormlite

import (
	"context"
	"fmt"
	"testing"

	"gorm.io/gorm"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestDialector(t *testing.T) {
	// This is the DSN of the in-memory SQLite database for these tests.
	const InMemoryDSN = "file:testdatabase?mode=memory&cache=shared"

	// Custom connection with a custom function called "my_custom_function".
	db, err := driver.Open(InMemoryDSN, func(ctx context.Context, conn *sqlite3.Conn) error {
		return conn.CreateFunction("my_custom_function", 0, sqlite3.DETERMINISTIC,
			func(ctx sqlite3.Context, arg ...sqlite3.Value) {
				ctx.ResultText("my-result")
			})
	})
	if err != nil {
		t.Fatal(err)
	}

	rows := []struct {
		description  string
		dialector    gorm.Dialector
		openSuccess  bool
		query        string
		querySuccess bool
	}{
		{
			description:  "Default driver",
			dialector:    Open(InMemoryDSN),
			openSuccess:  true,
			query:        "SELECT 1",
			querySuccess: true,
		},
		{
			description:  "Custom function",
			dialector:    Open(InMemoryDSN),
			openSuccess:  true,
			query:        "SELECT my_custom_function()",
			querySuccess: false,
		},
		{
			description:  "Custom connection",
			dialector:    OpenDB(db),
			openSuccess:  true,
			query:        "SELECT 1",
			querySuccess: true,
		},
		{
			description:  "Custom connection, custom function",
			dialector:    OpenDB(db),
			openSuccess:  true,
			query:        "SELECT my_custom_function()",
			querySuccess: true,
		},
	}
	for rowIndex, row := range rows {
		t.Run(fmt.Sprintf("%d/%s", rowIndex, row.description), func(t *testing.T) {
			db, err := gorm.Open(row.dialector, &gorm.Config{})
			if !row.openSuccess {
				if err == nil {
					t.Errorf("Expected Open to fail.")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected Open to succeed; got error: %v", err)
			}
			if db == nil {
				t.Errorf("Expected db to be non-nil.")
			}
			if row.query != "" {
				err = db.Exec(row.query).Error
				if !row.querySuccess {
					if err == nil {
						t.Errorf("Expected query to fail.")
					}
					return
				}

				if err != nil {
					t.Errorf("Expected query to succeed; got error: %v", err)
				}
			}
		})
	}
}
