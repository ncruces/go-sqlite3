package sqlite3

import (
	"context"
	"os"

	"github.com/tetratelabs/wazero/api"
)

func (vfsFileMethods) NewID(ctx context.Context, file *os.File) uint32 {
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

func (vfsFileMethods) Open(ctx context.Context, mod api.Module, pFile uint32, file *os.File) {
	mem := memory{mod}
	id := vfsFile.NewID(ctx, file)
	mem.writeUint32(pFile+ptrlen, id)
	mem.writeUint32(pFile+2*ptrlen, _NO_LOCK)
}

func (vfsFileMethods) Close(ctx context.Context, mod api.Module, pFile uint32) error {
	mem := memory{mod}
	id := mem.readUint32(pFile + ptrlen)
	vfs := ctx.Value(vfsKey{}).(*vfsState)
	file := vfs.files[id]
	vfs.files[id] = nil
	return file.Close()
}

func (vfsFileMethods) GetOS(ctx context.Context, mod api.Module, pFile uint32) *os.File {
	mem := memory{mod}
	id := mem.readUint32(pFile + ptrlen)
	vfs := ctx.Value(vfsKey{}).(*vfsState)
	return vfs.files[id]
}

func (vfsFileMethods) GetLock(ctx context.Context, mod api.Module, pFile uint32) vfsLockState {
	mem := memory{mod}
	return vfsLockState(mem.readUint32(pFile + 2*ptrlen))
}

func (vfsFileMethods) SetLock(ctx context.Context, mod api.Module, pFile uint32, lock vfsLockState) {
	mem := memory{mod}
	mem.writeUint32(pFile+2*ptrlen, uint32(lock))
}
