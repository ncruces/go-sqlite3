// Package vfsutil implements virtual filesystem utilities.
package vfsutil

import (
	"testing"

	"github.com/ncruces/go-sqlite3/vfs"
)

func TestWrapOpen(t *testing.T) {
	called := 0

	WrapOpen(mockVFS{open: func(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
		called++
		return nil, flags, nil
	}}, "", 0)

	if called != 1 {
		t.Error("open not called")
	}

	WrapOpenFilename(mockVFS{open: func(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
		called++
		return nil, flags, nil
	}}, nil, 0)

	if called != 2 {
		t.Error("open not called")
	}
}

func TestWrapOpenFilename(t *testing.T) {
	called := 0

	WrapOpen(mockVFSFilename{openFilename: func(name *vfs.Filename, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
		called++
		return nil, flags, nil
	}}, "", 0)

	if called != 1 {
		t.Error("openFilename not called")
	}

	WrapOpenFilename(mockVFSFilename{openFilename: func(name *vfs.Filename, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
		called++
		return nil, flags, nil
	}}, nil, 0)

	if called != 2 {
		t.Error("openFilename not called")
	}
}

func TestWrapLockState(t *testing.T) {
	called := 0

	WrapLockState(mockFile{lockState: func() vfs.LockLevel {
		called++
		return 0
	}})

	if called != 1 {
		t.Error("lockState not called")
	}
}

func TestWrapPersistWAL(t *testing.T) {
	persist := false
	WrapSetPersistWAL(mockFile{setPersistWAL: func(b bool) { persist = b }}, true)
	if !persist {
		t.Error("setPersistWAL not called")
	}

	called := 0
	WrapPersistWAL(mockFile{persistWAL: func() bool { called++; return persist }})
	if !persist {
		t.Error("persistWAL not called")
	}
	if called != 1 {
	}
}

func TestWrapPowersafeOverwrite(t *testing.T) {
	persist := false
	WrapSetPowersafeOverwrite(mockFile{setPowersafeOverwrite: func(b bool) { persist = b }}, true)
	if !persist {
		t.Error("setPowersafeOverwrite not called")
	}

	called := 0
	WrapPowersafeOverwrite(mockFile{powersafeOverwrite: func() bool { called++; return persist }})
	if !persist {
		t.Error("powersafeOverwrite not called")
	}
	if called != 1 {
	}
}

func TestWrapChunkSize(t *testing.T) {
	var chunk int

	WrapChunkSize(mockFile{chunkSize: func(size int) {
		chunk = size
	}}, 5)

	if chunk != 5 {
		t.Error("chunkSize not called")
	}
}

func TestWrapSizeHint(t *testing.T) {
	var hint int64

	WrapSizeHint(mockFile{sizeHint: func(size int64) error {
		hint = size
		return nil
	}}, 5)

	if hint != 5 {
		t.Error("sizeHint not called")
	}
}

func TestWrapHasMoved(t *testing.T) {
	called := 0

	WrapHasMoved(mockFile{hasMoved: func() (bool, error) {
		called++
		return false, nil
	}})

	if called != 1 {
		t.Error("hasMoved not called")
	}
}

func TestWrapOverwrite(t *testing.T) {
	called := 0

	WrapOverwrite(mockFile{overwrite: func() error {
		called++
		return nil
	}})

	if called != 1 {
		t.Error("overwrite not called")
	}
}

func TestWrapSyncSuper(t *testing.T) {
	called := 0

	WrapSyncSuper(mockFile{syncSuper: func(super string) error {
		called++
		return nil
	}}, "")

	if called != 1 {
		t.Error("syncSuper not called")
	}
}

func TestWrapCommitPhaseTwo(t *testing.T) {
	called := 0

	WrapCommitPhaseTwo(mockFile{commitPhaseTwo: func() error {
		called++
		return nil
	}})

	if called != 1 {
		t.Error("commitPhaseTwo not called")
	}
}

func TestWrapBatchAtomicWrite(t *testing.T) {
	calledBegin := 0
	calledCommit := 0
	calledRollback := 0

	f := mockFile{
		begin:    func() error { calledBegin++; return nil },
		commit:   func() error { calledCommit++; return nil },
		rollback: func() error { calledRollback++; return nil },
	}
	WrapBeginAtomicWrite(f)
	WrapCommitAtomicWrite(f)
	WrapRollbackAtomicWrite(f)

	if calledBegin != 1 {
		t.Error("beginAtomicWrite not called")
	}
	if calledCommit != 1 {
		t.Error("commitAtomicWrite not called")
	}
	if calledRollback != 1 {
		t.Error("rollbackAtomicWrite not called")
	}
}

