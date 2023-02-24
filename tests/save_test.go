package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/ncruces/go-sqlite3"
)

func TestConn_Savepoint_exec(t *testing.T) {
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	errFailed := errors.New("failed")

	count := func() int {
		stmt, _, err := db.Prepare(`SELECT count(*) FROM test`)
		if err != nil {
			t.Fatal(err)
		}
		if stmt.Step() {
			return stmt.ColumnInt(0)
		}
		t.Fatal(stmt.Err())
		return 0
	}

	insert := func(succeed bool) (err error) {
		defer db.Savepoint()(&err)

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
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO test VALUES ('one');`)
	if err != nil {
		t.Fatal(err)
	}

	panics := func() (err error) {
		defer db.Savepoint()(&err)

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
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	release := db.Savepoint()
	err = db.Exec(`INSERT INTO test(col) VALUES(1)`)
	if err != nil {
		t.Fatal(err)
	}
	release(&err)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	db.SetInterrupt(ctx)

	release1 := db.Savepoint()
	err = db.Exec(`INSERT INTO test(col) VALUES(2)`)
	if err != nil {
		t.Fatal(err)
	}
	release2 := db.Savepoint()
	err = db.Exec(`INSERT INTO test(col) VALUES(3)`)
	if err != nil {
		t.Fatal(err)
	}

	checkInterrupt := func(err error) {
		var serr *sqlite3.Error
		if err == nil {
			t.Fatal("want error")
		}
		if !errors.As(err, &serr) {
			t.Fatalf("got %T, want sqlite3.Error", err)
		}
		if rc := serr.Code(); rc != sqlite3.INTERRUPT {
			t.Errorf("got %d, want sqlite3.INTERRUPT", rc)
		}
		if got := err.Error(); got != `sqlite3: interrupted` {
			t.Error("got message: ", got)
		}
	}

	cancel()
	db.Savepoint()(&err)
	checkInterrupt(err)

	err = db.Exec(`INSERT INTO test(col) VALUES(4)`)
	checkInterrupt(err)

	err = context.Canceled
	release2(&err)
	if err != context.Canceled {
		t.Fatal(err)
	}

	var nilErr error
	release1(&nilErr)
	checkInterrupt(nilErr)

	db.SetInterrupt(nil)
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
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	release := db.Savepoint()
	err = db.Exec(`INSERT INTO test(col) VALUES(1)`)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Exec(`COMMIT`)
	if err != nil {
		t.Fatal(err)
	}
	release(&err)
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
