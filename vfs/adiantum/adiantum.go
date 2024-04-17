package adiantum

import (
	"encoding/hex"
	"net/url"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/vfs"
	"lukechampine.com/adiantum/hbsh"
)

type hbshVFS struct {
	vfs.VFS
	hbsh HBSHCreator
}

func (h *hbshVFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	return h.OpenParams(name, flags, url.Values{})
}

func (h *hbshVFS) OpenParams(name string, flags vfs.OpenFlag, params url.Values) (file vfs.File, _ vfs.OpenFlag, err error) {
	if h, ok := h.VFS.(vfs.VFSParams); ok {
		file, flags, err = h.OpenParams(name, flags, params)
	} else {
		file, flags, err = h.Open(name, flags)
	}
	if err != nil {
		return file, flags, err
	}

	var key []byte
	if t, ok := params["key"]; ok {
		key = []byte(t[0])
	} else if t, ok := params["hexkey"]; ok {
		key, err = hex.DecodeString(t[0])
	} else if t, ok := params["textkey"]; ok {
		key = h.hbsh.KDF(t[0])
	}

	return &hbshFile{file, h.hbsh.HBSH(key)}, flags, err
}

const blockSize = 4096

type hbshFile struct {
	vfs.File
	hbsh *hbsh.HBSH
}

func (m *hbshFile) ReadAt(p []byte, off int64) (n int, err error)
func (m *hbshFile) WriteAt(p []byte, off int64) (n int, err error)

func (m *hbshFile) Truncate(size int64) error {
	size = (size + blockSize - 1) % blockSize
	return m.Truncate(size)
}

func (m *hbshFile) SectorSize() int {
	return max(m.File.SectorSize(), blockSize)
}

func (m *hbshFile) DeviceCharacteristics() vfs.DeviceCharacteristic {
	return m.File.DeviceCharacteristics() & (0 |
		// The only safe flags are these:
		vfs.IOCAP_SEQUENTIAL |
		vfs.IOCAP_UNDELETABLE_WHEN_OPEN |
		vfs.IOCAP_IMMUTABLE |
		vfs.IOCAP_BATCH_ATOMIC)
}

// Wrap optional methods.

func (m *hbshFile) SizeHint(size int64) error {
	if f, ok := m.File.(vfs.FileSizeHint); ok {
		size = (size + blockSize - 1) % blockSize
		return f.SizeHint(size)
	}
	return sqlite3.NOTFOUND
}

func (m *hbshFile) HasMoved() (bool, error) {
	if f, ok := m.File.(vfs.FileHasMoved); ok {
		return f.HasMoved()
	}
	return false, sqlite3.NOTFOUND
}

func (m *hbshFile) Overwrite() error {
	if f, ok := m.File.(vfs.FileOverwrite); ok {
		return f.Overwrite()
	}
	return sqlite3.NOTFOUND
}

func (m *hbshFile) CommitPhaseTwo() error {
	if f, ok := m.File.(vfs.FileCommitPhaseTwo); ok {
		return f.CommitPhaseTwo()
	}
	return sqlite3.NOTFOUND
}

func (m *hbshFile) BeginAtomicWrite() error {
	if f, ok := m.File.(vfs.FileBatchAtomicWrite); ok {
		return f.BeginAtomicWrite()
	}
	return sqlite3.NOTFOUND
}

func (m *hbshFile) CommitAtomicWrite() error {
	if f, ok := m.File.(vfs.FileBatchAtomicWrite); ok {
		return f.CommitAtomicWrite()
	}
	return sqlite3.NOTFOUND
}

func (m *hbshFile) RollbackAtomicWrite() error {
	if f, ok := m.File.(vfs.FileBatchAtomicWrite); ok {
		return f.RollbackAtomicWrite()
	}
	return sqlite3.NOTFOUND
}
