package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/ncruces/go-sqlite3"
)

func TestConn_Transaction_exec(t *testing.T) {
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
		tx := db.Begin()
		defer tx.End(&err)

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

func TestConn_Transaction_panic(t *testing.T) {
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
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := db.BeginImmediate()
	if err != nil {
		t.Fatal(err)
	}
	err = db.Exec(`INSERT INTO test(col) VALUES(1)`)
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
	err = db.Exec(`INSERT INTO test(col) VALUES(2)`)
	if err != nil {
		t.Fatal(err)
	}

	cancel()
	_, err = db.BeginImmediate()
	if !errors.Is(err, sqlite3.INTERRUPT) {
		t.Errorf("got %v, want sqlite3.INTERRUPT", err)
	}

	err = db.Exec(`INSERT INTO test(col) VALUES(3)`)
	if !errors.Is(err, sqlite3.INTERRUPT) {
		t.Errorf("got %v, want sqlite3.INTERRUPT", err)
	}

	var nilErr error
	tx.End(&nilErr)
	if !errors.Is(nilErr, sqlite3.INTERRUPT) {
		t.Errorf("got %v, want sqlite3.INTERRUPT", nilErr)
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

func TestConn_Transaction_rollback(t *testing.T) {
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	tx := db.Begin()
	err = db.Exec(`INSERT INTO test(col) VALUES(1)`)
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

	cancel()
	db.Savepoint()(&err)
	if !errors.Is(err, sqlite3.INTERRUPT) {
		t.Errorf("got %v, want sqlite3.INTERRUPT", err)
	}

	err = db.Exec(`INSERT INTO test(col) VALUES(4)`)
	if !errors.Is(err, sqlite3.INTERRUPT) {
		t.Errorf("got %v, want sqlite3.INTERRUPT", err)
	}

	err = context.Canceled
	release2(&err)
	if err != context.Canceled {
		t.Fatal(err)
	}

	var nilErr error
	release1(&nilErr)
	if !errors.Is(nilErr, sqlite3.INTERRUPT) {
		t.Errorf("got %v, want sqlite3.INTERRUPT", nilErr)
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
