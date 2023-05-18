package vfs

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ncruces/go-sqlite3/internal/util"
)

func Test_vfsLock(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "darwin", "windows":
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
	mod := util.NewMockModule(128)
	ctx, vfs := Context(context.TODO())
	defer vfs.Close()

	vfsFileRegister(ctx, mod, pFile1, &vfsFile{File: file1})
	vfsFileRegister(ctx, mod, pFile2, &vfsFile{File: file2})

	rc := vfsCheckReservedLock(ctx, mod, pFile1, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, pOutput); got != 0 {
		t.Error("file was locked")
	}
	rc = vfsCheckReservedLock(ctx, mod, pFile2, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, pOutput); got != 0 {
		t.Error("file was locked")
	}

	rc = vfsLock(ctx, mod, pFile2, _LOCK_SHARED)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	rc = vfsCheckReservedLock(ctx, mod, pFile1, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, pOutput); got != 0 {
		t.Error("file was locked")
	}
	rc = vfsCheckReservedLock(ctx, mod, pFile2, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, pOutput); got != 0 {
		t.Error("file was locked")
	}

	rc = vfsLock(ctx, mod, pFile2, _LOCK_RESERVED)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	rc = vfsLock(ctx, mod, pFile2, _LOCK_SHARED)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	rc = vfsCheckReservedLock(ctx, mod, pFile1, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, pOutput); got == 0 {
		t.Error("file wasn't locked")
	}
	rc = vfsCheckReservedLock(ctx, mod, pFile2, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, pOutput); got == 0 {
		t.Error("file wasn't locked")
	}

	rc = vfsLock(ctx, mod, pFile2, _LOCK_EXCLUSIVE)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	rc = vfsCheckReservedLock(ctx, mod, pFile1, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, pOutput); got == 0 {
		t.Error("file wasn't locked")
	}
	rc = vfsCheckReservedLock(ctx, mod, pFile2, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, pOutput); got == 0 {
		t.Error("file wasn't locked")
	}

	rc = vfsLock(ctx, mod, pFile1, _LOCK_SHARED)
	if rc == _OK {
		t.Fatal("returned", rc)
	}

	rc = vfsCheckReservedLock(ctx, mod, pFile1, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, pOutput); got == 0 {
		t.Error("file wasn't locked")
	}
	rc = vfsCheckReservedLock(ctx, mod, pFile2, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, pOutput); got == 0 {
		t.Error("file wasn't locked")
	}

	rc = vfsUnlock(ctx, mod, pFile2, _LOCK_SHARED)
	if rc != _OK {
		t.Fatal("returned", rc)
	}

	rc = vfsCheckReservedLock(ctx, mod, pFile1, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, pOutput); got != 0 {
		t.Error("file was locked")
	}
	rc = vfsCheckReservedLock(ctx, mod, pFile2, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, pOutput); got != 0 {
		t.Error("file was locked")
	}

	rc = vfsLock(ctx, mod, pFile1, _LOCK_SHARED)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
}
