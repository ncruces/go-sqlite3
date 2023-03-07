package sqlite3

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func Test_vfsLock(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "darwin", "illumos", "windows":
		break
	default:
		t.Skip("OS lacks OFD locks")
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

	const (
		pFile1  = 4
		pFile2  = 16
		pOutput = 32
	)
	mem := newMemory(128)
	ctx, vfs := vfsContext(context.TODO())
	defer vfs.Close()

	vfsFile.Open(ctx, mem.mod, pFile1, file1)
	vfsFile.Open(ctx, mem.mod, pFile2, file2)

	rc := vfsCheckReservedLock(ctx, mem.mod, pFile1, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readUint32(pOutput); got != 0 {
		t.Error("file was locked")
	}

	rc = vfsLock(ctx, mem.mod, pFile2, _SHARED_LOCK)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	rc = vfsCheckReservedLock(ctx, mem.mod, pFile1, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readUint32(pOutput); got != 0 {
		t.Error("file was locked")
	}

	rc = vfsLock(ctx, mem.mod, pFile2, _RESERVED_LOCK)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	rc = vfsLock(ctx, mem.mod, pFile2, _SHARED_LOCK)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	rc = vfsCheckReservedLock(ctx, mem.mod, pFile1, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readUint32(pOutput); got == 0 {
		t.Error("file wasn't locked")
	}

	rc = vfsLock(ctx, mem.mod, pFile2, _EXCLUSIVE_LOCK)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	rc = vfsCheckReservedLock(ctx, mem.mod, pFile1, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readUint32(pOutput); got == 0 {
		t.Error("file wasn't locked")
	}

	rc = vfsLock(ctx, mem.mod, pFile1, _SHARED_LOCK)
	if rc == _OK {
		t.Fatal("returned", rc)
	}

	rc = vfsCheckReservedLock(ctx, mem.mod, pFile1, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readUint32(pOutput); got == 0 {
		t.Error("file wasn't locked")
	}

	rc = vfsUnlock(ctx, mem.mod, pFile2, _SHARED_LOCK)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	rc = vfsCheckReservedLock(ctx, mem.mod, pFile1, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := mem.readUint32(pOutput); got != 0 {
		t.Error("file was locked")
	}

	rc = vfsLock(ctx, mem.mod, pFile1, _SHARED_LOCK)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
}
