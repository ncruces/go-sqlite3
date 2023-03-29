package vfs

import (
	"context"
	"os"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero/api"
)

const (
	// These need to match the offsets asserted in os.c
	vfsFileIDOffset          = 4
	vfsFileLockOffset        = 8
	vfsFileSyncDirOffset     = 10
	vfsFileReadOnlyOffset    = 11
	vfsFileLockTimeoutOffset = 12
)

func newFileID(ctx context.Context, file *os.File) uint32 {
	vfs := ctx.Value(vfsKey{}).(*vfsState)

	// Find an empty slot.
	for id, ptr := range vfs.files {
		if ptr == nil {
			vfs.files[id] = file
			return uint32(id)
		}
	}

	// Add a new slot.
	vfs.files = append(vfs.files, file)
	return uint32(len(vfs.files) - 1)
}

func openFile(ctx context.Context, mod api.Module, pFile uint32, file *os.File) {
	id := newFileID(ctx, file)
	util.WriteUint32(mod, pFile+vfsFileIDOffset, id)
}

func closeFile(ctx context.Context, mod api.Module, pFile uint32) error {
	id := util.ReadUint32(mod, pFile+vfsFileIDOffset)
	vfs := ctx.Value(vfsKey{}).(*vfsState)
	file := vfs.files[id]
	vfs.files[id] = nil
	return file.Close()
}

func getOSFile(ctx context.Context, mod api.Module, pFile uint32) *os.File {
	id := util.ReadUint32(mod, pFile+vfsFileIDOffset)
	vfs := ctx.Value(vfsKey{}).(*vfsState)
	return vfs.files[id]
}

func getFileLock(ctx context.Context, mod api.Module, pFile uint32) vfsLockState {
	return vfsLockState(util.ReadUint8(mod, pFile+vfsFileLockOffset))
}

func setFileLock(ctx context.Context, mod api.Module, pFile uint32, lock vfsLockState) {
	util.WriteUint8(mod, pFile+vfsFileLockOffset, uint8(lock))
}

func getFileLockTimeout(ctx context.Context, mod api.Module, pFile uint32) time.Duration {
	return time.Duration(util.ReadUint32(mod, pFile+vfsFileLockTimeoutOffset)) * time.Millisecond
}

func getFileSyncDir(ctx context.Context, mod api.Module, pFile uint32) bool {
	return util.ReadBool8(mod, pFile+vfsFileSyncDirOffset)
}

func setFileSyncDir(ctx context.Context, mod api.Module, pFile uint32, val bool) {
	util.WriteBool8(mod, pFile+vfsFileSyncDirOffset, val)
}

func getFileReadOnly(ctx context.Context, mod api.Module, pFile uint32) bool {
	return util.ReadBool8(mod, pFile+vfsFileReadOnlyOffset)
}

func setFileReadOnly(ctx context.Context, mod api.Module, pFile uint32, val bool) {
	util.WriteBool8(mod, pFile+vfsFileReadOnlyOffset, val)
}
