//go:build unix && !sqlite3_flock && !sqlite3_nosys

package vfs

import (
	"crypto/rand"
	"io"
	"os"
	"runtime"
	"testing"

	"golang.org/x/sys/unix"
)

func Test_osAllocate(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip()
	}

	f, err := os.CreateTemp(t.TempDir(), "file")
	if err != nil {
		t.Fatal(err)
	}

	_, err = io.CopyN(f, rand.Reader, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	n, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1024*1024 {
		t.Fatalf("got %d, want %d", n, 1024*1024)
	}

	err = osAllocate(f, 16*1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	var stat unix.Stat_t
	err = unix.Stat(f.Name(), &stat)
	if err != nil {
		t.Fatal(err)
	}

	if stat.Blocks*512 != 16*1024*1024 {
		t.Fatalf("got %d, want %d", stat.Blocks*512, 16*1024*1024)
	}
}
