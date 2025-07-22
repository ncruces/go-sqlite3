package xts

import (
	"encoding/hex"
	"io"

	"golang.org/x/crypto/xts"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/util/vfsutil"
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
	file, flags, err = vfsutil.WrapOpenFilename(x.VFS, name, flags)

	// Encrypt everything except super journals and memory files.
	if err != nil || flags&(vfs.OPEN_SUPER_JOURNAL|vfs.OPEN_MEMORY) != 0 {
		return file, flags, err
	}

	var cipher *xts.Cipher
	if f, ok := vfsutil.UnwrapFile[*xtsFile](name.DatabaseFile()); ok {
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
			// Main databases may have their key specified as a PRAGMA.
			return &xtsFile{File: file, init: x.init}, flags, nil
		}
		cipher = x.init.XTS(key)
	}

	if cipher == nil {
		file.Close()
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

// Ensure sectorSize is a power of two.
var _ [0]struct{} = [sectorSize & (sectorSize - 1)]struct{}{}

func roundDown(i int64) int64 {
	return i &^ (sectorSize - 1)
}

func roundUp[T int | int64](i T) T {
	return (i + (sectorSize - 1)) &^ (sectorSize - 1)
}

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
		return vfsutil.WrapPragma(x.File, name, value)
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

	min := roundDown(off)
	max := roundUp(off + int64(len(p)))

	// Read one block at a time.
	for ; min < max; min += sectorSize {
		m, err := x.File.ReadAt(x.sector[:], min)
		if m != sectorSize {
			return n, err
		}

		data := x.sector[:]
		sectorNum := uint64(min / sectorSize)
		x.cipher.Decrypt(data, x.sector[:], sectorNum)

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

	min := roundDown(off)
	max := roundUp(off + int64(len(p)))

	// Write one block at a time.
	for ; min < max; min += sectorSize {
		data := x.sector[:]
		sectorNum := uint64(min / sectorSize)

		if off > min || len(p[n:]) < sectorSize {
			// Partial block write: read-update-write.
			m, err := x.File.ReadAt(x.sector[:], min)
			if m == sectorSize {
				x.cipher.Decrypt(data, x.sector[:], sectorNum)
			} else if err != io.EOF {
				return n, err
			} else {
				// Writing past the EOF.
				// We're either appending an entirely new block,
				// or the final block was only partially written.
				// A partially written block can't be decrypted,
				// and is as good as corrupt.
				// Either way, zero pad the file to the next block size.
				clear(data)
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

func (x *xtsFile) DeviceCharacteristics() vfs.DeviceCharacteristic {
	var _ [0]struct{} = [sectorSize - 512]struct{}{} // Ensure sectorSize is 512.
	return x.File.DeviceCharacteristics() & (0 |
		// These flags are safe:
		vfs.IOCAP_ATOMIC512 |
		vfs.IOCAP_IMMUTABLE |
		vfs.IOCAP_SEQUENTIAL |
		vfs.IOCAP_SUBPAGE_READ |
		vfs.IOCAP_BATCH_ATOMIC |
		vfs.IOCAP_UNDELETABLE_WHEN_OPEN)
}

func (x *xtsFile) SectorSize() int {
	return util.LCM(x.File.SectorSize(), sectorSize)
}

func (x *xtsFile) Truncate(size int64) error {
	return x.File.Truncate(roundUp(size))
}

func (x *xtsFile) ChunkSize(size int) {
	vfsutil.WrapChunkSize(x.File, roundUp(size))
}

func (x *xtsFile) SizeHint(size int64) error {
	return vfsutil.WrapSizeHint(x.File, roundUp(size))
}

// Wrap optional methods.

func (x *xtsFile) Unwrap() vfs.File {
	return x.File // notest
}

func (x *xtsFile) SharedMemory() vfs.SharedMemory {
	return vfsutil.WrapSharedMemory(x.File) // notest
}

func (x *xtsFile) LockState() vfs.LockLevel {
	return vfsutil.WrapLockState(x.File) // notest
}

func (x *xtsFile) PersistentWAL() bool {
	return vfsutil.WrapPersistWAL(x.File) // notest
}

func (x *xtsFile) SetPersistentWAL(keepWAL bool) {
	vfsutil.WrapSetPersistWAL(x.File, keepWAL) // notest
}

func (x *xtsFile) HasMoved() (bool, error) {
	return vfsutil.WrapHasMoved(x.File) // notest
}

func (x *xtsFile) Overwrite() error {
	return vfsutil.WrapOverwrite(x.File) // notest
}

func (x *xtsFile) SyncSuper(super string) error {
	return vfsutil.WrapSyncSuper(x.File, super) // notest
}

func (x *xtsFile) CommitPhaseTwo() error {
	return vfsutil.WrapCommitPhaseTwo(x.File) // notest
}

func (x *xtsFile) BeginAtomicWrite() error {
	return vfsutil.WrapBeginAtomicWrite(x.File) // notest
}

func (x *xtsFile) CommitAtomicWrite() error {
	return vfsutil.WrapCommitAtomicWrite(x.File) // notest
}

func (x *xtsFile) RollbackAtomicWrite() error {
	return vfsutil.WrapRollbackAtomicWrite(x.File) // notest
}

func (x *xtsFile) CheckpointStart() {
	vfsutil.WrapCheckpointStart(x.File) // notest
}

func (x *xtsFile) CheckpointDone() {
	vfsutil.WrapCheckpointDone(x.File) // notest
}

func (x *xtsFile) BusyHandler(handler func() bool) {
	vfsutil.WrapBusyHandler(x.File, handler) // notest
}
