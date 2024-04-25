package adiantum

import (
	"encoding/binary"
	"encoding/hex"
	"io"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/vfs"
	"lukechampine.com/adiantum/hbsh"
)

type hbshVFS struct {
	vfs.VFS
	hbsh HBSHCreator
}

func (h *hbshVFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	return nil, 0, sqlite3.CANTOPEN
}

func (h *hbshVFS) OpenFilename(name *vfs.Filename, flags vfs.OpenFlag) (file vfs.File, _ vfs.OpenFlag, err error) {
	var hbsh *hbsh.HBSH

	// Encrypt everything except super journals.
	if flags&vfs.OPEN_SUPER_JOURNAL == 0 {
		if f, ok := name.DatabaseFile().(*hbshFile); ok {
			hbsh = f.hbsh
		} else {
			var key []byte
			if params := name.URIParameters(); name == nil {
				key = h.hbsh.KDF("") // Temporary files get a random key.
			} else if t, ok := params["key"]; ok {
				key = []byte(t[0])
			} else if t, ok := params["hexkey"]; ok {
				key, _ = hex.DecodeString(t[0])
			} else if t, ok := params["textkey"]; ok {
				key = h.hbsh.KDF(t[0])
			}
			if hbsh = h.hbsh.HBSH(key); hbsh == nil {
				// Can't open without a valid key.
				return nil, flags, sqlite3.CANTOPEN
			}
		}
	}

	if h, ok := h.VFS.(vfs.VFSFilename); ok {
		file, flags, err = h.OpenFilename(name, flags)
	} else {
		file, flags, err = h.Open(name.String(), flags)
	}
	if err != nil || hbsh == nil || flags&vfs.OPEN_MEMORY != 0 {
		// Error, or no encryption (super journals, memory files).
		return file, flags, err
	}
	return &hbshFile{File: file, hbsh: hbsh}, flags, err
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

	// Read one block at a time.
	for ; min < max; min += blockSize {
		m, err := h.File.ReadAt(h.block[:], min)
		if m != blockSize {
			return n, err
		}

		binary.LittleEndian.PutUint64(h.tweak[:], uint64(min))
		data := h.hbsh.Decrypt(h.block[:], h.tweak[:])

		if off > min {
			data = data[off-min:]
		}
		n += copy(p[n:], data)
	}

	if n != len(p) {
		panic(util.AssertErr())
	}
	return n, nil
}

func (h *hbshFile) WriteAt(p []byte, off int64) (n int, err error) {
	min := (off) &^ (blockSize - 1)                                 // round down
	max := (off + int64(len(p)) + blockSize - 1) &^ (blockSize - 1) // round up

	// Write one block at a time.
	for ; min < max; min += blockSize {
		binary.LittleEndian.PutUint64(h.tweak[:], uint64(min))
		data := h.block[:]

		if off > min || len(p[n:]) < blockSize {
			// Partial block write: read-update-write.
			m, err := h.File.ReadAt(h.block[:], min)
			if m != blockSize {
				if err != io.EOF {
					return n, err
				}
				// Writing past the EOF:
				// We're either appending an entirely new block,
				// or the final block was only partially written.
				// A partially written block can't be decripted,
				// and is as good as corrupt.
				// Either way, zero pad the file to the next block size.
				clear(data)
			}

			data = h.hbsh.Decrypt(h.block[:], h.tweak[:])
			if off > min {
				data = data[off-min:]
			}
		}

		t := copy(data, p[n:])
		h.hbsh.Encrypt(h.block[:], h.tweak[:])

		m, err := h.File.WriteAt(h.block[:], min)
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
	return h.File.Truncate(size)
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

func (h *hbshFile) SharedMemory() vfs.SharedMemory {
	if shm, ok := h.File.(vfs.FileSharedMemory); ok {
		return shm.SharedMemory()
	}
	return nil
}

// Wrap optional methods.

func (h *hbshFile) ChunkSize(size int) {
	if f, ok := h.File.(vfs.FileChunkSize); ok {
		size = (size + blockSize - 1) &^ (blockSize - 1) // round up
		f.ChunkSize(size)
	}
}

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

func (h *hbshFile) CheckpointDone() error {
	if f, ok := h.File.(vfs.FileCheckpoint); ok {
		return f.CheckpointDone()
	}
	return sqlite3.NOTFOUND
}

func (h *hbshFile) CheckpointStart() error {
	if f, ok := h.File.(vfs.FileCheckpoint); ok {
		return f.CheckpointStart()
	}
	return sqlite3.NOTFOUND
}
