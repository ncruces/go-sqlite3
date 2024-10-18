package cksmvfs

import (
	"github.com/ncruces/go-sqlite3"
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
}

// Wrap optional methods.

func (c *cksmFile) SharedMemory() vfs.SharedMemory {
	if f, ok := c.File.(vfs.FileSharedMemory); ok {
		return f.SharedMemory()
	}
	return nil
}

func (c *cksmFile) ChunkSize(size int) {
	if f, ok := c.File.(vfs.FileChunkSize); ok {
		f.ChunkSize(size)
	}
}

func (c *cksmFile) SizeHint(size int64) error {
	if f, ok := c.File.(vfs.FileSizeHint); ok {
		return f.SizeHint(size)
	}
	return sqlite3.NOTFOUND
}

func (c *cksmFile) HasMoved() (bool, error) {
	if f, ok := c.File.(vfs.FileHasMoved); ok {
		return f.HasMoved()
	}
	return false, sqlite3.NOTFOUND
}

func (c *cksmFile) Overwrite() error {
	if f, ok := c.File.(vfs.FileOverwrite); ok {
		return f.Overwrite()
	}
	return sqlite3.NOTFOUND
}

func (c *cksmFile) PersistentWAL() bool {
	if f, ok := c.File.(vfs.FilePersistentWAL); ok {
		return f.PersistentWAL()
	}
	return false
}

func (c *cksmFile) SetPersistentWAL(keepWAL bool) {
	if f, ok := c.File.(vfs.FilePersistentWAL); ok {
		f.SetPersistentWAL(keepWAL)
	}
}

func (c *cksmFile) PowersafeOverwrite() bool {
	if f, ok := c.File.(vfs.FilePowersafeOverwrite); ok {
		return f.PowersafeOverwrite()
	}
	return false
}

func (c *cksmFile) SetPowersafeOverwrite(psow bool) {
	if f, ok := c.File.(vfs.FilePowersafeOverwrite); ok {
		f.SetPowersafeOverwrite(psow)
	}
}

func (c *cksmFile) CommitPhaseTwo() error {
	if f, ok := c.File.(vfs.FileCommitPhaseTwo); ok {
		return f.CommitPhaseTwo()
	}
	return sqlite3.NOTFOUND
}

func (c *cksmFile) BeginAtomicWrite() error {
	if f, ok := c.File.(vfs.FileBatchAtomicWrite); ok {
		return f.BeginAtomicWrite()
	}
	return sqlite3.NOTFOUND
}

func (c *cksmFile) CommitAtomicWrite() error {
	if f, ok := c.File.(vfs.FileBatchAtomicWrite); ok {
		return f.CommitAtomicWrite()
	}
	return sqlite3.NOTFOUND
}

func (c *cksmFile) RollbackAtomicWrite() error {
	if f, ok := c.File.(vfs.FileBatchAtomicWrite); ok {
		return f.RollbackAtomicWrite()
	}
	return sqlite3.NOTFOUND
}

func (c *cksmFile) CheckpointDone() error {
	if f, ok := c.File.(vfs.FileCheckpoint); ok {
		return f.CheckpointDone()
	}
	return sqlite3.NOTFOUND
}

func (c *cksmFile) CheckpointStart() error {
	if f, ok := c.File.(vfs.FileCheckpoint); ok {
		return f.CheckpointStart()
	}
	return sqlite3.NOTFOUND
}
