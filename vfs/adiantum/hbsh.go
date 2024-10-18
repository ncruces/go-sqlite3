package adiantum

import (
	"encoding/binary"
	"encoding/hex"
	"io"

	"lukechampine.com/adiantum/hbsh"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/util/vfsutil"
	"github.com/ncruces/go-sqlite3/vfs"
)

type hbshVFS struct {
	vfs.VFS
	init HBSHCreator
}

func (h *hbshVFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	// notest // OpenFilename is called instead
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
			key = h.init.KDF("") // Temporary files get a random key.
		} else if t, ok := params["key"]; ok {
			key = []byte(t[0])
		} else if t, ok := params["hexkey"]; ok {
			key, _ = hex.DecodeString(t[0])
		} else if t, ok := params["textkey"]; ok && len(t[0]) > 0 {
			key = h.init.KDF(t[0])
		} else if flags&vfs.OPEN_MAIN_DB != 0 {
			// Main datatabases may have their key specified as a PRAGMA.
			return &hbshFile{File: file, init: h.init}, flags, nil
		}
		hbsh = h.init.HBSH(key)
	}

	if hbsh == nil {
		return nil, flags, sqlite3.CANTOPEN
	}
	return &hbshFile{File: file, hbsh: hbsh, init: h.init}, flags, nil
}

// Larger blocks improve both security (wide-block cipher)
// and throughput (cheap hashes amortize the block cipher's cost).
// Use the default SQLite page size;
// smaller pages pay the cost of unaligned access.
// https://sqlite.org/pgszchng2016.html
const (
	tweakSize = 8
	blockSize = 4096
)

type hbshFile struct {
	vfs.File
	init  HBSHCreator
	hbsh  *hbsh.HBSH
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
		if len(value) > 0 {
			key = h.init.KDF(value)
		}
	default:
		return vfsutil.WrapPragma(h.File, name, value)
	}

	if h.hbsh = h.init.HBSH(key); h.hbsh != nil {
		return "ok", nil
	}
	return "", sqlite3.CANTOPEN
}

func (h *hbshFile) ReadAt(p []byte, off int64) (n int, err error) {
	if h.hbsh == nil {
		// Only OPEN_MAIN_DB can have a missing key.
		if off == 0 && len(p) == 100 {
			// SQLite is trying to read the header of a database file.
			// Pretend the file is empty so the key may be specified as a PRAGMA.
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
	return util.LCM(h.File.SectorSize(), blockSize)
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
	return vfsutil.WrapSharedMemory(h.File)
}

func (h *hbshFile) ChunkSize(size int) {
	size = (size + (blockSize - 1)) &^ (blockSize - 1) // round up
	vfsutil.WrapChunkSize(h.File, size)
}

func (h *hbshFile) SizeHint(size int64) error {
	size = (size + (blockSize - 1)) &^ (blockSize - 1) // round up
	return vfsutil.WrapSizeHint(h.File, size)
}

func (h *hbshFile) HasMoved() (bool, error) {
	return vfsutil.WrapHasMoved(h.File) // notest
}

func (h *hbshFile) Overwrite() error {
	return vfsutil.WrapOverwrite(h.File) // notest
}

func (h *hbshFile) PersistentWAL() bool {
	return vfsutil.WrapPersistentWAL(h.File) // notest
}

func (h *hbshFile) SetPersistentWAL(keepWAL bool) {
	vfsutil.WrapSetPersistentWAL(h.File, keepWAL) // notest
}

func (h *hbshFile) CommitPhaseTwo() error {
	return vfsutil.WrapCommitPhaseTwo(h.File) // notest
}

func (h *hbshFile) BeginAtomicWrite() error {
	return vfsutil.WrapBeginAtomicWrite(h.File) // notest
}

func (h *hbshFile) CommitAtomicWrite() error {
	return vfsutil.WrapCommitAtomicWrite(h.File) // notest
}

func (h *hbshFile) RollbackAtomicWrite() error {
	return vfsutil.WrapRollbackAtomicWrite(h.File) // notest
}

func (h *hbshFile) CheckpointDone() error {
	return vfsutil.WrapCheckpointDone(h.File) // notest
}

func (h *hbshFile) CheckpointStart() error {
	return vfsutil.WrapCheckpointStart(h.File) // notest
}
