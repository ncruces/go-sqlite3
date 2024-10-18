package cksmvfs

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"io"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
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
	if cf, ok := c.VFS.(vfs.VFSFilename); ok {
		file, flags, err = cf.OpenFilename(name, flags)
	} else {
		file, flags, err = c.VFS.Open(name.String(), flags)
	}

	// Checksum only databases and WALs.
	if err != nil || flags&(vfs.OPEN_MAIN_DB|vfs.OPEN_WAL) == 0 {
		return file, flags, err
	}

	return &cksmFile{File: file}, flags, err
}

type cksmFile struct {
	vfs.File

	partner     *cksmFile
	computeCksm bool
	verifyCksm  bool
	isWAL       bool
	inCkpt      bool
}

func (c *cksmFile) ReadAt(p []byte, off int64) (n int, err error) {
	n, err = c.File.ReadAt(p, off)

	// SQLite is trying to read from the first page of an empty database file.
	// Read from an empty database that had checksums enabled,
	// so checksums are enabled by default.
	if n == 0 && err == io.EOF && off < 100 && !c.isWAL {
		n = copy(p, empty[off:])
		if n < len(p) {
			clear(p[n:])
		}
		err = nil
	}

	// SQLite is trying to read the header of a database file.
	if off == 0 && len(p) >= 100 && bytes.HasPrefix(p, []byte("SQLite format 3\000")) {
		c.setFlags(p[20])
	}

	// Verify the checksum if:
	//   - the size indicates that we are dealing with a complete database page
	//   - checksum verification is enabled
	//   - we are not in the middle of checkpoint
	if len(p) >= 512 && len(p)&(len(p)-1) == 0 &&
		c.verifyCksm && !c.inCkpt {
		cksm1 := cksmCompute(p[:len(p)-8])
		cksm2 := *(*[8]byte)(p[len(p)-8:])
		if cksm1 != cksm2 {
			return 0, sqlite3.IOERR_DATA
		}
	}
	return n, err
}

func (c *cksmFile) WriteAt(p []byte, off int64) (n int, err error) {
	// SQLite is trying to write the first page of a database file.
	if off == 0 && len(p) >= 100 && bytes.HasPrefix(p, []byte("SQLite format 3\000")) {
		c.setFlags(p[20])
	}

	// Compute the checksum if:
	//   - the size is appropriate for a database page
	//   - checksums where ever enabled
	//   - we are not in the middle of checkpoint
	if len(p) >= 512 &&
		c.computeCksm && !c.inCkpt {
		*(*[8]byte)(p[len(p)-8:]) = cksmCompute(p[:len(p)-8])
	}

	return c.File.WriteAt(p, off)
}

func (c *cksmFile) setFlags(reserved uint8) {
	if r := reserved == 8; r != c.computeCksm {
		c.verifyCksm = r
		c.computeCksm = r
		if c.partner != nil {
			c.partner.verifyCksm = r
			c.partner.computeCksm = r
		}
	}
}

//go:embed empty.db
var empty string

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

func (c *cksmFile) CheckpointStart() {
	vfsutil.WrapCheckpointStart(c.File) // notest
}

func (c *cksmFile) CheckpointDone() {
	vfsutil.WrapCheckpointDone(c.File) // notest
}
