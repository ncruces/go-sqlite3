package sqlite3vfs

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero/experimental/wazerotest"
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
	mod := wazerotest.NewModule(wazerotest.NewMemory(wazerotest.PageSize))
	ctx, vfs := NewContext(context.TODO())
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
	rc = vfsFileControl(ctx, mod, pFile2, _FCNTL_LOCKSTATE, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, pOutput); got != uint32(LOCK_NONE) {
		t.Error("invalid lock state", got)
	}

	rc = vfsLock(ctx, mod, pFile2, LOCK_SHARED)
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
	rc = vfsFileControl(ctx, mod, pFile2, _FCNTL_LOCKSTATE, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, pOutput); got != uint32(LOCK_SHARED) {
		t.Error("invalid lock state", got)
	}

	rc = vfsLock(ctx, mod, pFile2, LOCK_RESERVED)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	rc = vfsLock(ctx, mod, pFile2, LOCK_SHARED)
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
	rc = vfsFileControl(ctx, mod, pFile2, _FCNTL_LOCKSTATE, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, pOutput); got != uint32(LOCK_RESERVED) {
		t.Error("invalid lock state", got)
	}

	rc = vfsLock(ctx, mod, pFile2, LOCK_EXCLUSIVE)
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
	rc = vfsFileControl(ctx, mod, pFile2, _FCNTL_LOCKSTATE, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, pOutput); got != uint32(LOCK_EXCLUSIVE) {
		t.Error("invalid lock state", got)
	}

	rc = vfsLock(ctx, mod, pFile1, LOCK_SHARED)
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
	rc = vfsFileControl(ctx, mod, pFile1, _FCNTL_LOCKSTATE, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, pOutput); got != uint32(LOCK_NONE) {
		t.Error("invalid lock state", got)
	}

	rc = vfsUnlock(ctx, mod, pFile2, LOCK_SHARED)
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

	rc = vfsLock(ctx, mod, pFile1, LOCK_SHARED)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	rc = vfsFileControl(ctx, mod, pFile1, _FCNTL_LOCKSTATE, pOutput)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
	if got := util.ReadUint32(mod, pOutput); got != uint32(LOCK_SHARED) {
		t.Error("invalid lock state", got)
	}

	rc = vfsFileControl(ctx, mod, pFile1, _FCNTL_LOCK_TIMEOUT, 1)
	if rc != _OK {
		t.Fatal("returned", rc)
	}
}
