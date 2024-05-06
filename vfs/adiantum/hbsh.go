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
	if hf, ok := h.VFS.(vfs.VFSFilename); ok {
		file, flags, err = hf.OpenFilename(name, flags)
	} else {
		file, flags, err = h.VFS.Open(name.String(), flags)
	}

	// Encrypt everything except super journals and memory files.
	if err != nil || flags&(vfs.OPEN_SUPER_JOURNAL|vfs.OPEN_MEMORY) != 0 {
		return file, flags, err
	}

	var hbsh *hbsh.HBSH
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
		} else if flags&vfs.OPEN_MAIN_DB != 0 {
			// Main datatabases may have their key specified as a PRAGMA.
			return &hbshFile{File: file, reset: h.hbsh}, flags, nil
		}
		hbsh = h.hbsh.HBSH(key)
	}

	if hbsh == nil {
		return nil, flags, sqlite3.CANTOPEN
	}
	return &hbshFile{File: file, hbsh: hbsh, reset: h.hbsh}, flags, nil
}

const (
	tweakSize = 8
	blockSize = 4096
)

type hbshFile struct {
	vfs.File
	hbsh  *hbsh.HBSH
	reset HBSHCreator
	tweak [tweakSize]byte
	block [blockSize]byte
}

func (h *hbshFile) Pragma(name string, value string) (string, error) {
	var key []byte
	switch name {
	case "key":
		key = []byte(value)
	case "hexkey":
		key, _ = hex.DecodeString(value)
	case "textkey":
		key = h.reset.KDF(value)
	default:
		if f, ok := h.File.(vfs.FilePragma); ok {
			return f.Pragma(name, value)
		}
		return "", sqlite3.NOTFOUND
	}

	if h.hbsh = h.reset.HBSH(key); h.hbsh != nil {
		return "ok", nil
	}
	return "", sqlite3.CANTOPEN
}

func (h *hbshFile) ReadAt(p []byte, off int64) (n int, err error) {
	if h.hbsh == nil {
		// Only OPEN_MAIN_DB can have a missing key.
		if off == 0 && len(p) == 100 {
			// SQLite is trying to read the header of a database file.
			// Pretend the file is empty so the key may specified as a PRAGMA.
			return 0, io.EOF
		}
		return 0, sqlite3.CANTOPEN
	}

	min := (off) &^ (blockSize - 1)                                   // round down
	max := (off + int64(len(p)) + (blockSize - 1)) &^ (blockSize - 1) // round up

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
	if h.hbsh == nil {
		return 0, sqlite3.READONLY
	}

	min := (off) &^ (blockSize - 1)                                   // round down
	max := (off + int64(len(p)) + (blockSize - 1)) &^ (blockSize - 1) // round up

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
				// Writing past the EOF.
				// We're either appending an entirely new block,
				// or the final block was only partially written.
				// A partially written block can't be decrypted,
				// and is as good as corrupt.
				// Either way, zero pad the file to the next block size.
				clear(data)
			} else {
				data = h.hbsh.Decrypt(h.block[:], h.tweak[:])
			}
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
	size = (size + (blockSize - 1)) &^ (blockSize - 1) // round up
	return h.File.Truncate(size)
}

func (h *hbshFile) SectorSize() int {
	return lcm(h.File.SectorSize(), blockSize)
}

func (h *hbshFile) DeviceCharacteristics() vfs.DeviceCharacteristic {
	return h.File.DeviceCharacteristics() & (0 |
		// The only safe flags are these:
		vfs.IOCAP_UNDELETABLE_WHEN_OPEN |
		vfs.IOCAP_IMMUTABLE |
		vfs.IOCAP_BATCH_ATOMIC)
}

// Wrap optional methods.

func (h *hbshFile) SharedMemory() vfs.SharedMemory {
	if f, ok := h.File.(vfs.FileSharedMemory); ok {
		return f.SharedMemory()
	}
	return nil
}

func (h *hbshFile) ChunkSize(size int) {
	if f, ok := h.File.(vfs.FileChunkSize); ok {
		size = (size + (blockSize - 1)) &^ (blockSize - 1) // round up
		f.ChunkSize(size)
	}
}

func (h *hbshFile) SizeHint(size int64) error {
	if f, ok := h.File.(vfs.FileSizeHint); ok {
		size = (size + (blockSize - 1)) &^ (blockSize - 1) // round up
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
