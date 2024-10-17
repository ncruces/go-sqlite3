package xts

import (
	"encoding/hex"
	"io"

	"golang.org/x/crypto/xts"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/vfs"
)

type xtsVFS struct {
	vfs.VFS
	init XTSCreator
}

func (x *xtsVFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	// notest // OpenFilename is called instead
	return nil, 0, sqlite3.CANTOPEN
}

func (x *xtsVFS) OpenFilename(name *vfs.Filename, flags vfs.OpenFlag) (file vfs.File, _ vfs.OpenFlag, err error) {
	if hf, ok := x.VFS.(vfs.VFSFilename); ok {
		file, flags, err = hf.OpenFilename(name, flags)
	} else {
		file, flags, err = x.VFS.Open(name.String(), flags)
	}

	// Encrypt everything except super journals and memory files.
	if err != nil || flags&(vfs.OPEN_SUPER_JOURNAL|vfs.OPEN_MEMORY) != 0 {
		return file, flags, err
	}

	var cipher *xts.Cipher
	if f, ok := name.DatabaseFile().(*xtsFile); ok {
		cipher = f.cipher
	} else {
		var key []byte
		if params := name.URIParameters(); name == nil {
			key = x.init.KDF("") // Temporary files get a random key.
		} else if t, ok := params["key"]; ok {
			key = []byte(t[0])
		} else if t, ok := params["hexkey"]; ok {
			key, _ = hex.DecodeString(t[0])
		} else if t, ok := params["textkey"]; ok && len(t[0]) > 0 {
			key = x.init.KDF(t[0])
		} else if flags&vfs.OPEN_MAIN_DB != 0 {
			// Main datatabases may have their key specified as a PRAGMA.
			return &xtsFile{File: file, init: x.init}, flags, nil
		}
		cipher = x.init.XTS(key)
	}

	if cipher == nil {
		return nil, flags, sqlite3.CANTOPEN
	}
	return &xtsFile{File: file, cipher: cipher, init: x.init}, flags, nil
}

// Larger sectors don't seem to significantly improve security,
// and don't affect perfomance.
// https://crossbowerbt.github.io/docs/crypto/pdf00086.pdf
// For flexibility, pick the minimum size of an SQLite page.
// https://sqlite.org/fileformat.html#pages
const sectorSize = 512

type xtsFile struct {
	vfs.File
	init   XTSCreator
	cipher *xts.Cipher
	sector [sectorSize]byte
}

func (x *xtsFile) Pragma(name string, value string) (string, error) {
	var key []byte
	switch name {
	case "key":
		key = []byte(value)
	case "hexkey":
		key, _ = hex.DecodeString(value)
	case "textkey":
		if len(value) > 0 {
			key = x.init.KDF(value)
		}
	default:
		if f, ok := x.File.(vfs.FilePragma); ok {
			return f.Pragma(name, value)
		}
		return "", sqlite3.NOTFOUND
	}

	if x.cipher = x.init.XTS(key); x.cipher != nil {
		return "ok", nil
	}
	return "", sqlite3.CANTOPEN
}

