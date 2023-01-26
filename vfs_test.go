package sqlite3

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/ncruces/julianday"
)

func Test_vfsLocaltime(t *testing.T) {
	mem := newMemory(128)

	rc := vfsLocaltime(context.TODO(), mem.mod, 0, 4)
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

	rand.Seed(0)
	rc := vfsRandomness(context.TODO(), mem.mod, 0, 16, 4)
	if rc != 16 {
		t.Fatal("returned", rc)
	}

	var want [16]byte
	rand.Seed(0)
	rand.Read(want[:])

	if got := mem.mustRead(4, 16); !bytes.Equal(got, want[:]) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_vfsSleep(t *testing.T) {
	start := time.Now()

	rc := vfsSleep(context.TODO(), 0, 123456)
	if rc != 0 {
		t.Fatal("returned", rc)
	}

	want := 123456 * time.Microsecond
	if got := time.Since(start); got < want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_vfsCurrentTime(t *testing.T) {
	mem := newMemory(128)

	now := time.Now()
	rc := vfsCurrentTime(context.TODO(), mem.mod, 0, 4)
	if rc != 0 {
		t.Fatal("returned", rc)
	}

	want := julianday.Float(now)
	if got := mem.readFloat64(4); float32(got) != float32(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_vfsCurrentTime64(t *testing.T) {
	memory := make(mockMemory, 128)
	module := &mockModule{&memory}

	now := time.Now()
	time.Sleep(time.Millisecond)
	rc := vfsCurrentTime64(context.TODO(), module, 0, 4)
	if rc != 0 {
		t.Fatal("returned", rc)
	}

	day, nsec := julianday.Date(now)
	want := day*86_400_000 + nsec/1_000_000
	if got, _ := memory.ReadUint64Le(4); int64(got)-want > 100 {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_vfsFullPathname(t *testing.T) {
	memory := make(mockMemory, 128+_MAX_PATHNAME)
	module := &mockModule{&memory}

	memory.Write(4, []byte{'.', 0})

	rc := vfsFullPathname(context.TODO(), module, 0, 4, 0, 8)
	if rc != uint32(CANTOPEN_FULLPATH) {
		t.Errorf("returned %d, want %d", rc, CANTOPEN_FULLPATH)
	}

	rc = vfsFullPathname(context.TODO(), module, 0, 4, _MAX_PATHNAME, 8)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	// want, _ := filepath.Abs(".")
	// if got := getString(&memory, 8, _MAX_PATHNAME); got != want {
	// 	t.Errorf("got %v, want %v", got, want)
	// }
}

func Test_vfsDelete(t *testing.T) {
	memory := make(mockMemory, 128+_MAX_PATHNAME)
	module := &mockModule{&memory}

	os.CreateTemp("", "sqlite3")
	file, err := os.CreateTemp("", "sqlite3-")
	if err != nil {
		t.Fatal(err)
	}
	name := file.Name()
	defer os.RemoveAll(name)
	file.Close()

	memory.Write(4, []byte(name))

	rc := vfsDelete(context.TODO(), module, 0, 4, 1)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	if _, err := os.Stat(name); !errors.Is(err, fs.ErrNotExist) {
		t.Error("did not delete the file")
	}
}

func Test_vfsAccess(t *testing.T) {
	memory := make(mockMemory, 128+_MAX_PATHNAME)
	module := &mockModule{&memory}

	os.CreateTemp("", "sqlite3")
	dir, err := os.MkdirTemp("", "sqlite3-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	memory.Write(8, []byte(dir))

	rc := vfsAccess(context.TODO(), module, 0, 8, ACCESS_EXISTS, 4)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got, ok := memory.ReadByte(4); !ok && got != 1 {
		t.Error("directory did not exist")
	}

	rc = vfsAccess(context.TODO(), module, 0, 8, ACCESS_READWRITE, 4)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got, ok := memory.ReadByte(4); !ok && got != 1 {
		t.Error("can't access directory")
	}
}
