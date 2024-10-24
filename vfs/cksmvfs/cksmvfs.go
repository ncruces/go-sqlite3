package cksmvfs

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"io"
	"runtime"
	"strconv"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/util/sql3util"
	"github.com/ncruces/go-sqlite3/util/vfsutil"
	"github.com/ncruces/go-sqlite3/vfs"
)

type cksmVFS struct {
	vfs.VFS
}

func (c *cksmVFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	// notest // OpenFilename is called instead
	return nil, 0, sqlite3.CANTOPEN
}

func (c *cksmVFS) OpenFilename(name *vfs.Filename, flags vfs.OpenFlag) (file vfs.File, _ vfs.OpenFlag, err error) {
	// Prevent accidental wrapping.
	if pc, _, _, ok := runtime.Caller(1); ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			if fn.Name() != "github.com/ncruces/go-sqlite3/vfs.vfsOpen" {
				return nil, 0, sqlite3.CANTOPEN
			}
		}
	}

	file, flags, err = vfsutil.WrapOpenFilename(c.VFS, name, flags)

	// Checksum only main databases and WALs.
	if err != nil || flags&(vfs.OPEN_MAIN_DB|vfs.OPEN_WAL) == 0 {
		return file, flags, err
	}

	cksm := cksmFile{File: file}

	if flags&vfs.OPEN_WAL != 0 {
		main, _ := name.DatabaseFile().(*cksmFile)
		cksm.cksmFlags = main.cksmFlags
	} else {
		cksm.isDB = true
		cksm.cksmFlags = new(cksmFlags)
	}

	return &cksm, flags, err
}

type cksmFile struct {
	vfs.File
	*cksmFlags
	isDB bool
}

type cksmFlags struct {
	computeCksm bool
	verifyCksm  bool
	inCkpt      bool
	pageSize    int
}

//go:embed empty.db
var empty string

func (c *cksmFile) ReadAt(p []byte, off int64) (n int, err error) {
	n, err = c.File.ReadAt(p, off)

	// SQLite is trying to read from the first page of an empty database file.
	// Instead, read from an empty database that had checksums enabled,
	// so checksums are enabled by default.
	if c.isDB && n == 0 && err == io.EOF && off < 100 {
		n = copy(p, empty[off:])
		if n < len(p) {
			clear(p[n:])
		}
		err = nil
	}

	// SQLite is reading the header of a database file.
	if c.isDB && off == 0 && len(p) >= 100 &&
		bytes.HasPrefix(p, []byte("SQLite format 3\000")) {
		c.updateFlags(p)
	}

	// Verify checksums.
	if c.verifyCksm && !c.inCkpt && len(p) == c.pageSize {
		cksm1 := cksmCompute(p[:len(p)-8])
		cksm2 := *(*[8]byte)(p[len(p)-8:])
		if cksm1 != cksm2 {
			return 0, sqlite3.IOERR_DATA
		}
	}
	return n, err
}

func (c *cksmFile) WriteAt(p []byte, off int64) (n int, err error) {
	// SQLite is writing the first page of a database file.
	if c.isDB && off == 0 && len(p) >= 100 &&
		bytes.HasPrefix(p, []byte("SQLite format 3\000")) {
		c.updateFlags(p)
	}

	// Compute checksums.
	if c.computeCksm && !c.inCkpt && len(p) == c.pageSize {
		*(*[8]byte)(p[len(p)-8:]) = cksmCompute(p[:len(p)-8])
	}

	return c.File.WriteAt(p, off)
}

func (c *cksmFile) updateFlags(header []byte) {
	c.pageSize = 256 * int(binary.LittleEndian.Uint16(header[16:18]))
	if r := header[20] == 8; r != c.computeCksm {
		c.computeCksm = r
		c.verifyCksm = r
	}
}

func (c *cksmFile) CheckpointStart() {
	c.inCkpt = true
}

func (c *cksmFile) CheckpointDone() {
	c.inCkpt = false
}

func (c *cksmFile) Pragma(name string, value string) (string, error) {
	switch name {
	case "checksum_verification":
		b, ok := sql3util.ParseBool(value)
		if ok {
			c.verifyCksm = b && c.computeCksm
		}
		if !c.verifyCksm {
			return "0", nil
		}
		return "1", nil

	case "page_size":
		if c.computeCksm {
			// Do not allow page size changes on a checksum database.
			return strconv.Itoa(c.pageSize), nil
		}
	}
	return vfsutil.WrapPragma(c.File, name, value)
}

func cksmCompute(a []byte) (cksm [8]byte) {
	var s1, s2 uint32
	for len(a) >= 8 {
		s1 += binary.LittleEndian.Uint32(a[0:4]) + s2
		s2 += binary.LittleEndian.Uint32(a[4:8]) + s1
		a = a[8:]
	}
	if len(a) != 0 {
		panic(util.AssertErr())
	}
	binary.LittleEndian.PutUint32(cksm[0:4], s1)
	binary.LittleEndian.PutUint32(cksm[4:8], s2)
	return
}

func (c *cksmFile) Unwrap() vfs.File {
	return c.File
}

func (c *cksmFile) SharedMemory() vfs.SharedMemory {
	return vfsutil.WrapSharedMemory(c.File)
}

// Wrap optional methods.

func (c *cksmFile) LockState() vfs.LockLevel {
	return vfsutil.WrapLockState(c.File) // notest
}

func (c *cksmFile) PersistentWAL() bool {
	return vfsutil.WrapPersistentWAL(c.File) // notest
}

func (c *cksmFile) SetPersistentWAL(keepWAL bool) {
	vfsutil.WrapSetPersistentWAL(c.File, keepWAL) // notest
}

func (c *cksmFile) PowersafeOverwrite() bool {
	return vfsutil.WrapPowersafeOverwrite(c.File) // notest
}

func (c *cksmFile) SetPowersafeOverwrite(psow bool) {
	vfsutil.WrapSetPowersafeOverwrite(c.File, psow) // notest
}

func (c *cksmFile) ChunkSize(size int) {
	vfsutil.WrapChunkSize(c.File, size) // notest
}

func (c *cksmFile) SizeHint(size int64) error {
	return vfsutil.WrapSizeHint(c.File, size) // notest
}

func (c *cksmFile) HasMoved() (bool, error) {
	return vfsutil.WrapHasMoved(c.File) // notest
}

func (c *cksmFile) Overwrite() error {
	return vfsutil.WrapOverwrite(c.File) // notest
}

func (c *cksmFile) CommitPhaseTwo() error {
	return vfsutil.WrapCommitPhaseTwo(c.File) // notest
}

func (c *cksmFile) BeginAtomicWrite() error {
	return vfsutil.WrapBeginAtomicWrite(c.File) // notest
}

func (c *cksmFile) CommitAtomicWrite() error {
	return vfsutil.WrapCommitAtomicWrite(c.File) // notest
}

func (c *cksmFile) RollbackAtomicWrite() error {
	return vfsutil.WrapRollbackAtomicWrite(c.File) // notest
}