func (x *xtsFile) ReadAt(p []byte, off int64) (n int, err error) {
	if x.cipher == nil {
		// Only OPEN_MAIN_DB can have a missing key.
		if off == 0 && len(p) == 100 {
			// SQLite is trying to read the header of a database file.
			// Pretend the file is empty so the key may be specified as a PRAGMA.
			return 0, io.EOF
		}
		return 0, sqlite3.CANTOPEN
	}

	min := (off) &^ (sectorSize - 1)                                    // round down
	max := (off + int64(len(p)) + (sectorSize - 1)) &^ (sectorSize - 1) // round up

	// Read one block at a time.
	for ; min < max; min += sectorSize {
		m, err := x.File.ReadAt(x.sector[:], min)
		if m != sectorSize {
			return n, err
		}

		sectorNum := uint64(min / sectorSize)
		x.cipher.Decrypt(x.sector[:], x.sector[:], sectorNum)

		data := x.sector[:]
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

func (x *xtsFile) WriteAt(p []byte, off int64) (n int, err error) {
	if x.cipher == nil {
		return 0, sqlite3.READONLY
	}

	min := (off) &^ (sectorSize - 1)                                    // round down
	max := (off + int64(len(p)) + (sectorSize - 1)) &^ (sectorSize - 1) // round up

	// Write one block at a time.
	for ; min < max; min += sectorSize {
		sectorNum := uint64(min / sectorSize)
		data := x.sector[:]

		if off > min || len(p[n:]) < sectorSize {
			// Partial block write: read-update-write.
			m, err := x.File.ReadAt(x.sector[:], min)
			if m != sectorSize {
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
				x.cipher.Decrypt(data, data, sectorNum)
			}
			if off > min {
				data = data[off-min:]
			}
		}

		t := copy(data, p[n:])
		x.cipher.Encrypt(x.sector[:], x.sector[:], sectorNum)

		m, err := x.File.WriteAt(x.sector[:], min)
		if m != sectorSize {
			return n, err
		}
		n += t
	}

	if n != len(p) {
		panic(util.AssertErr())
	}
	return n, nil
}

func (x *xtsFile) Truncate(size int64) error {
	size = (size + (sectorSize - 1)) &^ (sectorSize - 1) // round up
	return x.File.Truncate(size)
}

func (x *xtsFile) SectorSize() int {
	return util.LCM(x.File.SectorSize(), sectorSize)
}

func (x *xtsFile) DeviceCharacteristics() vfs.DeviceCharacteristic {
	return x.File.DeviceCharacteristics() & (0 |
		// The only safe flags are these:
		vfs.IOCAP_UNDELETABLE_WHEN_OPEN |
		vfs.IOCAP_IMMUTABLE |
		vfs.IOCAP_BATCH_ATOMIC)
}

// Wrap optional methods.

func (x *xtsFile) SharedMemory() vfs.SharedMemory {
	if f, ok := x.File.(vfs.FileSharedMemory); ok {
		return f.SharedMemory()
	}
	return nil
}

func (x *xtsFile) ChunkSize(size int) {
	if f, ok := x.File.(vfs.FileChunkSize); ok {
		size = (size + (sectorSize - 1)) &^ (sectorSize - 1) // round up
		f.ChunkSize(size)
	}
}

func (x *xtsFile) SizeHint(size int64) error {
	if f, ok := x.File.(vfs.FileSizeHint); ok {
		size = (size + (sectorSize - 1)) &^ (sectorSize - 1) // round up
		return f.SizeHint(size)
	}
	return sqlite3.NOTFOUND
}

func (x *xtsFile) HasMoved() (bool, error) {
	if f, ok := x.File.(vfs.FileHasMoved); ok {
		return f.HasMoved()
	}
	return false, sqlite3.NOTFOUND
}

func (x *xtsFile) Overwrite() error {
	if f, ok := x.File.(vfs.FileOverwrite); ok {
		return f.Overwrite()
	}
	return sqlite3.NOTFOUND
}

func (x *xtsFile) CommitPhaseTwo() error {
	if f, ok := x.File.(vfs.FileCommitPhaseTwo); ok {
		return f.CommitPhaseTwo()
	}
	return sqlite3.NOTFOUND
}

func (x *xtsFile) BeginAtomicWrite() error {
	if f, ok := x.File.(vfs.FileBatchAtomicWrite); ok {
		return f.BeginAtomicWrite()
	}
	return sqlite3.NOTFOUND
}

func (x *xtsFile) CommitAtomicWrite() error {
	if f, ok := x.File.(vfs.FileBatchAtomicWrite); ok {
		return f.CommitAtomicWrite()
	}
	return sqlite3.NOTFOUND
}

func (x *xtsFile) RollbackAtomicWrite() error {
	if f, ok := x.File.(vfs.FileBatchAtomicWrite); ok {
		return f.RollbackAtomicWrite()
	}
	return sqlite3.NOTFOUND
}

func (x *xtsFile) CheckpointDone() error {
	if f, ok := x.File.(vfs.FileCheckpoint); ok {
		return f.CheckpointDone()
	}
	return sqlite3.NOTFOUND
}

func (x *xtsFile) CheckpointStart() error {
	if f, ok := x.File.(vfs.FileCheckpoint); ok {
		return f.CheckpointStart()
	}
	return sqlite3.NOTFOUND
}
