package vfs

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/julianday"
	"github.com/tetratelabs/wazero/experimental/wazerotest"
)

func Test_vfsLocaltime(t *testing.T) {
	mod := wazerotest.NewModule(wazerotest.NewMemory(wazerotest.PageSize))
	ctx := context.TODO()

	tm := time.Now()
	rc := vfsLocaltime(ctx, mod, 4, tm.Unix())
	if rc != 0 {
		t.Fatal("returned", rc)
	}

	if s := util.ReadUint32(mod, 4+0*4); int(s) != tm.Second() {
		t.Error("wrong second")
	}
	if m := util.ReadUint32(mod, 4+1*4); int(m) != tm.Minute() {
		t.Error("wrong minute")
	}
	if h := util.ReadUint32(mod, 4+2*4); int(h) != tm.Hour() {
		t.Error("wrong hour")
	}
	if d := util.ReadUint32(mod, 4+3*4); int(d) != tm.Day() {
		t.Error("wrong day")
	}
	if m := util.ReadUint32(mod, 4+4*4); time.Month(1+m) != tm.Month() {
		t.Error("wrong month")
	}
	if y := util.ReadUint32(mod, 4+5*4); 1900+int(y) != tm.Year() {
		t.Error("wrong year")
	}
	if w := util.ReadUint32(mod, 4+6*4); time.Weekday(w) != tm.Weekday() {
		t.Error("wrong weekday")
	}
	if d := util.ReadUint32(mod, 4+7*4); int(d) != tm.YearDay()-1 {
		t.Error("wrong yearday")
	}
}

func Test_vfsRandomness(t *testing.T) {
	mod := wazerotest.NewModule(wazerotest.NewMemory(wazerotest.PageSize))
	ctx := context.TODO()

	rc := vfsRandomness(ctx, mod, 0, 16, 4)
	if rc != 16 {
		t.Fatal("returned", rc)
	}

	var zero [16]byte
	if got := util.View(mod, 4, 16); bytes.Equal(got, zero[:]) {
		t.Fatal("all zero")
	}
}

