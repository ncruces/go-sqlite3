package vfs

import "github.com/ncruces/go-sqlite3/internal/util"

type vfsShm interface {
	ShmMap() error
	ShmLock() error
	ShmUnmap()
}

func (f *vfsFile) ShmMap() error {
	return util.ValueErr
}

func (f *vfsFile) ShmLock() error {
	return util.ValueErr
}

func (f *vfsFile) ShmUnmap() {}