func TestWrapCheckpoint(t *testing.T) {
	calledStart := 0
	calledDone := 0

	f := mockFile{
		ckptStart: func() { calledStart++ },
		ckptDone:  func() { calledDone++ },
	}
	WrapCheckpointStart(f)
	WrapCheckpointDone(f)

	if calledStart != 1 {
		t.Error("checkpointStart not called")
	}
	if calledDone != 1 {
		t.Error("checkpointDone not called")
	}
}

func TestWrapPragma(t *testing.T) {
	called := 0

	val, err := WrapPragma(mockFile{
		pragma: func(name, value string) (string, error) {
			called++
			if name != "foo" || value != "bar" {
				t.Error("wrong pragma arguments")
			}
			return "baz", nil
		},
	}, "foo", "bar")

	if called != 1 {
		t.Error("pragma not called")
	}
	if err != nil {
		t.Error(err)
	}
	if val != "baz" {
		t.Error("unexpected pragma return value")
	}
}

func TestWrapBusyHandler(t *testing.T) {
	called := 0

	WrapBusyHandler(mockFile{
		busyHandler: func(handler func() bool) {
			handler()
			called++
		},
	}, func() bool { return true })

	if called != 1 {
		t.Error("busyHandler not called")
	}
}

type mockVFS struct {
	open func(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error)
}

func (m mockVFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	return m.open(name, flags)
}

func (m mockVFS) Delete(name string, syncDir bool) error                 { panic("unimplemented") }
func (m mockVFS) FullPathname(name string) (string, error)               { panic("unimplemented") }
func (m mockVFS) Access(name string, flags vfs.AccessFlag) (bool, error) { panic("unimplemented") }

type mockVFSFilename struct {
	mockVFS
	openFilename func(name *vfs.Filename, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error)
}

func (m mockVFSFilename) OpenFilename(name *vfs.Filename, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	return m.openFilename(name, flags)
}

type mockFile struct {
	lockState             func() vfs.LockLevel
	persistWAL            func() bool
	setPersistWAL         func(bool)
	powersafeOverwrite    func() bool
	setPowersafeOverwrite func(bool)
	chunkSize             func(int)
	sizeHint              func(int64) error
	hasMoved              func() (bool, error)
	overwrite             func() error
	syncSuper             func(super string) error
	commitPhaseTwo        func() error
	begin                 func() error
	commit                func() error
	rollback              func() error
	ckptStart             func()
	ckptDone              func()
	busyHandler           func(func() bool)
	pragma                func(name, value string) (string, error)
}

func (m mockFile) LockState() vfs.LockLevel           { return m.lockState() }
func (m mockFile) PersistWAL() bool                   { return m.persistWAL() }
func (m mockFile) SetPersistWAL(v bool)               { m.setPersistWAL(v) }
func (m mockFile) PowersafeOverwrite() bool           { return m.powersafeOverwrite() }
func (m mockFile) SetPowersafeOverwrite(v bool)       { m.setPowersafeOverwrite(v) }
func (m mockFile) ChunkSize(s int)                    { m.chunkSize(s) }
func (m mockFile) SizeHint(s int64) error             { return m.sizeHint(s) }
func (m mockFile) HasMoved() (bool, error)            { return m.hasMoved() }
func (m mockFile) Overwrite() error                   { return m.overwrite() }
func (m mockFile) SyncSuper(s string) error           { return m.syncSuper(s) }
func (m mockFile) CommitPhaseTwo() error              { return m.commitPhaseTwo() }
func (m mockFile) BeginAtomicWrite() error            { return m.begin() }
func (m mockFile) CommitAtomicWrite() error           { return m.commit() }
func (m mockFile) RollbackAtomicWrite() error         { return m.rollback() }
func (m mockFile) CheckpointStart()                   { m.ckptStart() }
func (m mockFile) CheckpointDone()                    { m.ckptDone() }
func (m mockFile) BusyHandler(f func() bool)          { m.busyHandler(f) }
func (m mockFile) Pragma(n, v string) (string, error) { return m.pragma(n, v) }

func (m mockFile) Close() error                                    { panic("unimplemented") }
func (m mockFile) ReadAt(p []byte, off int64) (n int, err error)   { panic("unimplemented") }
func (m mockFile) WriteAt(p []byte, off int64) (n int, err error)  { panic("unimplemented") }
func (m mockFile) Truncate(size int64) error                       { panic("unimplemented") }
func (m mockFile) Sync(flags vfs.SyncFlag) error                   { panic("unimplemented") }
func (m mockFile) Size() (int64, error)                            { panic("unimplemented") }
func (m mockFile) Lock(lock vfs.LockLevel) error                   { panic("unimplemented") }
func (m mockFile) Unlock(lock vfs.LockLevel) error                 { panic("unimplemented") }
func (m mockFile) CheckReservedLock() (bool, error)                { panic("unimplemented") }
func (m mockFile) SectorSize() int                                 { panic("unimplemented") }
func (m mockFile) DeviceCharacteristics() vfs.DeviceCharacteristic { panic("unimplemented") }