func Test_vfsSleep(t *testing.T) {
	mod := wazerotest.NewModule(wazerotest.NewMemory(wazerotest.PageSize))
	ctx := context.TODO()

	now := time.Now()
	rc := vfsSleep(ctx, mod, 0, 123456)
	if rc != 0 {
		t.Fatal("returned", rc)
	}

	want := 123456 * time.Microsecond
	if got := time.Since(now); got < want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_vfsCurrentTime(t *testing.T) {
	mod := wazerotest.NewModule(wazerotest.NewMemory(wazerotest.PageSize))
	ctx := context.TODO()

	now := time.Now()
	rc := vfsCurrentTime(ctx, mod, 0, 4)
	if rc != 0 {
		t.Fatal("returned", rc)
	}

	want := julianday.Float(now)
	if got := util.ReadFloat64(mod, 4); float32(got) != float32(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_vfsCurrentTime64(t *testing.T) {
	mod := wazerotest.NewModule(wazerotest.NewMemory(wazerotest.PageSize))
	ctx := context.TODO()

	now := time.Now()
	time.Sleep(time.Millisecond)
	rc := vfsCurrentTime64(ctx, mod, 0, 4)
	if rc != 0 {
		t.Fatal("returned", rc)
	}

	day, nsec := julianday.Date(now)
	want := day*86_400_000 + nsec/1_000_000
	if got := util.ReadUint64(mod, 4); float32(got) != float32(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_vfsFullPathname(t *testing.T) {
	mod := wazerotest.NewModule(wazerotest.NewMemory(wazerotest.PageSize))
	util.WriteString(mod, 4, ".")
	ctx := context.TODO()

	rc := vfsFullPathname(ctx, mod, 0, 4, 0, 8)
	if rc != _CANTOPEN_FULLPATH {
		t.Errorf("returned %d, want %d", rc, _CANTOPEN_FULLPATH)
	}

	rc = vfsFullPathname(ctx, mod, 0, 4, _MAX_PATHNAME, 8)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	want, _ := filepath.Abs(".")
	if got := util.ReadString(mod, 8, _MAX_PATHNAME); got != want {
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

	mod := wazerotest.NewModule(wazerotest.NewMemory(wazerotest.PageSize))
	util.WriteString(mod, 4, name)
	ctx := context.TODO()

	rc := vfsDelete(ctx, mod, 0, 4, 1)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	if _, err := os.Stat(name); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("did not delete the file")
	}

	rc = vfsDelete(ctx, mod, 0, 4, 1)
	if rc != _IOERR_DELETE_NOENT {
		t.Fatal("returned", rc)
	}
}

func Test_vfsAccess(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.db")
	if f, err := os.Create(file); err != nil {
		t.Fatal(err)
	} else {
		f.Close()
	}
	if err := os.Chmod(file, syscall.S_IRUSR); err != nil {
		t.Fatal(err)
	}

	mod := wazerotest.NewModule(wazerotest.NewMemory(wazerotest.PageSize))
	util.WriteString(mod, 8, dir)
	ctx := context.TODO()

	rc := vfsAccess(ctx, mod, 0, 8, ACCESS_EXISTS, 4)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, 4); got != 1 {
		t.Error("directory did not exist")
	}

	rc = vfsAccess(ctx, mod, 0, 8, ACCESS_READWRITE, 4)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, 4); got != 1 {
		t.Error("can't access directory")
	}

	util.WriteString(mod, 8, file)
	rc = vfsAccess(ctx, mod, 0, 8, ACCESS_READ, 4)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, 4); got != 1 {
		t.Error("can't access file")
	}

	util.WriteString(mod, 8, file)
	rc = vfsAccess(ctx, mod, 0, 8, ACCESS_READWRITE, 4)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, 4); got != 0 {
		t.Error("can access file")
	}
}

func Test_vfsFile(t *testing.T) {
	mod := wazerotest.NewModule(wazerotest.NewMemory(wazerotest.PageSize))
	ctx := util.NewContext(context.TODO())

	// Open a temporary file.
	rc := vfsOpen(ctx, mod, 0, 0, 4, OPEN_CREATE|OPEN_EXCLUSIVE|OPEN_READWRITE|OPEN_DELETEONCLOSE, 0)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	// Check sector size.
	if size := vfsSectorSize(ctx, mod, 4); size != _DEFAULT_SECTOR_SIZE {
		t.Fatal("returned", size)
	}

	// Write stuff.
	text := "Hello world!"
	util.WriteString(mod, 16, text)
	rc = vfsWrite(ctx, mod, 4, 16, uint32(len(text)), 0)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	// Check file size.
	rc = vfsFileSize(ctx, mod, 4, 16)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, 16); got != uint32(len(text)) {
		t.Errorf("got %d", got)
	}

	// Partial read at offset.
	rc = vfsRead(ctx, mod, 4, 16, uint32(len(text)), 4)
	if rc != _IOERR_SHORT_READ {
		t.Fatal("returned", rc)
	}
	if got := util.ReadString(mod, 16, 64); got != text[4:] {
		t.Errorf("got %q", got)
	}

	// Truncate the file.
	rc = vfsTruncate(ctx, mod, 4, 4)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	// Check file size.
	rc = vfsFileSize(ctx, mod, 4, 16)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, 16); got != 4 {
		t.Errorf("got %d", got)
	}

	// Read at offset.
	rc = vfsRead(ctx, mod, 4, 32, 4, 0)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadString(mod, 32, 64); got != text[:4] {
		t.Errorf("got %q", got)
	}

	// Close the file.
	rc = vfsClose(ctx, mod, 4)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
}

func Test_vfsFile_psow(t *testing.T) {
	mod := wazerotest.NewModule(wazerotest.NewMemory(wazerotest.PageSize))
	ctx := util.NewContext(context.TODO())

	// Open a temporary file.
	rc := vfsOpen(ctx, mod, 0, 0, 4, OPEN_CREATE|OPEN_EXCLUSIVE|OPEN_READWRITE|OPEN_DELETEONCLOSE, 0)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	// Read powersafe overwrite.
	util.WriteUint32(mod, 16, math.MaxUint32)
	rc = vfsFileControl(ctx, mod, 4, _FCNTL_POWERSAFE_OVERWRITE, 16)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, 16); got == 0 {
		t.Error("psow disabled")
	}

	// Unset powersafe overwrite.
	util.WriteUint32(mod, 16, 0)
	rc = vfsFileControl(ctx, mod, 4, _FCNTL_POWERSAFE_OVERWRITE, 16)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	// Read powersafe overwrite.
	util.WriteUint32(mod, 16, math.MaxUint32)
	rc = vfsFileControl(ctx, mod, 4, _FCNTL_POWERSAFE_OVERWRITE, 16)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, 16); got != 0 {
		t.Error("psow enabled")
	}

	// Set powersafe overwrite.
	util.WriteUint32(mod, 16, 1)
	rc = vfsFileControl(ctx, mod, 4, _FCNTL_POWERSAFE_OVERWRITE, 16)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	// Read powersafe overwrite.
	util.WriteUint32(mod, 16, math.MaxUint32)
	rc = vfsFileControl(ctx, mod, 4, _FCNTL_POWERSAFE_OVERWRITE, 16)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, 16); got == 0 {
		t.Error("psow disabled")
	}

	// Close the file.
	rc = vfsClose(ctx, mod, 4)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
}
