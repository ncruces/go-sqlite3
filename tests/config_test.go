package tests

import (
	"errors"
	"math"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func TestConn_Config(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	o, err := db.Config(sqlite3.DBCONFIG_DEFENSIVE)
	if err != nil {
		t.Fatal(err)
	}
	if o != false {
		t.Error("want false")
	}

	o, err = db.Config(sqlite3.DBCONFIG_DEFENSIVE, true)
	if err != nil {
		t.Fatal(err)
	}
	if o != true {
		t.Error("want true")
	}

	o, err = db.Config(sqlite3.DBCONFIG_DEFENSIVE)
	if err != nil {
		t.Fatal(err)
	}
	if o != true {
		t.Error("want true")
	}

	o, err = db.Config(sqlite3.DBCONFIG_DEFENSIVE, false)
	if err != nil {
		t.Fatal(err)
	}
	if o != false {
		t.Error("want false")
	}

	o, err = db.Config(sqlite3.DBCONFIG_DEFENSIVE)
	if err != nil {
		t.Fatal(err)
	}
	if o != false {
		t.Error("want false")
	}

	_, err = db.Config(0)
	if !errors.Is(err, sqlite3.MISUSE) {
		t.Errorf("got %v, want MISUSE", err)
	}
}

func TestConn_ConfigLog(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var code sqlite3.ExtendedErrorCode
	err = db.ConfigLog(func(c sqlite3.ExtendedErrorCode, msg string) {
		t.Log(msg)
		code = c
	})
	if err != nil {
		t.Fatal(err)
	}

	db.Prepare(`SELECT * FRM sqlite_schema`)

	if code != sqlite3.ERROR.ExtendedCode() {
		t.Error("want sqlite3.ERROR")
	}

	db.Log(sqlite3.NOTICE.ExtendedCode(), "")

	if code.Code() != sqlite3.NOTICE {
		t.Error("want sqlite3.NOTICE")
	}
}

func TestConn_FileControl(t *testing.T) {
	t.Parallel()

	file := filepath.Join(t.TempDir(), "test.db")
	db, err := sqlite3.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	t.Run("MISUSE", func(t *testing.T) {
		_, err := db.FileControl("main", 0)
		if !errors.Is(err, sqlite3.MISUSE) {
			t.Errorf("got %v, want MISUSE", err)
		}
	})
	t.Run("FCNTL_RESET_CACHE", func(t *testing.T) {
		o, err := db.FileControl("", sqlite3.FCNTL_RESET_CACHE)
		if err != nil {
			t.Fatal(err)
		}
		if o != nil {
			t.Errorf("got %v, want nil", o)
		}
	})

	t.Run("FCNTL_PERSIST_WAL", func(t *testing.T) {
		o, err := db.FileControl("", sqlite3.FCNTL_PERSIST_WAL)
		if err != nil {
			t.Fatal(err)
		}
		if o != false {
			t.Errorf("got %v, want false", o)
		}

		o, err = db.FileControl("", sqlite3.FCNTL_PERSIST_WAL, true)
		if err != nil {
			t.Fatal(err)
		}
		if o != true {
			t.Errorf("got %v, want true", o)
		}

		o, err = db.FileControl("", sqlite3.FCNTL_PERSIST_WAL)
		if err != nil {
			t.Fatal(err)
		}
		if o != true {
			t.Errorf("got %v, want true", o)
		}
	})

	t.Run("FCNTL_CHUNK_SIZE", func(t *testing.T) {
		o, err := db.FileControl("", sqlite3.FCNTL_CHUNK_SIZE, 1024*1024)
		if !errors.Is(err, sqlite3.NOTFOUND) {
			t.Errorf("got %v, want NOTFOUND", err)
		}
		if o != nil {
			t.Errorf("got %v, want nil", o)
		}
	})

	t.Run("FCNTL_RESERVE_BYTES", func(t *testing.T) {
		o, err := db.FileControl("", sqlite3.FCNTL_RESERVE_BYTES, 4)
		if err != nil {
			t.Fatal(err)
		}
		if o != 0 {
			t.Errorf("got %v, want 0", o)
		}

		o, err = db.FileControl("", sqlite3.FCNTL_RESERVE_BYTES)
		if err != nil {
			t.Fatal(err)
		}
		if o != 4 {
			t.Errorf("got %v, want 4", o)
		}
	})

	t.Run("FCNTL_DATA_VERSION", func(t *testing.T) {
		o, err := db.FileControl("", sqlite3.FCNTL_DATA_VERSION)
		if err != nil {
			t.Fatal(err)
		}
		if o != uint32(2) {
			t.Errorf("got %v, want 2", o)
		}
	})

	t.Run("FCNTL_VFS_POINTER", func(t *testing.T) {
		o, err := db.FileControl("", sqlite3.FCNTL_VFS_POINTER)
		if err != nil {
			t.Fatal(err)
		}
		if o != vfs.Find("os") {
			t.Errorf("got %v, want os", o)
		}
	})

	t.Run("FCNTL_FILE_POINTER", func(t *testing.T) {
		o, err := db.FileControl("", sqlite3.FCNTL_FILE_POINTER)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := o.(vfs.File); !ok {
			t.Errorf("got %v, want File", o)
		}
	})

	t.Run("FCNTL_JOURNAL_POINTER", func(t *testing.T) {
		o, err := db.FileControl("", sqlite3.FCNTL_JOURNAL_POINTER)
		if err != nil {
			t.Fatal(err)
		}
		if o != nil {
			t.Errorf("got %v, want nil", o)
		}
	})

	t.Run("FCNTL_LOCKSTATE", func(t *testing.T) {
		if !vfs.SupportsFileLocking {
			t.Skip("skipping without locks")
		}

		txn, err := db.BeginExclusive()
		if err != nil {
			t.Fatal(err)
		}
		defer txn.End(&err)

		o, err := db.FileControl("", sqlite3.FCNTL_LOCKSTATE)
		if err != nil {
			t.Fatal(err)
		}
		if o != vfs.LOCK_EXCLUSIVE {
			t.Errorf("got %v, want LOCK_EXCLUSIVE", o)
		}
	})
}

func TestConn_Limit(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	l := db.Limit(sqlite3.LIMIT_COLUMN, -1)
	if l != 2000 {
		t.Errorf("got %d, want 2000", l)
	}

	l = db.Limit(sqlite3.LIMIT_COLUMN, 100)
	if l != 2000 {
		t.Errorf("got %d, want 2000", l)
	}

	l = db.Limit(sqlite3.LIMIT_COLUMN, -1)
	if l != 100 {
		t.Errorf("got %d, want 100", l)
	}

	l = db.Limit(math.MaxUint32, -1)
	if l != -1 {
		t.Errorf("got %d, want -1", l)
	}
}

func TestConn_SetAuthorizer(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.SetAuthorizer(func(action sqlite3.AuthorizerActionCode, name3rd, name4th, schema, nameInner string) sqlite3.AuthorizerReturnCode {
		if action != sqlite3.AUTH_PRAGMA {
			t.Errorf("got %v, want PRAGMA", action)
		}
		if name3rd != "busy_timeout" {
			t.Errorf("got %q, want busy_timeout", name3rd)
		}
		if name4th != "5000" {
			t.Errorf("got %q, want 5000", name4th)
		}
		if schema != "main" {
			t.Errorf("got %q, want main", schema)
		}
		return sqlite3.AUTH_DENY
	})
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`PRAGMA main.busy_timeout=5000`)
	if !errors.Is(err, sqlite3.AUTH) {
		t.Errorf("got %v, want sqlite3.AUTH", err)
	}
}

