package adiantum

import (
	"encoding/binary"
	"encoding/hex"
	"net/url"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/vfs"
	"lukechampine.com/adiantum/hbsh"
)

// HBSHCreator creates an [hbsh.HBSH] cipher,
// given key material.
type HBSHCreator interface {
	// KDF maps a secret (text) to a key of the appropriate size.
	KDF(text string) (key []byte)

	// HBSH creates an HBSH cipher given an appropriate key.
	HBSH(key []byte) *hbsh.HBSH
}

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
	if err != nil || flags&(0|
		vfs.OPEN_MAIN_DB|
		vfs.OPEN_MAIN_JOURNAL|
		vfs.OPEN_SUBJOURNAL|
		vfs.OPEN_WAL) == 0 {
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

	return &hbshFile{File: file, hbsh: h.hbsh.HBSH(key)}, flags, err
}

const (
	blockSize = 4096
	tweakSize = 8
)

type hbshFile struct {
	vfs.File
	hbsh  *hbsh.HBSH
	block [blockSize]byte
	tweak [tweakSize]byte
}

func (h *hbshFile) ReadAt(p []byte, off int64) (n int, err error) {
	min := (off) &^ (blockSize - 1)                                 // round down
	max := (off + int64(len(p)) + blockSize - 1) &^ (blockSize - 1) // round up

	for ; min < max; min += blockSize {
		block := h.block[:]
		tweak := h.tweak[:]

		// Read full block.
		m, err := h.File.ReadAt(block, min)
		if m != blockSize {
			return n, err
		}

		binary.LittleEndian.PutUint64(tweak, uint64(min))
		h.hbsh.Decrypt(block, tweak)

		if off > min {
			block = block[off-min:]
		}
		n += copy(p[n:], block)
	}

	if n != len(p) {
		panic(util.AssertErr())
	}
	return n, nil
}

func (h *hbshFile) WriteAt(p []byte, off int64) (n int, err error) {
	min := (off) &^ (blockSize - 1)                                 // round down
	max := (off + int64(len(p)) + blockSize - 1) &^ (blockSize - 1) // round up

	// TODO: this is broken.
	// Writing is *also* a partial update if p is too small.

	for ; min < off; min += blockSize {
		block := h.block[:]
		tweak := h.tweak[:]

		// Read full block.
		m, err := h.File.ReadAt(block, min)
		if m != blockSize {
			return n, err
		}

		// Partial update.
		binary.LittleEndian.PutUint64(tweak, uint64(min))
		h.hbsh.Decrypt(block, tweak)
		t := copy(h.block[off-min:], p[n:])
		h.hbsh.Encrypt(block, tweak)

		// Write full block.
		m, err = h.File.WriteAt(h.block[:], min)
		if m != blockSize {
			return n, err
		}
		n += t
	}
	for ; min+blockSize <= max; min += blockSize {
		block := h.block[:]
		tweak := h.tweak[:]

		binary.LittleEndian.PutUint64(tweak, uint64(min))
		t := copy(h.block[:], p[n:])
		h.hbsh.Encrypt(block, tweak)

		// Write full block.
		m, err := h.File.WriteAt(h.block[:], min)
		if m != blockSize {
			return n, err
		}
		n += t
	}
	for ; min < max; min += blockSize {
		block := h.block[:]
		tweak := h.tweak[:]

		// Read full block.
		m, err := h.File.ReadAt(block, min)
		if m != blockSize {
			return n, err
		}

		// Partial update.
		binary.LittleEndian.PutUint64(tweak, uint64(min))
		h.hbsh.Decrypt(block, tweak)
		t := copy(h.block[:], p[n:])
		h.hbsh.Encrypt(block, tweak)

		// Write full block.
		m, err = h.File.WriteAt(h.block[:], min)
		if m != blockSize {
			return n, err
		}
		n += t
	}

	if n != len(p) {
		panic(util.AssertErr())
	}
	return n, nil
}

func (h *hbshFile) Truncate(size int64) error {
	size = (size + blockSize - 1) &^ (blockSize - 1) // round up
	return h.Truncate(size)
}

func (h *hbshFile) SectorSize() int {
	return max(h.File.SectorSize(), blockSize)
}

func (h *hbshFile) DeviceCharacteristics() vfs.DeviceCharacteristic {
	return h.File.DeviceCharacteristics() & (0 |
		// The only safe flags are these:
		vfs.IOCAP_UNDELETABLE_WHEN_OPEN |
		vfs.IOCAP_IMMUTABLE |
		vfs.IOCAP_BATCH_ATOMIC)
}

// Wrap optional methods.

func (h *hbshFile) SizeHint(size int64) error {
	if f, ok := h.File.(vfs.FileSizeHint); ok {
		size = (size + blockSize - 1) &^ (blockSize - 1) // round up
		return f.SizeHint(size)
	}
	return sqlite3.NOTFOUND
}

func (h *hbshFile) HasMoved() (bool, error) {
	if f, ok := h.File.(vfs.FileHasMoved); ok {
		return f.HasMoved()
	}
	return false, sqlite3.NOTFOUND
}

func (h *hbshFile) Overwrite() error {
	if f, ok := h.File.(vfs.FileOverwrite); ok {
		return f.Overwrite()
	}
	return sqlite3.NOTFOUND
}

func (h *hbshFile) CommitPhaseTwo() error {
	if f, ok := h.File.(vfs.FileCommitPhaseTwo); ok {
		return f.CommitPhaseTwo()
	}
	return sqlite3.NOTFOUND
}

func (h *hbshFile) BeginAtomicWrite() error {
	if f, ok := h.File.(vfs.FileBatchAtomicWrite); ok {
		return f.BeginAtomicWrite()
	}
	return sqlite3.NOTFOUND
}

func (h *hbshFile) CommitAtomicWrite() error {
	if f, ok := h.File.(vfs.FileBatchAtomicWrite); ok {
		return f.CommitAtomicWrite()
	}
	return sqlite3.NOTFOUND
}

func (h *hbshFile) RollbackAtomicWrite() error {
	if f, ok := h.File.(vfs.FileBatchAtomicWrite); ok {
		return f.RollbackAtomicWrite()
	}
	return sqlite3.NOTFOUND
}
