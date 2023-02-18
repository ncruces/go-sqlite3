package sqlite3

import (
	"math"
	"testing"
)

func TestStmt(t *testing.T) {
	t.Parallel()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`INSERT INTO test(col) VALUES(?)`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	if got := stmt.BindCount(); got != 1 {
		t.Errorf("got %d, want 1", got)
	}

	err = stmt.BindBool(1, false)
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.Exec()
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.ClearBindings()
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.Exec()
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.BindBool(1, true)
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.Exec()
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.BindInt(1, 2)
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.Exec()
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.BindFloat(1, math.Pi)
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.Exec()
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.BindNull(1)
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.Exec()
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.BindText(1, "")
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.Exec()
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.BindText(1, "text")
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.Exec()
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.BindBlob(1, []byte("blob"))
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.Exec()
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.BindBlob(1, nil)
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

	// The table should have: 0, NULL, 1, 2, π, NULL, "", "text", `blob`, NULL
	stmt, _, err = db.Prepare(`SELECT col FROM test`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != INTEGER {
			t.Errorf("got %v, want INTEGER", got)
		}
		if got := stmt.ColumnBool(0); got != false {
			t.Errorf("got %v, want false", got)
		}
		if got := stmt.ColumnInt(0); got != 0 {
			t.Errorf("got %v, want zero", got)
		}
		if got := stmt.ColumnFloat(0); got != 0 {
			t.Errorf("got %v, want zero", got)
		}
		if got := stmt.ColumnText(0); got != "0" {
			t.Errorf("got %q, want zero", got)
		}
		if got := stmt.ColumnBlob(0, nil); string(got) != "0" {
			t.Errorf("got %q, want zero", got)
		}
	}

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != NULL {
			t.Errorf("got %v, want NULL", got)
		}
		if got := stmt.ColumnBool(0); got != false {
			t.Errorf("got %v, want false", got)
		}
		if got := stmt.ColumnInt(0); got != 0 {
			t.Errorf("got %v, want zero", got)
		}
		if got := stmt.ColumnFloat(0); got != 0 {
			t.Errorf("got %v, want zero", got)
		}
		if got := stmt.ColumnText(0); got != "" {
			t.Errorf("got %q, want empty", got)
		}
		if got := stmt.ColumnBlob(0, nil); got != nil {
			t.Errorf("got %q, want nil", got)
		}
	}

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != INTEGER {
			t.Errorf("got %v, want INTEGER", got)
		}
		if got := stmt.ColumnBool(0); got != true {
			t.Errorf("got %v, want true", got)
		}
		if got := stmt.ColumnInt(0); got != 1 {
			t.Errorf("got %v, want one", got)
		}
		if got := stmt.ColumnFloat(0); got != 1 {
			t.Errorf("got %v, want one", got)
		}
		if got := stmt.ColumnText(0); got != "1" {
			t.Errorf("got %q, want one", got)
		}
		if got := stmt.ColumnBlob(0, nil); string(got) != "1" {
			t.Errorf("got %q, want one", got)
		}
	}

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != INTEGER {
			t.Errorf("got %v, want INTEGER", got)
		}
		if got := stmt.ColumnBool(0); got != true {
			t.Errorf("got %v, want true", got)
		}
		if got := stmt.ColumnInt(0); got != 2 {
			t.Errorf("got %v, want two", got)
		}
		if got := stmt.ColumnFloat(0); got != 2 {
			t.Errorf("got %v, want two", got)
		}
		if got := stmt.ColumnText(0); got != "2" {
			t.Errorf("got %q, want two", got)
		}
		if got := stmt.ColumnBlob(0, nil); string(got) != "2" {
			t.Errorf("got %q, want two", got)
		}
	}

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != FLOAT {
			t.Errorf("got %v, want FLOAT", got)
		}
		if got := stmt.ColumnBool(0); got != true {
			t.Errorf("got %v, want true", got)
		}
		if got := stmt.ColumnInt(0); got != 3 {
			t.Errorf("got %v, want three", got)
		}
		if got := stmt.ColumnFloat(0); got != math.Pi {
			t.Errorf("got %v, want π", got)
		}
		if got := stmt.ColumnText(0); got != "3.14159265358979" {
			t.Errorf("got %q, want π", got)
		}
		if got := stmt.ColumnBlob(0, nil); string(got) != "3.14159265358979" {
			t.Errorf("got %q, want π", got)
		}
	}

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != NULL {
			t.Errorf("got %v, want NULL", got)
		}
		if got := stmt.ColumnBool(0); got != false {
			t.Errorf("got %v, want false", got)
		}
		if got := stmt.ColumnInt(0); got != 0 {
			t.Errorf("got %v, want zero", got)
		}
		if got := stmt.ColumnFloat(0); got != 0 {
			t.Errorf("got %v, want zero", got)
		}
		if got := stmt.ColumnText(0); got != "" {
			t.Errorf("got %q, want empty", got)
		}
		if got := stmt.ColumnBlob(0, nil); got != nil {
			t.Errorf("got %q, want nil", got)
		}
	}

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != TEXT {
			t.Errorf("got %v, want TEXT", got)
		}
		if got := stmt.ColumnBool(0); got != false {
			t.Errorf("got %v, want false", got)
		}
		if got := stmt.ColumnInt(0); got != 0 {
			t.Errorf("got %v, want zero", got)
		}
		if got := stmt.ColumnFloat(0); got != 0 {
			t.Errorf("got %v, want zero", got)
		}
		if got := stmt.ColumnText(0); got != "" {
			t.Errorf("got %q, want empty", got)
		}
		if got := stmt.ColumnBlob(0, nil); got != nil {
			t.Errorf("got %q, want nil", got)
		}
	}

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != TEXT {
			t.Errorf("got %v, want TEXT", got)
		}
		if got := stmt.ColumnBool(0); got != false {
			t.Errorf("got %v, want false", got)
		}
		if got := stmt.ColumnInt(0); got != 0 {
			t.Errorf("got %v, want zero", got)
		}
		if got := stmt.ColumnFloat(0); got != 0 {
			t.Errorf("got %v, want zero", got)
		}
		if got := stmt.ColumnText(0); got != "text" {
			t.Errorf(`got %q, want "text"`, got)
		}
		if got := stmt.ColumnBlob(0, nil); string(got) != "text" {
			t.Errorf(`got %q, want "text"`, got)
		}
	}

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != BLOB {
			t.Errorf("got %v, want BLOB", got)
		}
		if got := stmt.ColumnBool(0); got != false {
			t.Errorf("got %v, want false", got)
		}
		if got := stmt.ColumnInt(0); got != 0 {
			t.Errorf("got %v, want zero", got)
		}
		if got := stmt.ColumnFloat(0); got != 0 {
			t.Errorf("got %v, want zero", got)
		}
		if got := stmt.ColumnText(0); got != "blob" {
			t.Errorf(`got %q, want "blob"`, got)
		}
		if got := stmt.ColumnBlob(0, nil); string(got) != "blob" {
			t.Errorf(`got %q, want "blob"`, got)
		}
	}

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != NULL {
			t.Errorf("got %v, want NULL", got)
		}
		if got := stmt.ColumnBool(0); got != false {
			t.Errorf("got %v, want false", got)
		}
		if got := stmt.ColumnInt(0); got != 0 {
			t.Errorf("got %v, want zero", got)
		}
		if got := stmt.ColumnFloat(0); got != 0 {
			t.Errorf("got %v, want zero", got)
		}
		if got := stmt.ColumnText(0); got != "" {
			t.Errorf("got %q, want empty", got)
		}
		if got := stmt.ColumnBlob(0, nil); got != nil {
			t.Errorf("got %q, want nil", got)
		}
	}

	err = stmt.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = db.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestStmt_Close(t *testing.T) {
	var stmt *Stmt
	stmt.Close()
}

func TestStmt_BindName(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	want := []string{"", "", "", "", "?5", ":AAA", "@AAA", "$AAA"}
	stmt, _, err := db.Prepare(`SELECT ?, ?5, :AAA, @AAA, $AAA`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	if got := stmt.BindCount(); got != len(want) {
		t.Errorf("got %d, want %d", got, len(want))
	}

	for i, name := range want {
		id := i + 1
		if got := stmt.BindName(id); got != name {
			t.Errorf("got %q, want %q", got, name)
		}
		if name == "" {
			id = 0
		}
		if got := stmt.BindIndex(name); got != id {
			t.Errorf("got %d, want %d", got, id)
		}
	}
}