func TestConn_Trace(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rows := 0
	closed := false
	err = db.Trace(math.MaxUint32, func(evt sqlite3.TraceEvent, a1 any, a2 any) error {
		switch evt {
		case sqlite3.TRACE_CLOSE:
			closed = true
			_ = a1.(*sqlite3.Conn)
			return db.Exec(`PRAGMA optimize`)
		case sqlite3.TRACE_STMT:
			stmt := a1.(*sqlite3.Stmt)
			if sql := a2.(string); sql != stmt.SQL() {
				t.Errorf("got %q, want %q", sql, stmt.SQL())
			}
			if sql := stmt.ExpandedSQL(); sql != `SELECT 1` {
				t.Errorf("got %q", sql)
			}
		case sqlite3.TRACE_PROFILE:
			_ = a1.(*sqlite3.Stmt)
			if ns := a2.(int64); ns < 0 {
				t.Errorf("got %d", ns)
			}
		case sqlite3.TRACE_ROW:
			_ = a1.(*sqlite3.Stmt)
			if a2 != nil {
				t.Errorf("got %v", a2)
			}
			rows++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT ?`)
	if err != nil {
		t.Fatal(err)
	}
	err = stmt.BindInt(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	err = stmt.Exec()
	if err != nil {
		t.Fatal(err)
	}
	err = stmt.Close()
	if err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Error("want 1")
	}

	err = db.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !closed {
		t.Error("want closed")
	}
}

func TestConn_AutoVacuumPages(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t, url.Values{
		"_pragma": {"auto_vacuum(full)"},
	})

	db, err := sqlite3.Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.AutoVacuumPages(func(schema string, dbPages, freePages, bytesPerPage uint) uint {
		return freePages
	})
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`CREATE TABLE test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO test VALUES (zeroblob(1024*1024))`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`DROP TABLE test`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestConn_memoryLimit(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	n := db.HardHeapLimit(-1)
	if n != 0 {
		t.Fatal("want zero")
	}

	const limit = 64 * 1024 * 1024

	n = db.SoftHeapLimit(limit)
	if n != 0 {
		t.Fatal("want zero")
	}

	n = db.SoftHeapLimit(-1)
	if n != limit {
		t.Fatal("want", limit)
	}
}
