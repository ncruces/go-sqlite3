package sqlite3

import (
	"context"
	"os"
	"runtime"
	"testing"
)

func Test_vfsLock(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "darwin":
		//
	default:
		t.Skip()
	}

	file1, err := os.CreateTemp("", "sqlite3-")
	if err != nil {
		t.Fatal(err)
	}
	defer file1.Close()

	name := file1.Name()
	defer os.RemoveAll(name)

	file2, err := os.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer file2.Close()

	vfsOpenFiles = append(vfsOpenFiles, &vfsOpenFile{
		file:      file1,
		nref:      1,
		vfsLocker: &vfsFileLocker{file1, _NO_LOCK},
	}, &vfsOpenFile{
		file:      file2,
		nref:      1,
		vfsLocker: &vfsFileLocker{file2, _NO_LOCK},
	})

	mem := newMemory(128)
	mem.writeUint32(4+4, 0)
	mem.writeUint32(16+4, 1)

	rc := vfsCheckReservedLock(context.TODO(), mem.mod, 4, 32)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readUint32(16); got != 0 {
		t.Error("file was locked")
	}

	rc = vfsLock(context.TODO(), mem.mod, 16, _SHARED_LOCK)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	rc = vfsCheckReservedLock(context.TODO(), mem.mod, 4, 32)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readUint32(32); got != 0 {
		t.Error("file was locked")
	}

	rc = vfsLock(context.TODO(), mem.mod, 16, _EXCLUSIVE_LOCK)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	rc = vfsCheckReservedLock(context.TODO(), mem.mod, 4, 32)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readUint32(32); got == 0 {
		t.Error("file wasn't locked")
	}
}
