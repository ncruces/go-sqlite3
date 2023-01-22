package sqlite3

import (
	"context"

	"github.com/tetratelabs/wazero/api"
)

func vfsLock(ctx context.Context, pFile, eLock uint32) uint32 {
	return _OK
}

func vfsUnlock(ctx context.Context, pFile, eLock uint32) uint32 {
	return _OK
}

func vfsCheckReservedLock(ctx context.Context, mod api.Module, pFile, pResOut uint32) uint32 {
	mod.Memory().WriteUint32Le(pResOut, 0)
	return _OK
}
