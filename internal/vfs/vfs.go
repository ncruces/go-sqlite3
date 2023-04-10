package vfs

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/julianday"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

func Instantiate(ctx context.Context, r wazero.Runtime) {
	env := NewEnvModuleBuilder(r)
	_, err := env.Instantiate(ctx)
	if err != nil {
		panic(err)
	}
}

func NewEnvModuleBuilder(r wazero.Runtime) wazero.HostModuleBuilder {
	env := r.NewHostModuleBuilder("env")
	registerFuncT(env, "os_localtime", vfsLocaltime)
	registerFunc3(env, "os_randomness", vfsRandomness)
	registerFunc2(env, "os_sleep", vfsSleep)
	registerFunc2(env, "os_current_time", vfsCurrentTime)
	registerFunc2(env, "os_current_time_64", vfsCurrentTime64)
	registerFunc4(env, "os_full_pathname", vfsFullPathname)
	registerFunc3(env, "os_delete", vfsDelete)
	registerFunc4(env, "os_access", vfsAccess)
	registerFunc5(env, "os_open", vfsOpen)
	registerFunc1(env, "os_close", vfsClose)
	registerFuncRW(env, "os_read", vfsRead)
	registerFuncRW(env, "os_write", vfsWrite)
	registerFuncT(env, "os_truncate", vfsTruncate)
	registerFunc2(env, "os_sync", vfsSync)
	registerFunc2(env, "os_file_size", vfsFileSize)
	registerFunc3(env, "os_file_control", vfsFileControl)
	registerFunc1(env, "os_sector_size", vfsSectorSize)
	registerFunc1(env, "os_device_characteristics", vfsDeviceCharacteristics)
	registerFunc2(env, "os_lock", vfsLock)
	registerFunc2(env, "os_unlock", vfsUnlock)
	registerFunc2(env, "os_check_reserved_lock", vfsCheckReservedLock)
	return env
}

type vfsKey struct{}
type vfsState struct {
	files []vfsFile
}

func Context(ctx context.Context) (context.Context, io.Closer) {
	vfs := &vfsState{}
	return context.WithValue(ctx, vfsKey{}, vfs), vfs
}

func (vfs *vfsState) Close() error {
	for _, f := range vfs.files {
		if f.File != nil {
			f.Close()
		}
	}
	vfs.files = nil
	return nil
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
	rel := util.ReadString(mod, zRelative, _MAX_PATHNAME)
	abs, err := filepath.Abs(rel)
	if err != nil {
		return _CANTOPEN_FULLPATH
	}

	size := uint64(len(abs) + 1)
	if size > uint64(nFull) {
		return _CANTOPEN_FULLPATH
	}
	mem := util.View(mod, zFull, size)
	mem[len(abs)] = 0
	copy(mem, abs)

	if fi, err := os.Lstat(abs); err == nil {
		if fi.Mode()&fs.ModeSymlink != 0 {
			return _OK_SYMLINK
		}
		return _OK
	} else if errors.Is(err, fs.ErrNotExist) {
		return _OK
	}
	return _CANTOPEN_FULLPATH
}

func vfsDelete(ctx context.Context, mod api.Module, pVfs, zPath, syncDir uint32) _ErrorCode {
	path := util.ReadString(mod, zPath, _MAX_PATHNAME)
	err := os.Remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return _IOERR_DELETE_NOENT
	}
	if err != nil {
		return _IOERR_DELETE
	}
	if runtime.GOOS != "windows" && syncDir != 0 {
		f, err := os.Open(filepath.Dir(path))
		if err != nil {
			return _OK
		}
		defer f.Close()
		err = osSync(f, false, false)
		if err != nil {
			return _IOERR_DIR_FSYNC
		}
	}
	return _OK
}

func vfsAccess(ctx context.Context, mod api.Module, pVfs, zPath uint32, flags _AccessFlag, pResOut uint32) _ErrorCode {
	path := util.ReadString(mod, zPath, _MAX_PATHNAME)
	err := osAccess(path, flags)

	var res uint32
	var rc _ErrorCode
	if flags == _ACCESS_EXISTS {
		switch {
		case err == nil:
			res = 1
		case errors.Is(err, fs.ErrNotExist):
			res = 0
		default:
			rc = _IOERR_ACCESS
		}
	} else {
		switch {
		case err == nil:
			res = 1
		case errors.Is(err, fs.ErrPermission):
			res = 0
		default:
			rc = _IOERR_ACCESS
		}
	}

	util.WriteUint32(mod, pResOut, res)
	return rc
}

func vfsOpen(ctx context.Context, mod api.Module, pVfs, zName, pFile uint32, flags _OpenFlag, pOutFlags uint32) _ErrorCode {
	var oflags int
	if flags&_OPEN_EXCLUSIVE != 0 {
		oflags |= os.O_EXCL
	}
	if flags&_OPEN_CREATE != 0 {
		oflags |= os.O_CREATE
	}
	if flags&_OPEN_READONLY != 0 {
		oflags |= os.O_RDONLY
	}
	if flags&_OPEN_READWRITE != 0 {
		oflags |= os.O_RDWR
	}

	var err error
	var f *os.File
	if zName == 0 {
		f, err = os.CreateTemp("", "*.db")
	} else {
		name := util.ReadString(mod, zName, _MAX_PATHNAME)
		f, err = osOpenFile(name, oflags, 0666)
	}
	if err != nil {
		return _CANTOPEN
	}

	if flags&_OPEN_DELETEONCLOSE != 0 {
		os.Remove(f.Name())
	}

	file := openVFSFile(ctx, mod, pFile, f)
	file.psow = true
	file.readOnly = flags&_OPEN_READONLY != 0
	file.syncDir = runtime.GOOS != "windows" &&
		flags&(_OPEN_CREATE) != 0 &&
		flags&(_OPEN_MAIN_JOURNAL|_OPEN_SUPER_JOURNAL|_OPEN_WAL) != 0

	if pOutFlags != 0 {
		util.WriteUint32(mod, pOutFlags, uint32(flags))
	}
	return _OK
}

