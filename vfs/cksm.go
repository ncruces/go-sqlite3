package vfs

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"strconv"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/util/sql3util"
)

func cksmWrapFile(name *Filename, flags OpenFlag, file File) File {
	// Checksum only main databases and WALs.
	if flags&(OPEN_MAIN_DB|OPEN_WAL) == 0 {
		return file
	}

	cksm := cksmFile{File: file}

	if flags&OPEN_WAL != 0 {
		main, _ := name.DatabaseFile().(*cksmFile)
		cksm.cksmFlags = main.cksmFlags
	} else {
		cksm.cksmFlags = new(cksmFlags)
		cksm.isDB = true
	}

	return &cksm
}

type cksmFile struct {
	File
	*cksmFlags
	isDB bool
}

type cksmFlags struct {
	computeCksm bool
	verifyCksm  bool
	inCkpt      bool
	pageSize    int
}

func (c *cksmFile) ReadAt(p []byte, off int64) (n int, err error) {
	n, err = c.File.ReadAt(p, off)

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
			return 0, _IOERR_DATA
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
	if f, ok := c.File.(FileCheckpoint); ok {
		f.CheckpointStart()
	}
	c.inCkpt = true
}

func (c *cksmFile) CheckpointDone() {
	if f, ok := c.File.(FileCheckpoint); ok {
		f.CheckpointDone()
	}
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
	if f, ok := c.File.(FilePragma); ok {
		return f.Pragma(name, value)
	}
	return "", _NOTFOUND
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

func (c *cksmFile) Unwrap() File {
	return c.File
}

func (c *cksmFile) SharedMemory() SharedMemory {
	if f, ok := c.File.(FileSharedMemory); ok {
		return f.SharedMemory()
	}
	return nil
}

// Wrap optional methods.

func (c *cksmFile) LockState() LockLevel {
	if f, ok := c.File.(FileLockState); ok {
		return f.LockState()
	}
	return LOCK_EXCLUSIVE + 1 // UNKNOWN_LOCK
}

func (c *cksmFile) PersistentWAL() bool {
	if f, ok := c.File.(FilePersistentWAL); ok {
		return f.PersistentWAL()
	}
	return false
}

func (c *cksmFile) SetPersistentWAL(keepWAL bool) {
	if f, ok := c.File.(FilePersistentWAL); ok {
		f.SetPersistentWAL(keepWAL)
	}
}

func (c *cksmFile) PowersafeOverwrite() bool {
	if f, ok := c.File.(FilePowersafeOverwrite); ok {
		return f.PowersafeOverwrite()
	}
	return false
}

func (c *cksmFile) SetPowersafeOverwrite(psow bool) {
	if f, ok := c.File.(FilePowersafeOverwrite); ok {
		f.SetPowersafeOverwrite(psow)
	}
}

func (c *cksmFile) ChunkSize(size int) {
	if f, ok := c.File.(FileChunkSize); ok {
		f.ChunkSize(size)
	}
}

func (c *cksmFile) SizeHint(size int64) error {
	if f, ok := c.File.(FileSizeHint); ok {
		f.SizeHint(size)
	}
	return _NOTFOUND
}

func (c *cksmFile) HasMoved() (bool, error) {
	if f, ok := c.File.(FileHasMoved); ok {
		return f.HasMoved()
	}
	return false, _NOTFOUND
}

func (c *cksmFile) Overwrite() error {
	if f, ok := c.File.(FileOverwrite); ok {
		return f.Overwrite()
	}
	return _NOTFOUND
}

func (c *cksmFile) CommitPhaseTwo() error {
	if f, ok := c.File.(FileCommitPhaseTwo); ok {
		return f.CommitPhaseTwo()
	}
	return _NOTFOUND
}

func (c *cksmFile) BeginAtomicWrite() error {
	if f, ok := c.File.(FileBatchAtomicWrite); ok {
		return f.BeginAtomicWrite()
	}
	return _NOTFOUND
}

func (c *cksmFile) CommitAtomicWrite() error {
	if f, ok := c.File.(FileBatchAtomicWrite); ok {
		return f.CommitAtomicWrite()
	}
	return _NOTFOUND
}

func (c *cksmFile) RollbackAtomicWrite() error {
	if f, ok := c.File.(FileBatchAtomicWrite); ok {
		return f.RollbackAtomicWrite()
	}
	return _NOTFOUND
}
