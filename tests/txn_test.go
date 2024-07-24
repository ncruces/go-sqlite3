package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

func TestConn_Transaction_exec(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.RollbackHook(func() {})
	db.CommitHook(func() bool { return true })
	db.UpdateHook(func(sqlite3.AuthorizerActionCode, string, string, int64) {})

	err = db.Exec(`CREATE TABLE test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	errFailed := errors.New("failed")

	count := func() int {
		stmt, _, err := db.Prepare(`SELECT count(*) FROM test`)
		if err != nil {
			t.Fatal(err)
		}
		defer stmt.Close()
		if stmt.Step() {
			return stmt.ColumnInt(0)
		}
		t.Fatal(stmt.Err())
		return 0
	}

	insert := func(succeed bool) (err error) {
		tx := db.Begin()
		defer tx.End(&err)

		err = db.Exec(`INSERT INTO test VALUES ('hello')`)
		if err != nil {
			t.Fatal(err)
		}

		if s := db.TxnState("main"); s != sqlite3.TXN_WRITE {
			t.Errorf("got %d", s)
		}

		if succeed {
			return nil
		}
		return errFailed
	}

	err = insert(true)
	if err != nil {
		t.Fatal(err)
	}
	if got := count(); got != 1 {
		t.Errorf("got %d, want 1", got)
	}

	err = insert(true)
	if err != nil {
		t.Fatal(err)
	}
	if got := count(); got != 2 {
		t.Errorf("got %d, want 2", got)
	}

	err = insert(false)
	if err != errFailed {
		t.Errorf("got %v, want errFailed", err)
	}
	if got := count(); got != 2 {
		t.Errorf("got %d, want 2", got)
	}
}

func TestConn_Transaction_panic(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO test VALUES ('one');`)
	if err != nil {
		t.Fatal(err)
	}

	panics := func() (err error) {
		tx := db.Begin()
		defer tx.End(&err)

		err = db.Exec(`INSERT INTO test VALUES ('hello')`)
		if err != nil {
			return err
		}

		panic("omg!")
	}

	defer func() {
		p := recover()
		if p != "omg!" {
			t.Errorf("got %v, want panic", p)
		}

		stmt, _, err := db.Prepare(`SELECT count(*) FROM test`)
		if err != nil {
			t.Fatal(err)
		}
		defer stmt.Close()
		if stmt.Step() {
			got := stmt.ColumnInt(0)
			if got != 1 {
				t.Errorf("got %d, want 1", got)
			}
			return
		}
		t.Fatal(stmt.Err())
	}()

	err = panics()
	if err != nil {
		t.Error(err)
	}
}

