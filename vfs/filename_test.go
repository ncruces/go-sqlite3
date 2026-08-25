package vfs_test

import (
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/vfs"
)

type uriVFS struct {
	seen bool

	on, off, badBool, missingBool bool
	num, badNum, missingNum       int64
}

func (v *uriVFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	return nil, flags, sqlite3.CANTOPEN
}

func (v *uriVFS) OpenFilename(name *vfs.Filename, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	if flags&vfs.OPEN_MAIN_DB != 0 {
		v.seen = true
		v.on = name.URIBoolean("on", false)
		v.off = name.URIBoolean("off", true)
		v.badBool = name.URIBoolean("word", true)
		v.missingBool = name.URIBoolean("absent", true)
		v.num = name.URIInt64("num", -1)
		v.badNum = name.URIInt64("word", -1)
		v.missingNum = name.URIInt64("absent", -1)
	}
	return nil, flags, sqlite3.CANTOPEN
}

func (v *uriVFS) Delete(name string, syncDir bool) error { return nil }
func (v *uriVFS) Access(name string, flags vfs.AccessFlag) (bool, error) {
	return false, nil
}
func (v *uriVFS) FullPathname(name string) (string, error) { return name, nil }

func TestFilename_URITyped(t *testing.T) {
	var got uriVFS
	vfs.Register("urityped", &got)
	defer vfs.Unregister("urityped")

	conn, err := sqlite3.OpenContext(testcfg.Context(t),
		"file:test.db?vfs=urityped&on=yes&off=0&word=hello&num=42")
	if err == nil {
		conn.Close()
	}

	if !got.seen {
		t.Fatal("main database was never opened through the VFS")
	}
	if !got.on {
		t.Error("URIBoolean(on) = false, want true")
	}
	if got.off {
		t.Error("URIBoolean(off) = true, want false")
	}
	if !got.badBool {
		t.Error("URIBoolean(word) = false, want default true")
	}
	if !got.missingBool {
		t.Error("URIBoolean(absent) = false, want default true")
	}
	if got.num != 42 {
		t.Errorf("URIInt64(num) = %d, want 42", got.num)
	}
	if got.badNum != -1 {
		t.Errorf("URIInt64(word) = %d, want default -1", got.badNum)
	}
	if got.missingNum != -1 {
		t.Errorf("URIInt64(absent) = %d, want default -1", got.missingNum)
	}
}
