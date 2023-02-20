package sqlite3

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func Test_vfsLock(t *testing.T) {
	// Other OSes lack open file descriptors locks.
	switch runtime.GOOS {
	case "linux", "darwin", "illumos", "windows":
		//
	default:
		t.Skip()
	}

	name := filepath.Join(t.TempDir(), "test.db")

	// Create a temporary file.
	file1, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		t.Fatal(err)
	}
	defer file1.Close()

	// Open the temporary file again.
	file2, err := os.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer file2.Close()

	// Bypass open file reuse.
	vfsOpenFiles = append(vfsOpenFiles, &vfsOpenFile{
		file:   file1,
		nref:   1,
		locker: vfsFileLocker{file: file1},
	}, &vfsOpenFile{
		file:   file2,
		nref:   1,
		locker: vfsFileLocker{file: file2},
	})

	mem := newMemory(128)
	mem.writeUint32(4+4, 0)
	mem.writeUint32(16+4, 1)

	const (
		pFile1  = 4
		pFile2  = 16
		pOutput = 32
	)

	rc := vfsCheckReservedLock(context.TODO(), mem.mod, pFile1, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readUint32(pOutput); got != 0 {
		t.Error("file was locked")
	}

	rc = vfsLock(context.TODO(), mem.mod, pFile2, _SHARED_LOCK)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	rc = vfsCheckReservedLock(context.TODO(), mem.mod, pFile1, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readUint32(pOutput); got != 0 {
		t.Error("file was locked")
	}

	rc = vfsLock(context.TODO(), mem.mod, pFile2, _RESERVED_LOCK)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	rc = vfsLock(context.TODO(), mem.mod, pFile2, _SHARED_LOCK)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	rc = vfsCheckReservedLock(context.TODO(), mem.mod, pFile1, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readUint32(pOutput); got == 0 {
		t.Error("file wasn't locked")
	}

	rc = vfsLock(context.TODO(), mem.mod, pFile2, _EXCLUSIVE_LOCK)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	rc = vfsCheckReservedLock(context.TODO(), mem.mod, pFile1, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readUint32(pOutput); got == 0 {
		t.Error("file wasn't locked")
	}

	rc = vfsLock(context.TODO(), mem.mod, pFile1, _SHARED_LOCK)
	if rc == _OK {
		t.Fatal("returned", rc)
	}

	rc = vfsUnlock(context.TODO(), mem.mod, pFile2, _SHARED_LOCK)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	rc = vfsLock(context.TODO(), mem.mod, pFile1, _SHARED_LOCK)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
}
