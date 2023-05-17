package sqlite3vfs_test

import (
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/sqlite3vfs"
)

type testVFS struct {
	*testing.T
}

func (t testVFS) Open(name string, flags sqlite3vfs.OpenFlag) (sqlite3vfs.File, sqlite3vfs.OpenFlag, error) {
	t.Log("Open", name, flags)
	t.SkipNow()
	return nil, flags, nil
}

func (testVFS) Delete(name string, dirSync bool) error {
	panic("unimplemented")
}

func (testVFS) Access(name string, flags sqlite3vfs.AccessFlag) (bool, error) {
	panic("unimplemented")
}

func (t testVFS) FullPathname(name string) (string, error) {
	t.Log("FullPathname", name)
	return name, nil
}

func TestRegister(t *testing.T) {
	vfs := testVFS{t}
	sqlite3vfs.Register("foo", vfs)
	defer sqlite3vfs.Unregister("foo")

	defer func() { _ = recover() }()

	conn, err := sqlite3.Open("file:file.db?vfs=foo")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
}
