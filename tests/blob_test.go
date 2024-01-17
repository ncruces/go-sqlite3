package tests

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"hash/adler32"
	"io"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestBlob(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO test VALUES (zeroblob(1024))`)
	if err != nil {
		t.Fatal(err)
	}

	blob, err := db.OpenBlob("main", "test", "col", db.LastInsertRowID(), true)
	if err != nil {
		t.Fatal(err)
	}
	defer blob.Close()

	size := blob.Size()
	if size != 1024 {
		t.Errorf("got %d, want 1024", size)
	}

	var data [1280]byte
	_, err = rand.Read(data[:])
	if err != nil {
		t.Fatal(err)
	}

	_, err = blob.Write(data[:size/2])
	if err != nil {
		t.Fatal(err)
	}

	n, err := blob.Write(data[:])
	if n != 0 || !errors.Is(err, sqlite3.ERROR) {
		t.Fatalf("got (%d, %v), want (0, ERROR)", n, err)
	}

	_, err = blob.Write(data[size/2 : size])
	if err != nil {
		t.Fatal(err)
	}

	_, err = blob.Seek(size/4, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}

	if got, err := io.ReadAll(blob); err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(got, data[size/4:size]) {
		t.Errorf("got %q, want %q", got, data[size/4:size])
	}

	if n, err := blob.Read(make([]byte, 1)); n != 0 || err != io.EOF {
		t.Errorf("got (%d, %v), want (0, EOF)", n, err)
	}

	if err := blob.Close(); err != nil {
		t.Fatal(err)
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestBlob_large(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO test VALUES (zeroblob(1000000))`)
	if err != nil {
		t.Fatal(err)
	}

	blob, err := db.OpenBlob("main", "test", "col", db.LastInsertRowID(), true)
	if err != nil {
		t.Fatal(err)
	}
	defer blob.Close()

	size := blob.Size()
	if size != 1000000 {
		t.Errorf("got %d, want 1000000", size)
	}

	hash := adler32.New()
	_, err = io.CopyN(blob, io.TeeReader(rand.Reader, hash), 1000000)
	if err != nil {
		t.Fatal(err)
	}

	_, err = blob.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}

	want := hash.Sum32()
	hash.Reset()
	_, err = io.Copy(hash, blob)
	if err != nil {
		t.Fatal(err)
	}

	if got := hash.Sum32(); got != want {
		t.Fatalf("got %d, want %d", got, want)
	}

	if err := blob.Close(); err != nil {
		t.Fatal(err)
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestBlob_overflow(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO test VALUES (zeroblob(1024))`)
	if err != nil {
		t.Fatal(err)
	}

	blob, err := db.OpenBlob("main", "test", "col", db.LastInsertRowID(), true)
	if err != nil {
		t.Fatal(err)
	}
	defer blob.Close()

	n, err := blob.ReadFrom(rand.Reader)
	if n != 1024 || !errors.Is(err, sqlite3.ERROR) {
		t.Fatalf("got (%d, %v), want (0, ERROR)", n, err)
	}

	n, err = blob.ReadFrom(rand.Reader)
	if n != 0 || !errors.Is(err, sqlite3.ERROR) {
		t.Fatalf("got (%d, %v), want (0, ERROR)", n, err)
	}

	_, err = blob.Seek(-128, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}

	n, err = blob.WriteTo(io.Discard)
	if n != 128 || err != nil {
		t.Fatalf("got (%d, %v), want (128, nil)", n, err)
	}

	n, err = blob.WriteTo(io.Discard)
	if n != 0 || err != nil {
		t.Fatalf("got (%d, %v), want (0, nil)", n, err)
	}

	if err := blob.Close(); err != nil {
		t.Fatal(err)
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestBlob_invalid(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO test VALUES (zeroblob(1024))`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.OpenBlob("", "test", "col", db.LastInsertRowID(), false)
	if !errors.Is(err, sqlite3.ERROR) {
		t.Fatal("want error")
	}
}

func TestBlob_Write_readonly(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO test VALUES (zeroblob(1024))`)
	if err != nil {
		t.Fatal(err)
	}

	blob, err := db.OpenBlob("main", "test", "col", db.LastInsertRowID(), false)
	if err != nil {
		t.Fatal(err)
	}
	defer blob.Close()

	_, err = blob.Write([]byte("data"))
	if !errors.Is(err, sqlite3.READONLY) {
		t.Fatal("want error")
	}
}

func TestBlob_Read_expired(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO test VALUES (zeroblob(1024))`)
	if err != nil {
		t.Fatal(err)
	}

	blob, err := db.OpenBlob("main", "test", "col", db.LastInsertRowID(), false)
	if err != nil {
		t.Fatal(err)
	}
	defer blob.Close()

	err = db.Exec(`DELETE FROM test`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = io.ReadAll(blob)
	if !errors.Is(err, sqlite3.ABORT) {
		t.Fatal("want error", err)
	}
}

func TestBlob_Seek(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO test VALUES (zeroblob(1024))`)
	if err != nil {
		t.Fatal(err)
	}

	blob, err := db.OpenBlob("main", "test", "col", db.LastInsertRowID(), true)
	if err != nil {
		t.Fatal(err)
	}
	defer blob.Close()

	_, err = blob.Seek(0, 10)
	if err == nil {
		t.Fatal("want error")
	}

	_, err = blob.Seek(-1, io.SeekCurrent)
	if err == nil {
		t.Fatal("want error")
	}

	n, err := blob.Seek(1, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}
	if n != blob.Size()+1 {
		t.Errorf("got %d, want %d", n, blob.Size())
	}

	_, err = blob.Write([]byte("data"))
	if !errors.Is(err, sqlite3.ERROR) {
		t.Fatal("want error")
	}
}

func TestBlob_Reopen(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	var rowids []int64
	for i := 0; i < 100; i++ {
		err = db.Exec(`INSERT INTO test VALUES (zeroblob(10))`)
		if err != nil {
			t.Fatal(err)
		}
		rowids = append(rowids, db.LastInsertRowID())
	}
	if changes := db.Changes(); changes != 1 {
		t.Errorf("got %d want 1", changes)
	}
	if changes := db.TotalChanges(); changes != 100 {
		t.Errorf("got %d want 100", changes)
	}

	var blob *sqlite3.Blob

	for i, rowid := range rowids {
		if i > 0 {
			err = blob.Reopen(rowid)
		} else {
			blob, err = db.OpenBlob("main", "test", "col", rowid, true)
		}
		if err != nil {
			t.Fatal(err)
		}

		_, err = fmt.Fprintf(blob, "blob %d\n", i)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := blob.Close(); err != nil {
		t.Fatal(err)
	}

	for i, rowid := range rowids {
		if i > 0 {
			err = blob.Reopen(rowid)
		} else {
			blob, err = db.OpenBlob("main", "test", "col", rowid, false)
		}
		if err != nil {
			t.Fatal(err)
		}

		var got int
		_, err = fmt.Fscanf(blob, "blob %d\n", &got)
		if err != nil {
			t.Fatal(err)
		}
		if got != i {
			t.Errorf("got %d, want %d", got, i)
		}
	}
	if err := blob.Close(); err != nil {
		t.Fatal(err)
	}
}
