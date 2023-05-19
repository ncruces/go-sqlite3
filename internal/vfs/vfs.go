package vfs

import (
	"context"
	"crypto/rand"
	"io"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/sqlite3vfs"
	"github.com/ncruces/julianday"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

func Export(env wazero.HostModuleBuilder) wazero.HostModuleBuilder {
	util.RegisterFuncII(env, "go_vfs_find", vfsFind)
	util.RegisterFuncIIJ(env, "go_localtime", vfsLocaltime)
	util.RegisterFuncIIII(env, "go_randomness", vfsRandomness)
	util.RegisterFuncIII(env, "go_sleep", vfsSleep)
	util.RegisterFuncIII(env, "go_current_time", vfsCurrentTime)
	util.RegisterFuncIII(env, "go_current_time_64", vfsCurrentTime64)
	util.RegisterFuncIIIII(env, "go_full_pathname", vfsFullPathname)
	util.RegisterFuncIIII(env, "go_delete", vfsDelete)
	util.RegisterFuncIIIII(env, "go_access", vfsAccess)
	util.RegisterFuncIIIIII(env, "go_open", vfsOpen)
	util.RegisterFuncII(env, "go_close", vfsClose)
	util.RegisterFuncIIIIJ(env, "go_read", vfsRead)
	util.RegisterFuncIIIIJ(env, "go_write", vfsWrite)
	util.RegisterFuncIIJ(env, "go_truncate", vfsTruncate)
	util.RegisterFuncIII(env, "go_sync", vfsSync)
	util.RegisterFuncIII(env, "go_file_size", vfsFileSize)
	util.RegisterFuncIIII(env, "go_file_control", vfsFileControl)
	util.RegisterFuncII(env, "go_sector_size", vfsSectorSize)
	util.RegisterFuncII(env, "go_device_characteristics", vfsDeviceCharacteristics)
	util.RegisterFuncIII(env, "go_lock", vfsLock)
	util.RegisterFuncIII(env, "go_unlock", vfsUnlock)
	util.RegisterFuncIII(env, "go_check_reserved_lock", vfsCheckReservedLock)
	return env
}

type vfsKey struct{}
type vfsState struct {
	files []sqlite3vfs.File
}

func Context(ctx context.Context) (context.Context, io.Closer) {
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
	if sqlite3vfs.Find(name) != nil {
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
	vfs := vfsAPIGet(mod, pVfs)
	path := util.ReadString(mod, zRelative, _MAX_PATHNAME)

	path, err := vfs.FullPathname(path)

	size := uint64(len(path) + 1)
	if size > uint64(nFull) {
		return _CANTOPEN_FULLPATH
	}
	mem := util.View(mod, zFull, size)
	mem[len(path)] = 0
	copy(mem, path)

	return vfsAPIErrorCode(err, _CANTOPEN_FULLPATH)
}

func vfsDelete(ctx context.Context, mod api.Module, pVfs, zPath, syncDir uint32) _ErrorCode {
	vfs := vfsAPIGet(mod, pVfs)
	path := util.ReadString(mod, zPath, _MAX_PATHNAME)

	err := vfs.Delete(path, syncDir != 0)
	return vfsAPIErrorCode(err, _IOERR_DELETE)
}

func vfsAccess(ctx context.Context, mod api.Module, pVfs, zPath uint32, flags _AccessFlag, pResOut uint32) _ErrorCode {
	vfs := vfsAPIGet(mod, pVfs)
	path := util.ReadString(mod, zPath, _MAX_PATHNAME)

	ok, err := vfs.Access(path, flags)
	var res uint32
	if ok {
		res = 1
	}

	util.WriteUint32(mod, pResOut, res)
	return vfsAPIErrorCode(err, _IOERR_ACCESS)
}

func vfsOpen(ctx context.Context, mod api.Module, pVfs, zPath, pFile uint32, flags _OpenFlag, pOutFlags uint32) _ErrorCode {
	vfs := vfsAPIGet(mod, pVfs)

	var path string
	if zPath != 0 {
		path = util.ReadString(mod, zPath, _MAX_PATHNAME)
	}

	file, flags, err := vfs.Open(path, flags)
	if err != nil {
		return vfsAPIErrorCode(err, _CANTOPEN)
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
	return vfsAPIErrorCode(err, _IOERR_TRUNCATE)
}

func vfsSync(ctx context.Context, mod api.Module, pFile uint32, flags _SyncFlag) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile)
	err := file.Sync(flags)
	return vfsAPIErrorCode(err, _IOERR_FSYNC)
}

func vfsFileSize(ctx context.Context, mod api.Module, pFile, pSize uint32) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile)
	size, err := file.FileSize()
	util.WriteUint64(mod, pSize, uint64(size))
	return vfsAPIErrorCode(err, _IOERR_SEEK)
}

func vfsLock(ctx context.Context, mod api.Module, pFile uint32, eLock _LockLevel) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile)
	err := file.Lock(eLock)
	return vfsAPIErrorCode(err, _IOERR_LOCK)
}

func vfsUnlock(ctx context.Context, mod api.Module, pFile uint32, eLock _LockLevel) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile)
	err := file.Unlock(eLock)
	return vfsAPIErrorCode(err, _IOERR_UNLOCK)
}

func vfsCheckReservedLock(ctx context.Context, mod api.Module, pFile, pResOut uint32) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile)
	locked, err := file.CheckReservedLock()

	var res uint32
	if locked {
		res = 1
	}

	util.WriteUint32(mod, pResOut, res)
	return vfsAPIErrorCode(err, _IOERR_CHECKRESERVEDLOCK)
}

func vfsFileControl(ctx context.Context, mod api.Module, pFile uint32, op _FcntlOpcode, pArg uint32) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile)

	switch op {
	case _FCNTL_LOCKSTATE:
		if file, ok := file.(sqlite3vfs.FileLockState); ok {
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
		if file, ok := file.(sqlite3vfs.FilePowersafeOverwrite); ok {
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
		if file, ok := file.(sqlite3vfs.FileSizeHint); ok {
			size := util.ReadUint64(mod, pArg)
			err := file.SizeHint(int64(size))
			return vfsAPIErrorCode(err, _IOERR_TRUNCATE)
		}

	case _FCNTL_HAS_MOVED:
		if file, ok := file.(sqlite3vfs.FileHasMoved); ok {
			moved, err := file.HasMoved()

			var res uint32
			if moved {
				res = 1
			}

			util.WriteUint32(mod, pArg, res)
			return vfsAPIErrorCode(err, _IOERR_FSTAT)
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

func vfsDeviceCharacteristics(ctx context.Context, mod api.Module, pFile uint32) _DeviceCharacteristic {
	file := vfsFileGet(ctx, mod, pFile)
	return file.DeviceCharacteristics()
}