func TestConn_Transaction_interrupt(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := db.BeginImmediate()
	if err != nil {
		t.Fatal(err)
	}
	err = db.Exec(`INSERT INTO test VALUES (1)`)
	if err != nil {
		t.Fatal(err)
	}
	tx.End(&err)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	db.SetInterrupt(ctx)

	tx, err = db.BeginExclusive()
	if err != nil {
		t.Fatal(err)
	}
	err = db.Exec(`INSERT INTO test VALUES (2)`)
	if err != nil {
		t.Fatal(err)
	}

	cancel()
	_, err = db.BeginImmediate()
	if !errors.Is(err, sqlite3.INTERRUPT) {
		t.Errorf("got %v, want sqlite3.INTERRUPT", err)
	}

	err = db.Exec(`INSERT INTO test VALUES (3)`)
	if !errors.Is(err, sqlite3.INTERRUPT) {
		t.Errorf("got %v, want sqlite3.INTERRUPT", err)
	}

	err = nil
	tx.End(&err)
	if !errors.Is(err, sqlite3.INTERRUPT) {
		t.Errorf("got %v, want sqlite3.INTERRUPT", err)
	}

	db.SetInterrupt(context.Background())
	stmt, _, err := db.Prepare(`SELECT count(*) FROM test`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	if stmt.Step() {
		got := stmt.ColumnInt(0)
		if got != 1 {
			t.Errorf("got %d, want 1", got)
		}
	}
	err = stmt.Err()
	if err != nil {
		t.Error(err)
	}
}

func TestConn_Transaction_interrupted(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	db.SetInterrupt(ctx)
	cancel()

	tx := db.Begin()

	err = tx.Commit()
	if !errors.Is(err, sqlite3.INTERRUPT) {
		t.Errorf("got %v, want sqlite3.INTERRUPT", err)
	}

	err = nil
	tx.End(&err)
	if !errors.Is(err, sqlite3.INTERRUPT) {
		t.Errorf("got %v, want sqlite3.INTERRUPT", err)
	}
}

func TestConn_Transaction_busy(t *testing.T) {
	t.Parallel()

	db1, err := sqlite3.Open("file:/test.db?vfs=memdb")
	if err != nil {
		t.Fatal(err)
	}
	defer db1.Close()

	db2, err := sqlite3.Open("file:/test.db?vfs=memdb&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()

	err = db1.Exec(`CREATE TABLE test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := db1.BeginImmediate()
	if err != nil {
		t.Fatal(err)
	}
	err = db1.Exec(`INSERT INTO test VALUES (1)`)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	db2.SetInterrupt(ctx)
	go cancel()

	_, err = db2.BeginExclusive()
	if !errors.Is(err, sqlite3.BUSY) && !errors.Is(err, sqlite3.INTERRUPT) {
		t.Errorf("got %v, want sqlite3.BUSY or sqlite3.INTERRUPT", err)
	}

	err = nil
	tx.End(&err)
	if err != nil {
		t.Fatal(err)
	}
}

func TestConn_Transaction_rollback(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	tx := db.Begin()
	err = db.Exec(`INSERT INTO test VALUES (1)`)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Exec(`COMMIT`)
	if err != nil {
		t.Fatal(err)
	}
	tx.End(&err)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT count(*) FROM test`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	if stmt.Step() {
		got := stmt.ColumnInt(0)
		if got != 1 {
			t.Errorf("got %d, want 1", got)
		}
	}
	err = stmt.Err()
	if err != nil {
		t.Error(err)
	}
}

func TestConn_Transaction_concurrent(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.BeginConcurrent()
	if !errors.Is(err, sqlite3.ERROR) {
		t.Errorf("got %v, want sqlite3.ERROR", err)
	}
}

func TestConn_Savepoint_exec(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	errFailed := errors.New("failed")

	count := func() int {
		stmt, _, err := db.Prepare(`SELECT count(*) FROM test`)
		if err != nil {
			t.Fatal(err)
		}
		defer stmt.Close()
		if stmt.Step() {
			return stmt.ColumnInt(0)
		}
		t.Fatal(stmt.Err())
		return 0
	}

	insert := func(succeed bool) (err error) {
		defer db.Savepoint().Release(&err)

		err = db.Exec(`INSERT INTO test VALUES ('hello')`)
		if err != nil {
			t.Fatal(err)
		}

		if succeed {
			return nil
		}
		return errFailed
	}

	err = insert(true)
	if err != nil {
		t.Fatal(err)
	}
	if got := count(); got != 1 {
		t.Errorf("got %d, want 1", got)
	}

	err = insert(true)
	if err != nil {
		t.Fatal(err)
	}
	if got := count(); got != 2 {
		t.Errorf("got %d, want 2", got)
	}

	err = insert(false)
	if err != errFailed {
		t.Errorf("got %v, want errFailed", err)
	}
	if got := count(); got != 2 {
		t.Errorf("got %d, want 2", got)
	}
}

func TestConn_Savepoint_panic(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO test VALUES ('one');`)
	if err != nil {
		t.Fatal(err)
	}

	panics := func() (err error) {
		defer db.Savepoint().Release(&err)

		err = db.Exec(`INSERT INTO test VALUES ('hello')`)
		if err != nil {
			return err
		}

		panic("omg!")
	}

	defer func() {
		p := recover()
		if p != "omg!" {
			t.Errorf("got %v, want panic", p)
		}

		stmt, _, err := db.Prepare(`SELECT count(*) FROM test`)
		if err != nil {
			t.Fatal(err)
		}
		defer stmt.Close()
		if stmt.Step() {
			got := stmt.ColumnInt(0)
			if got != 1 {
				t.Errorf("got %d, want 1", got)
			}
			return
		}
		t.Fatal(stmt.Err())
	}()

	err = panics()
	if err != nil {
		t.Error(err)
	}
}

func TestConn_Savepoint_interrupt(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	savept := db.Savepoint()
	err = db.Exec(`INSERT INTO test VALUES (1)`)
	if err != nil {
		t.Fatal(err)
	}
	savept.Release(&err)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	db.SetInterrupt(ctx)

	savept1 := db.Savepoint()
	err = db.Exec(`INSERT INTO test VALUES (2)`)
	if err != nil {
		t.Fatal(err)
	}
	savept2 := db.Savepoint()
	err = db.Exec(`INSERT INTO test VALUES (3)`)
	if err != nil {
		t.Fatal(err)
	}

	cancel()
	db.Savepoint().Release(&err)
	if !errors.Is(err, sqlite3.INTERRUPT) {
		t.Errorf("got %v, want sqlite3.INTERRUPT", err)
	}

	err = db.Exec(`INSERT INTO test VALUES (4)`)
	if !errors.Is(err, sqlite3.INTERRUPT) {
		t.Errorf("got %v, want sqlite3.INTERRUPT", err)
	}

	err = context.Canceled
	savept2.Release(&err)
	if err != context.Canceled {
		t.Fatal(err)
	}

	err = nil
	savept1.Release(&err)
	if !errors.Is(err, sqlite3.INTERRUPT) {
		t.Errorf("got %v, want sqlite3.INTERRUPT", err)
	}

	db.SetInterrupt(context.Background())
	stmt, _, err := db.Prepare(`SELECT count(*) FROM test`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	if stmt.Step() {
		got := stmt.ColumnInt(0)
		if got != 1 {
			t.Errorf("got %d, want 1", got)
		}
	}
	err = stmt.Err()
	if err != nil {
		t.Error(err)
	}
}

func TestConn_Savepoint_rollback(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	savept := db.Savepoint()
	err = db.Exec(`INSERT INTO test VALUES (1)`)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Exec(`COMMIT`)
	if err != nil {
		t.Fatal(err)
	}
	savept.Release(&err)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT count(*) FROM test`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	if stmt.Step() {
		got := stmt.ColumnInt(0)
		if got != 1 {
			t.Errorf("got %d, want 1", got)
		}
	}
	err = stmt.Err()
	if err != nil {
		t.Error(err)
	}
}
