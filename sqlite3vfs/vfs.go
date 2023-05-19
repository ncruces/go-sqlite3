package sqlite3vfs

import (
	"context"
	"crypto/rand"
	"io"
	"reflect"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/julianday"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// ExportHostFunctions registers the required VFS host functions
// with the provided env module.
func ExportHostFunctions(env wazero.HostModuleBuilder) wazero.HostModuleBuilder {
	util.ExportFuncII(env, "go_vfs_find", vfsFind)
	util.ExportFuncIIJ(env, "go_localtime", vfsLocaltime)
	util.ExportFuncIIII(env, "go_randomness", vfsRandomness)
	util.ExportFuncIII(env, "go_sleep", vfsSleep)
	util.ExportFuncIII(env, "go_current_time", vfsCurrentTime)
	util.ExportFuncIII(env, "go_current_time_64", vfsCurrentTime64)
	util.ExportFuncIIIII(env, "go_full_pathname", vfsFullPathname)
	util.ExportFuncIIII(env, "go_delete", vfsDelete)
	util.ExportFuncIIIII(env, "go_access", vfsAccess)
	util.ExportFuncIIIIII(env, "go_open", vfsOpen)
	util.ExportFuncII(env, "go_close", vfsClose)
	util.ExportFuncIIIIJ(env, "go_read", vfsRead)
	util.ExportFuncIIIIJ(env, "go_write", vfsWrite)
	util.ExportFuncIIJ(env, "go_truncate", vfsTruncate)
	util.ExportFuncIII(env, "go_sync", vfsSync)
	util.ExportFuncIII(env, "go_file_size", vfsFileSize)
	util.ExportFuncIIII(env, "go_file_control", vfsFileControl)
	util.ExportFuncII(env, "go_sector_size", vfsSectorSize)
	util.ExportFuncII(env, "go_device_characteristics", vfsDeviceCharacteristics)
	util.ExportFuncIII(env, "go_lock", vfsLock)
	util.ExportFuncIII(env, "go_unlock", vfsUnlock)
	util.ExportFuncIII(env, "go_check_reserved_lock", vfsCheckReservedLock)
	return env
}

type vfsKey struct{}
type vfsState struct {
	files []File
}

// NewContext creates a new context to hold [api.Module] specific VFS data.
//
// This context should be passed to any [api.Function] calls that might
// generate VFS host callbacks.
//
// The returned [io.Closer] should be closed after the [api.Module] is closed,
// to release any associated resources.
func NewContext(ctx context.Context) (context.Context, io.Closer) {
	vfs := &vfsState{}
	return context.WithValue(ctx, vfsKey{}, vfs), vfs
}

func (vfs *vfsState) Close() error {
	for _, f := range vfs.files {
		if f != nil {
			f.Close()
		}
	}
	vfs.files = nil
	return nil
}

func vfsFind(ctx context.Context, mod api.Module, zVfsName uint32) uint32 {
	name := util.ReadString(mod, zVfsName, _MAX_STRING)
	if Find(name) != nil {
		return 1
	}
	return 0
}

func vfsLocaltime(ctx context.Context, mod api.Module, pTm uint32, t int64) _ErrorCode {
	tm := time.Unix(t, 0)
	var isdst int
	if tm.IsDST() {
		isdst = 1
	}

	const size = 32 / 8
	// https://pubs.opengroup.org/onlinepubs/7908799/xsh/time.h.html
	util.WriteUint32(mod, pTm+0*size, uint32(tm.Second()))
	util.WriteUint32(mod, pTm+1*size, uint32(tm.Minute()))
	util.WriteUint32(mod, pTm+2*size, uint32(tm.Hour()))
	util.WriteUint32(mod, pTm+3*size, uint32(tm.Day()))
	util.WriteUint32(mod, pTm+4*size, uint32(tm.Month()-time.January))
	util.WriteUint32(mod, pTm+5*size, uint32(tm.Year()-1900))
	util.WriteUint32(mod, pTm+6*size, uint32(tm.Weekday()-time.Sunday))
	util.WriteUint32(mod, pTm+7*size, uint32(tm.YearDay()-1))
	util.WriteUint32(mod, pTm+8*size, uint32(isdst))
	return _OK
}

func vfsRandomness(ctx context.Context, mod api.Module, pVfs, nByte, zByte uint32) uint32 {
	mem := util.View(mod, zByte, uint64(nByte))
	n, _ := rand.Reader.Read(mem)
	return uint32(n)
}

func vfsSleep(ctx context.Context, mod api.Module, pVfs, nMicro uint32) _ErrorCode {
	time.Sleep(time.Duration(nMicro) * time.Microsecond)
	return _OK
}

func vfsCurrentTime(ctx context.Context, mod api.Module, pVfs, prNow uint32) _ErrorCode {
	day := julianday.Float(time.Now())
	util.WriteFloat64(mod, prNow, day)
	return _OK
}

func vfsCurrentTime64(ctx context.Context, mod api.Module, pVfs, piNow uint32) _ErrorCode {
	day, nsec := julianday.Date(time.Now())
	msec := day*86_400_000 + nsec/1_000_000
	util.WriteUint64(mod, piNow, uint64(msec))
	return _OK
}

func vfsFullPathname(ctx context.Context, mod api.Module, pVfs, zRelative, nFull, zFull uint32) _ErrorCode {
	vfs := vfsGet(mod, pVfs)
	path := util.ReadString(mod, zRelative, _MAX_PATHNAME)

	path, err := vfs.FullPathname(path)

	size := uint64(len(path) + 1)
	if size > uint64(nFull) {
		return _CANTOPEN_FULLPATH
	}
	mem := util.View(mod, zFull, size)
	mem[len(path)] = 0
	copy(mem, path)

	return vfsErrorCode(err, _CANTOPEN_FULLPATH)
}

func vfsDelete(ctx context.Context, mod api.Module, pVfs, zPath, syncDir uint32) _ErrorCode {
	vfs := vfsGet(mod, pVfs)
	path := util.ReadString(mod, zPath, _MAX_PATHNAME)

	err := vfs.Delete(path, syncDir != 0)
	return vfsErrorCode(err, _IOERR_DELETE)
}

func vfsAccess(ctx context.Context, mod api.Module, pVfs, zPath uint32, flags AccessFlag, pResOut uint32) _ErrorCode {
	vfs := vfsGet(mod, pVfs)
	path := util.ReadString(mod, zPath, _MAX_PATHNAME)

	ok, err := vfs.Access(path, flags)
	var res uint32
	if ok {
		res = 1
	}

	util.WriteUint32(mod, pResOut, res)
	return vfsErrorCode(err, _IOERR_ACCESS)
}

func vfsOpen(ctx context.Context, mod api.Module, pVfs, zPath, pFile uint32, flags OpenFlag, pOutFlags uint32) _ErrorCode {
	vfs := vfsGet(mod, pVfs)

	var path string
	if zPath != 0 {
		path = util.ReadString(mod, zPath, _MAX_PATHNAME)
	}

	file, flags, err := vfs.Open(path, flags)
	if err != nil {
		return vfsErrorCode(err, _CANTOPEN)
	}

	vfsFileRegister(ctx, mod, pFile, file)
	if pOutFlags != 0 {
		util.WriteUint32(mod, pOutFlags, uint32(flags))
	}
	return _OK
}

func vfsClose(ctx context.Context, mod api.Module, pFile uint32) _ErrorCode {
	err := vfsFileClose(ctx, mod, pFile)
	if err != nil {
		return _IOERR_CLOSE
	}
	return _OK
}

func vfsRead(ctx context.Context, mod api.Module, pFile, zBuf, iAmt uint32, iOfst int64) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile)
	buf := util.View(mod, zBuf, uint64(iAmt))

	n, err := file.ReadAt(buf, iOfst)
	if n == int(iAmt) {
		return _OK
	}
	if n == 0 && err != io.EOF {
		return _IOERR_READ
	}
	for i := range buf[n:] {
		buf[n+i] = 0
	}
	return _IOERR_SHORT_READ
}