func vfsClose(ctx context.Context, mod api.Module, pFile uint32) _ErrorCode {
	err := closeVFSFile(ctx, mod, pFile)
	if err != nil {
		return _IOERR_CLOSE
	}
	return _OK
}

func vfsRead(ctx context.Context, mod api.Module, pFile, zBuf, iAmt uint32, iOfst int64) _ErrorCode {
	buf := util.View(mod, zBuf, uint64(iAmt))

	file := getVFSFile(ctx, mod, pFile)
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
	buf := util.View(mod, zBuf, uint64(iAmt))

	file := getVFSFile(ctx, mod, pFile)
	_, err := file.WriteAt(buf, iOfst)
	if err != nil {
		return _IOERR_WRITE
	}
	return _OK
}

func vfsTruncate(ctx context.Context, mod api.Module, pFile uint32, nByte int64) _ErrorCode {
	file := getVFSFile(ctx, mod, pFile)
	err := file.Truncate(nByte)
	if err != nil {
		return _IOERR_TRUNCATE
	}
	return _OK
}

func vfsSync(ctx context.Context, mod api.Module, pFile uint32, flags _SyncFlag) _ErrorCode {
	dataonly := (flags & _SYNC_DATAONLY) != 0
	fullsync := (flags & 0x0f) == _SYNC_FULL

	file := getVFSFile(ctx, mod, pFile)
	err := osSync(file.File, fullsync, dataonly)
	if err != nil {
		return _IOERR_FSYNC
	}
	if runtime.GOOS != "windows" && file.syncDir {
		file.syncDir = false
		f, err := os.Open(filepath.Dir(file.Name()))
		if err != nil {
			return _OK
		}
		defer f.Close()
		err = osSync(f, false, false)
		if err != nil {
			return _IOERR_DIR_FSYNC
		}
	}
	return _OK
}

func vfsFileSize(ctx context.Context, mod api.Module, pFile, pSize uint32) _ErrorCode {
	file := getVFSFile(ctx, mod, pFile)
	off, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return _IOERR_SEEK
	}

	util.WriteUint64(mod, pSize, uint64(off))
	return _OK
}

func vfsFileControl(ctx context.Context, mod api.Module, pFile uint32, op _FcntlOpcode, pArg uint32) _ErrorCode {
	switch op {
	case _FCNTL_LOCKSTATE:
		util.WriteUint32(mod, pArg, uint32(getVFSFile(ctx, mod, pFile).lock))
		return _OK
	case _FCNTL_LOCK_TIMEOUT:
		file := getVFSFile(ctx, mod, pFile)
		millis := file.lockTimeout.Milliseconds()
		file.lockTimeout = time.Duration(util.ReadUint32(mod, pArg)) * time.Millisecond
		util.WriteUint32(mod, pArg, uint32(millis))
		return _OK
	case _FCNTL_POWERSAFE_OVERWRITE:
		file := getVFSFile(ctx, mod, pFile)
		switch util.ReadUint32(mod, pArg) {
		case 0:
			file.psow = false
		case 1:
			file.psow = true
		default:
			if file.psow {
				util.WriteUint32(mod, pArg, 1)
			} else {
				util.WriteUint32(mod, pArg, 0)
			}
		}
	case _FCNTL_SIZE_HINT:
		return vfsSizeHint(ctx, mod, pFile, pArg)
	case _FCNTL_HAS_MOVED:
		return vfsFileMoved(ctx, mod, pFile, pArg)
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
	return _DEFAULT_SECTOR_SIZE
}

func vfsDeviceCharacteristics(ctx context.Context, mod api.Module, pFile uint32) _DeviceChars {
	file := getVFSFile(ctx, mod, pFile)
	if file.psow {
		return _IOCAP_POWERSAFE_OVERWRITE
	}
	return 0
}

func vfsSizeHint(ctx context.Context, mod api.Module, pFile, pArg uint32) _ErrorCode {
	file := getVFSFile(ctx, mod, pFile)
	size := util.ReadUint64(mod, pArg)
	err := osAllocate(file.File, int64(size))
	if err != nil {
		return _IOERR_TRUNCATE
	}
	return _OK
}

func vfsFileMoved(ctx context.Context, mod api.Module, pFile, pResOut uint32) _ErrorCode {
	file := getVFSFile(ctx, mod, pFile)
	fi, err := file.Stat()
	if err != nil {
		return _IOERR_FSTAT
	}
	pi, err := os.Stat(file.Name())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return _IOERR_FSTAT
	}
	var res uint32
	if !os.SameFile(fi, pi) {
		res = 1
	}
	util.WriteUint32(mod, pResOut, res)
	return _OK
}
