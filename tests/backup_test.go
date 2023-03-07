package tests

import (
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestBackup(t *testing.T) {
	t.Parallel()

	backupName := filepath.Join(t.TempDir(), "backup.db")

	func() { // Create backup.
		db, err := sqlite3.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		err = db.Exec(`CREATE TABLE IF NOT EXISTS users (id INT, name VARCHAR(10))`)
		if err != nil {
			t.Fatal(err)
		}

		err = db.Exec(`INSERT INTO users (id, name) VALUES (0, 'go'), (1, 'zig'), (2, 'whatever')`)
		if err != nil {
			t.Fatal(err)
		}

		err = db.Backup("main", backupName)
		if err != nil {
			t.Fatal(err)
		}

		err = db.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	func() { // Restore backup.
		db, err := sqlite3.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		err = db.Restore("main", backupName)
		if err != nil {
			t.Fatal(err)
		}

		stmt, _, err := db.Prepare(`SELECT id, name FROM users`)
		if err != nil {
			t.Fatal(err)
		}
		defer stmt.Close()

		row := 0
		ids := []int{0, 1, 2}
		names := []string{"go", "zig", "whatever"}
		for ; stmt.Step(); row++ {
			id := stmt.ColumnInt(0)
			name := stmt.ColumnText(1)

			if id != ids[row] {
				t.Errorf("got %d, want %d", id, ids[row])
			}
			if name != names[row] {
				t.Errorf("got %q, want %q", name, names[row])
			}
		}
		if row != 3 {
			t.Errorf("got %d, want %d", row, len(ids))
		}

		if err := stmt.Err(); err != nil {
			t.Fatal(err)
		}

		err = stmt.Close()
		if err != nil {
			t.Fatal(err)
		}

		err = db.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()
}