func vfsWrite(ctx context.Context, mod api.Module, pFile, zBuf, iAmt uint32, iOfst int64) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile)
	buf := util.View(mod, zBuf, uint64(iAmt))

	_, err := file.WriteAt(buf, iOfst)
	if err != nil {
		return _IOERR_WRITE
	}
	return _OK
}

func vfsTruncate(ctx context.Context, mod api.Module, pFile uint32, nByte int64) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile)
	err := file.Truncate(nByte)
	return vfsErrorCode(err, _IOERR_TRUNCATE)
}

func vfsSync(ctx context.Context, mod api.Module, pFile uint32, flags SyncFlag) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile)
	err := file.Sync(flags)
	return vfsErrorCode(err, _IOERR_FSYNC)
}

func vfsFileSize(ctx context.Context, mod api.Module, pFile, pSize uint32) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile)
	size, err := file.FileSize()
	util.WriteUint64(mod, pSize, uint64(size))
	return vfsErrorCode(err, _IOERR_SEEK)
}

func vfsLock(ctx context.Context, mod api.Module, pFile uint32, eLock LockLevel) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile)
	err := file.Lock(eLock)
	return vfsErrorCode(err, _IOERR_LOCK)
}

func vfsUnlock(ctx context.Context, mod api.Module, pFile uint32, eLock LockLevel) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile)
	err := file.Unlock(eLock)
	return vfsErrorCode(err, _IOERR_UNLOCK)
}

