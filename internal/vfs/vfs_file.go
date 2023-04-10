package vfs

import (
	"context"
	"os"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero/api"
)

type vfsFile struct {
	*os.File
	lock        _LockLevel
	lockTimeout time.Duration
	psow        bool
	syncDir     bool
	readOnly    bool
}

func newVFSFile(vfs *vfsState, file *os.File) uint32 {
	// Find an empty slot.
	for id, f := range vfs.files {
		if f.File == nil {
			vfs.files[id] = vfsFile{File: file}
			return uint32(id)
		}
	}

	// Add a new slot.
	vfs.files = append(vfs.files, vfsFile{File: file})
	return uint32(len(vfs.files) - 1)
}

func getVFSFile(ctx context.Context, mod api.Module, pFile uint32) *vfsFile {
	vfs := ctx.Value(vfsKey{}).(*vfsState)
	id := util.ReadUint32(mod, pFile+4)
	return &vfs.files[id]
}

func openVFSFile(ctx context.Context, mod api.Module, pFile uint32, file *os.File) *vfsFile {
	vfs := ctx.Value(vfsKey{}).(*vfsState)
	id := newVFSFile(vfs, file)
	util.WriteUint32(mod, pFile+4, id)
	return &vfs.files[id]
}

func closeVFSFile(ctx context.Context, mod api.Module, pFile uint32) error {
	vfs := ctx.Value(vfsKey{}).(*vfsState)
	id := util.ReadUint32(mod, pFile+4)
	file := vfs.files[id]
	vfs.files[id] = vfsFile{}
	return file.Close()
}
