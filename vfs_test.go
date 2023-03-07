package sqlite3

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/ncruces/julianday"
)

func Test_vfsExit(t *testing.T) {
	mem := newMemory(128)
	ctx := context.TODO()
	defer func() { _ = recover() }()
	vfsExit(ctx, mem.mod, 1)
	t.Error("want panic")
}

func Test_vfsLocaltime(t *testing.T) {
	mem := newMemory(128)
	ctx := context.TODO()

	rc := vfsLocaltime(ctx, mem.mod, 0, 4)
	if rc != 0 {
		t.Fatal("returned", rc)
	}

	epoch := time.Unix(0, 0)
	if s := mem.readUint32(4 + 0*4); int(s) != epoch.Second() {
		t.Error("wrong second")
	}
	if m := mem.readUint32(4 + 1*4); int(m) != epoch.Minute() {
		t.Error("wrong minute")
	}
	if h := mem.readUint32(4 + 2*4); int(h) != epoch.Hour() {
		t.Error("wrong hour")
	}
	if d := mem.readUint32(4 + 3*4); int(d) != epoch.Day() {
		t.Error("wrong day")
	}
	if m := mem.readUint32(4 + 4*4); time.Month(1+m) != epoch.Month() {
		t.Error("wrong month")
	}
	if y := mem.readUint32(4 + 5*4); 1900+int(y) != epoch.Year() {
		t.Error("wrong year")
	}
	if w := mem.readUint32(4 + 6*4); time.Weekday(w) != epoch.Weekday() {
		t.Error("wrong weekday")
	}
	if d := mem.readUint32(4 + 7*4); int(d) != epoch.YearDay()-1 {
		t.Error("wrong yearday")
	}
}

func Test_vfsRandomness(t *testing.T) {
	mem := newMemory(128)

	rc := vfsRandomness(context.TODO(), mem.mod, 0, 16, 4)
	if rc != 16 {
		t.Fatal("returned", rc)
	}

	var zero [16]byte
	if got := mem.view(4, 16); bytes.Equal(got, zero[:]) {
		t.Fatal("all zero")
	}
}

func Test_vfsSleep(t *testing.T) {
	ctx := context.TODO()

	now := time.Now()
	rc := vfsSleep(ctx, 0, 123456)
	if rc != 0 {
		t.Fatal("returned", rc)
	}

	want := 123456 * time.Microsecond
	if got := time.Since(now); got < want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_vfsCurrentTime(t *testing.T) {
	mem := newMemory(128)
	ctx := context.TODO()

	now := time.Now()
	rc := vfsCurrentTime(ctx, mem.mod, 0, 4)
	if rc != 0 {
		t.Fatal("returned", rc)
	}

	want := julianday.Float(now)
	if got := mem.readFloat64(4); float32(got) != float32(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_vfsCurrentTime64(t *testing.T) {
	mem := newMemory(128)
	ctx := context.TODO()

	now := time.Now()
	time.Sleep(time.Millisecond)
	rc := vfsCurrentTime64(ctx, mem.mod, 0, 4)
	if rc != 0 {
		t.Fatal("returned", rc)
	}

	day, nsec := julianday.Date(now)
	want := day*86_400_000 + nsec/1_000_000
	if got := mem.readUint64(4); float32(got) != float32(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_vfsFullPathname(t *testing.T) {
	mem := newMemory(128 + _MAX_PATHNAME)
	mem.writeString(4, ".")
	ctx := context.TODO()

	rc := vfsFullPathname(ctx, mem.mod, 0, 4, 0, 8)
	if rc != uint32(CANTOPEN_FULLPATH) {
		t.Errorf("returned %d, want %d", rc, CANTOPEN_FULLPATH)
	}

	rc = vfsFullPathname(ctx, mem.mod, 0, 4, _MAX_PATHNAME, 8)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	want, _ := filepath.Abs(".")
	if got := mem.readString(8, _MAX_PATHNAME); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_vfsDelete(t *testing.T) {
	name := filepath.Join(t.TempDir(), "test.db")

	file, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	file.Close()

	mem := newMemory(128 + _MAX_PATHNAME)
	mem.writeString(4, name)
	ctx := context.TODO()

	rc := vfsDelete(ctx, mem.mod, 0, 4, 1)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	if _, err := os.Stat(name); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("did not delete the file")
	}

	rc = vfsDelete(ctx, mem.mod, 0, 4, 1)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
}

func Test_vfsAccess(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(t.TempDir(), "test.db")
	if f, err := os.Create(file); err != nil {
		t.Fatal(err)
	} else {
		f.Close()
	}
	if err := os.Chmod(file, syscall.S_IRUSR); err != nil {
		t.Fatal(err)
	}

	mem := newMemory(128 + _MAX_PATHNAME)
	mem.writeString(8, dir)
	ctx := context.TODO()

	rc := vfsAccess(ctx, mem.mod, 0, 8, _ACCESS_EXISTS, 4)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readUint32(4); got != 1 {
		t.Error("directory did not exist")
	}

	rc = vfsAccess(ctx, mem.mod, 0, 8, _ACCESS_READWRITE, 4)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readUint32(4); got != 1 {
		t.Error("can't access directory")
	}

	mem.writeString(8, file)
	rc = vfsAccess(ctx, mem.mod, 0, 8, _ACCESS_READWRITE, 4)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readUint32(4); got != 0 {
		t.Error("can access file")
	}
}

func Test_vfsFile(t *testing.T) {
	mem := newMemory(128)
	ctx, vfs := vfsContext(context.TODO())
	defer vfs.Close()

	// Open a temporary file.
	rc := vfsOpen(ctx, mem.mod, 0, 0, 4, OPEN_CREATE|OPEN_EXCLUSIVE|OPEN_READWRITE|OPEN_DELETEONCLOSE, 0)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	// Write stuff.
	text := "Hello world!"
	mem.writeString(16, text)
	rc = vfsWrite(ctx, mem.mod, 4, 16, uint32(len(text)), 0)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	// Check file size.
	rc = vfsFileSize(ctx, mem.mod, 4, 16)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readUint32(16); got != uint32(len(text)) {
		t.Errorf("got %d", got)
	}

	// Partial read at offset.
	rc = vfsRead(ctx, mem.mod, 4, 16, uint32(len(text)), 4)
	if rc != uint32(IOERR_SHORT_READ) {
		t.Fatal("returned", rc)
	}
	if got := mem.readString(16, 64); got != text[4:] {
		t.Errorf("got %q", got)
	}

	// Truncate the file.
	rc = vfsTruncate(ctx, mem.mod, 4, 4)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	// Check file size.
	rc = vfsFileSize(ctx, mem.mod, 4, 16)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readUint32(16); got != 4 {
		t.Errorf("got %d", got)
	}

	// Read at offset.
	rc = vfsRead(ctx, mem.mod, 4, 32, 4, 0)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readString(32, 64); got != text[:4] {
		t.Errorf("got %q", got)
	}

	// Close the file.
	rc = vfsClose(ctx, mem.mod, 4)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
}