func vfsCheckReservedLock(ctx context.Context, mod api.Module, pFile, pResOut uint32) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile)
	locked, err := file.CheckReservedLock()

	var res uint32
	if locked {
		res = 1
	}

	util.WriteUint32(mod, pResOut, res)
	return vfsErrorCode(err, _IOERR_CHECKRESERVEDLOCK)
}

func vfsFileControl(ctx context.Context, mod api.Module, pFile uint32, op _FcntlOpcode, pArg uint32) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile)

	switch op {
	case _FCNTL_LOCKSTATE:
		if file, ok := file.(FileLockState); ok {
			util.WriteUint32(mod, pArg, uint32(file.LockState()))
			return _OK
		}

	case _FCNTL_LOCK_TIMEOUT:
		if file, ok := file.(*vfsFile); ok {
			millis := file.lockTimeout.Milliseconds()
			file.lockTimeout = time.Duration(util.ReadUint32(mod, pArg)) * time.Millisecond
			util.WriteUint32(mod, pArg, uint32(millis))
			return _OK
		}

	case _FCNTL_POWERSAFE_OVERWRITE:
		if file, ok := file.(FilePowersafeOverwrite); ok {
			switch util.ReadUint32(mod, pArg) {
			case 0:
				file.SetPowersafeOverwrite(false)
			case 1:
				file.SetPowersafeOverwrite(true)
			default:
				if file.PowersafeOverwrite() {
					util.WriteUint32(mod, pArg, 1)
				} else {
					util.WriteUint32(mod, pArg, 0)
				}
			}
			return _OK
		}

	case _FCNTL_SIZE_HINT:
		if file, ok := file.(FileSizeHint); ok {
			size := util.ReadUint64(mod, pArg)
			err := file.SizeHint(int64(size))
			return vfsErrorCode(err, _IOERR_TRUNCATE)
		}

	case _FCNTL_HAS_MOVED:
		if file, ok := file.(FileHasMoved); ok {
			moved, err := file.HasMoved()

			var res uint32
			if moved {
				res = 1
			}

			util.WriteUint32(mod, pArg, res)
			return vfsErrorCode(err, _IOERR_FSTAT)
		}
	}

	// Consider also implementing these opcodes (in use by SQLite):
	//  _FCNTL_BUSYHANDLER
	//  _FCNTL_COMMIT_PHASETWO
	//  _FCNTL_PDB
	//  _FCNTL_PRAGMA
	//  _FCNTL_SYNC
	return _NOTFOUND
}

func vfsSectorSize(ctx context.Context, mod api.Module, pFile uint32) uint32 {
	file := vfsFileGet(ctx, mod, pFile)
	return uint32(file.SectorSize())
}

func vfsDeviceCharacteristics(ctx context.Context, mod api.Module, pFile uint32) DeviceCharacteristic {
	file := vfsFileGet(ctx, mod, pFile)
	return file.DeviceCharacteristics()
}

func vfsGet(mod api.Module, pVfs uint32) VFS {
	if pVfs == 0 {
		return vfsOS{}
	}
	const zNameOffset = 16
	name := util.ReadString(mod, util.ReadUint32(mod, pVfs+zNameOffset), _MAX_STRING)
	if name == "os" {
		return vfsOS{}
	}
	if vfs := Find(name); vfs != nil {
		return vfs
	}
	panic(util.NoVFSErr + util.ErrorString(name))
}

func vfsErrorCode(err error, def _ErrorCode) _ErrorCode {
	if err == nil {
		return _OK
	}
	switch v := reflect.ValueOf(err); v.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return _ErrorCode(v.Uint())
	}
	return def
}
