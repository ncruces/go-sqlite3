package tests

import (
	"bytes"
	"encoding/binary"
	"log"
	"strconv"
	"testing"

	"github.com/ncruces/go-sqlite3"
)

func Test_endianness(t *testing.T) {
	big := binary.BigEndian.AppendUint64(nil, 0x1234567890ABCDEF)
	little := binary.LittleEndian.AppendUint64(nil, 0x1234567890ABCDEF)
	native := binary.NativeEndian.AppendUint64(nil, 0x1234567890ABCDEF)
	switch {
	case bytes.Equal(big, native):
		t.Log("Platform is big endian")
	case bytes.Equal(little, native):
		t.Log("Platform is little endian")
	default:
		t.Fatal("Platform is middle endian")
	}

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE test (col)`)
	if err != nil {
		log.Fatal(err)
	}

	const value int64 = -9223372036854775808
	{
		stmt, _, err := db.Prepare(`INSERT INTO test VALUES (?)`)
		if err != nil {
			t.Fatal(err)
		}
		defer stmt.Close()

		err = stmt.BindInt64(1, value)
		if err != nil {
			t.Fatal(err)
		}

		err = stmt.Exec()
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		stmt, _, err := db.Prepare(`SELECT * FROM test`)
		if err != nil {
			t.Fatal(err)
		}
		defer stmt.Close()

		if !stmt.Step() {
			t.Fatal(stmt.Err())
		} else {
			if got := stmt.ColumnInt64(0); got != value {
				t.Errorf("got %d, want %d", got, value)
			}
			if got := stmt.ColumnText(0); got != strconv.FormatInt(value, 10) {
				t.Errorf("got %s, want %d", got, value)
			}
		}
	}
}
