package uuid

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func Test_generate(t *testing.T) {
	t.Parallel()
	dsn := memdb.TestDB(t)

	db, err := driver.Open(dsn, Register)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var u uuid.UUID

	// Version 4, SQLite compatible
	err = db.QueryRow(`SELECT uuid()`).Scan(&u)
	if err != nil {
		t.Fatal(err)
	}
	if got := u.Version(); got != 4 {
		t.Errorf("got %d, want 4", got)
	}

	// Invalid version
	err = db.QueryRow(`SELECT uuid(8)`).Scan(&u)
	if err == nil {
		t.Error("want error")
	}

	// Custom version, no arguments
	for _, want := range []uuid.Version{1, 2, 4, 6, 7} {
		err = db.QueryRow(`SELECT uuid(?)`, want).Scan(&u)
		if err != nil {
			t.Fatal(err)
		}
		if got := u.Version(); got != want {
			t.Errorf("got %d, want %d", got, want)
		}
	}

	// Version 2, custom arguments
	err = db.QueryRow(`SELECT uuid(2, 4)`).Scan(&u)
	if err == nil {
		t.Error("want error")
	}

	err = db.QueryRow(`SELECT uuid(2, 'group')`).Scan(&u)
	if err != nil {
		t.Fatal(err)
	}
	if got := u.Version(); got != 2 {
		t.Errorf("got %d, want 2", got)
	}
	if got := u.Domain(); got != uuid.Group {
		t.Errorf("got %d, want 1", got)
	}

	dce := []struct {
		out uuid.Domain
		in  any
		id  uint32
	}{
		{uuid.Person, "user", 42},
		{uuid.Group, "group", 42},
		{uuid.Org, "org", 42},
		{uuid.Person, 0, 42},
		{uuid.Group, 1, 42},
		{uuid.Org, 2, 42},
		{3, 3, 42},
	}
	for _, tt := range dce {
		err = db.QueryRow(`SELECT uuid(2, ?, ?)`, tt.in, tt.id).Scan(&u)
		if err != nil {
			t.Fatal(err)
		}
		if got := u.Version(); got != 2 {
			t.Errorf("got %d, want 2", got)
		}
		if got := u.Domain(); got != tt.out {
			t.Errorf("got %d, want %d", got, tt.out)
		}
		if got := u.ID(); got != tt.id {
			t.Errorf("got %d, want %d", got, tt.id)
		}
	}

	// Versions 3 and 5
	err = db.QueryRow(`SELECT uuid(3)`).Scan(&u)
	if err == nil {
		t.Error("want error")
	}

	err = db.QueryRow(`SELECT uuid(3, 0, '')`).Scan(&u)
	if err == nil {
		t.Error("want error")
	}

	var tstamp time.Time
	var version uuid.Version
	err = db.QueryRow(`
		SELECT
			column1,
			uuid_extract_version(column1),
			uuid_extract_timestamp(column1)
		FROM (VALUES (uuid(7)))
	`).Scan(&u, &version, &tstamp)
	if err != nil {
		t.Fatal(err)
	}
	if got := u.Version(); got != version {
		t.Errorf("got %d, want %d", got, version)
	}
	if got := time.Unix(u.Time().UnixTime()); !got.Equal(tstamp) {
		t.Errorf("got %v, want %v", got, tstamp)
	}

	tests := []struct {
		ver  uuid.Version
		ns   any
		data string
		u    uuid.UUID
	}{
		{3, "oid", "2.999", uuid.MustParse("31cb1efa-18c4-3d19-89ba-df6a74ddbd1d")},
		{3, "dns", "www.example.com", uuid.MustParse("5df41881-3aed-3515-88a7-2f4a814cf09e")},
		{3, "fqdn", "www.example.com", uuid.MustParse("5df41881-3aed-3515-88a7-2f4a814cf09e")},
		{3, "url", "https://www.example.com/", uuid.MustParse("7fed185f-0864-319f-875b-a3d5458e30ac")},
		{3, "x500", "CN=Test User 1, O=Example Organization, ST=California, C=US", uuid.MustParse("addf5e97-9287-3834-abfd-7edcbe7db56f")},
		{3, "url", "https://www.php.net", uuid.MustParse("3f703955-aaba-3e70-a3cb-baff6aa3b28f")},
		{5, "url", "https://www.php.net", uuid.MustParse("a8f6ae40-d8a7-58f0-be05-a22f94eca9ec")},
	}
	for _, tt := range tests {
		err = db.QueryRow(`SELECT uuid(?, ?, ?)`, tt.ver, tt.ns, tt.data).Scan(&u)
		if err != nil {
			t.Fatal(err)
		}
		if u != tt.u {
			t.Errorf("got %v, want %v", u, tt.u)
		}
	}
}

func Test_convert(t *testing.T) {
	t.Parallel()
	dsn := memdb.TestDB(t)

	db, err := driver.Open(dsn, Register)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var u uuid.UUID
	tests := []string{
		"'6ba7b8119dad11d180b400c04fd430c8'",
		"'6ba7b811-9dad-11d1-80b4-00c04fd430c8'",
		"'{6ba7b811-9dad-11d1-80b4-00c04fd430c8}'",
		"X'6ba7b8119dad11d180b400c04fd430c8'",
	}

	for _, tt := range tests {
		err = db.QueryRow(`SELECT uuid_str(` + tt + `)`).Scan(&u)
		if err != nil {
			t.Fatal(err)
		}
		if u != uuid.NameSpaceURL {
			t.Errorf("got %v, want %v", u, uuid.NameSpaceURL)
		}
	}

	for _, tt := range tests {
		err = db.QueryRow(`SELECT uuid_blob(` + tt + `)`).Scan(&u)
		if err != nil {
			t.Fatal(err)
		}
		if u != uuid.NameSpaceURL {
			t.Errorf("got %v, want %v", u, uuid.NameSpaceURL)
		}
	}

	err = db.QueryRow(`SELECT uuid_str(X'cafe')`).Scan(&u)
	if err == nil {
		t.Fatal("want error")
	}

	err = db.QueryRow(`SELECT uuid_blob(X'cafe')`).Scan(&u)
	if err == nil {
		t.Fatal("want error")
	}

	err = db.QueryRow(`SELECT uuid_extract_version(X'cafe')`).Scan(&u)
	if err == nil {
		t.Fatal("want error")
	}

	err = db.QueryRow(`SELECT uuid_extract_timestamp(X'cafe')`).Scan(&u)
	if err == nil {
		t.Fatal("want error")
	}
}
