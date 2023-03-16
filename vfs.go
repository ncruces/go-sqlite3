package sqlite3

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

	"github.com/ncruces/julianday"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/sys"
)

func vfsInstantiate(ctx context.Context, r wazero.Runtime) {
	wasi := r.NewHostModuleBuilder("wasi_snapshot_preview1")
	wasi.NewFunctionBuilder().WithFunc(vfsExit).Export("proc_exit")
	_, err := wasi.Instantiate(ctx)
	if err != nil {
		panic(err)
	}

	env := vfsNewEnvModuleBuilder(r)
	_, err = env.Instantiate(ctx)
	if err != nil {
		panic(err)
	}
}

func vfsNewEnvModuleBuilder(r wazero.Runtime) wazero.HostModuleBuilder {
	env := r.NewHostModuleBuilder("env")
	env.NewFunctionBuilder().WithFunc(vfsLocaltime).Export("os_localtime")
	env.NewFunctionBuilder().WithFunc(vfsRandomness).Export("os_randomness")
	env.NewFunctionBuilder().WithFunc(vfsSleep).Export("os_sleep")
	env.NewFunctionBuilder().WithFunc(vfsCurrentTime).Export("os_current_time")
	env.NewFunctionBuilder().WithFunc(vfsCurrentTime64).Export("os_current_time_64")
	env.NewFunctionBuilder().WithFunc(vfsFullPathname).Export("os_full_pathname")
	env.NewFunctionBuilder().WithFunc(vfsDelete).Export("os_delete")
	env.NewFunctionBuilder().WithFunc(vfsAccess).Export("os_access")
	env.NewFunctionBuilder().WithFunc(vfsOpen).Export("os_open")
	env.NewFunctionBuilder().WithFunc(vfsClose).Export("os_close")
	env.NewFunctionBuilder().WithFunc(vfsRead).Export("os_read")
	env.NewFunctionBuilder().WithFunc(vfsWrite).Export("os_write")
	env.NewFunctionBuilder().WithFunc(vfsTruncate).Export("os_truncate")
	env.NewFunctionBuilder().WithFunc(vfsSync).Export("os_sync")
	env.NewFunctionBuilder().WithFunc(vfsFileSize).Export("os_file_size")
	env.NewFunctionBuilder().WithFunc(vfsLock).Export("os_lock")
	env.NewFunctionBuilder().WithFunc(vfsUnlock).Export("os_unlock")
	env.NewFunctionBuilder().WithFunc(vfsCheckReservedLock).Export("os_check_reserved_lock")
	env.NewFunctionBuilder().WithFunc(vfsFileControl).Export("os_file_control")
	return env
}

// Poor man's namespaces.
const (
	vfsOS   vfsOSMethods   = false
	vfsFile vfsFileMethods = false
)

type (
	vfsOSMethods   bool
	vfsFileMethods bool
)

type vfsKey struct{}
type vfsState struct {
	files []*os.File
}

func vfsContext(ctx context.Context) (context.Context, io.Closer) {
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

func vfsExit(ctx context.Context, mod api.Module, exitCode uint32) {
	// Ensure other callers see the exit code.
	_ = mod.CloseWithExitCode(ctx, exitCode)
	// Prevent any code from executing after this function.
	panic(sys.NewExitError(mod.Name(), exitCode))
}

func vfsLocaltime(ctx context.Context, mod api.Module, t uint64, pTm uint32) uint32 {
	tm := time.Unix(int64(t), 0)
	var isdst int
	if tm.IsDST() {
		isdst = 1
	}

	// https://pubs.opengroup.org/onlinepubs/7908799/xsh/time.h.html
	mem := memory{mod}
	mem.writeUint32(pTm+0*ptrlen, uint32(tm.Second()))
	mem.writeUint32(pTm+1*ptrlen, uint32(tm.Minute()))
	mem.writeUint32(pTm+2*ptrlen, uint32(tm.Hour()))
	mem.writeUint32(pTm+3*ptrlen, uint32(tm.Day()))
	mem.writeUint32(pTm+4*ptrlen, uint32(tm.Month()-time.January))
	mem.writeUint32(pTm+5*ptrlen, uint32(tm.Year()-1900))
	mem.writeUint32(pTm+6*ptrlen, uint32(tm.Weekday()-time.Sunday))
	mem.writeUint32(pTm+7*ptrlen, uint32(tm.YearDay()-1))
	mem.writeUint32(pTm+8*ptrlen, uint32(isdst))
	return _OK
}

func vfsRandomness(ctx context.Context, mod api.Module, pVfs, nByte, zByte uint32) uint32 {
	mem := memory{mod}.view(zByte, uint64(nByte))
	n, _ := rand.Reader.Read(mem)
	return uint32(n)
}

func vfsSleep(ctx context.Context, pVfs, nMicro uint32) uint32 {
	time.Sleep(time.Duration(nMicro) * time.Microsecond)
	return _OK
}

func vfsCurrentTime(ctx context.Context, mod api.Module, pVfs, prNow uint32) uint32 {
	day := julianday.Float(time.Now())
	memory{mod}.writeFloat64(prNow, day)
	return _OK
}

func vfsCurrentTime64(ctx context.Context, mod api.Module, pVfs, piNow uint32) uint32 {
	day, nsec := julianday.Date(time.Now())
	msec := day*86_400_000 + nsec/1_000_000
	memory{mod}.writeUint64(piNow, uint64(msec))
	return _OK
}

func vfsFullPathname(ctx context.Context, mod api.Module, pVfs, zRelative, nFull, zFull uint32) uint32 {
	rel := memory{mod}.readString(zRelative, _MAX_PATHNAME)
	abs, err := filepath.Abs(rel)
	if err != nil {
		return uint32(CANTOPEN_FULLPATH)
	}

	// Consider either using [filepath.EvalSymlinks] to canonicalize the path (as the Unix VFS does).
	// Or using [os.Readlink] to resolve a symbolic link (as the Unix VFS did).
	// This might be buggy on Windows (the Windows VFS doesn't try).

	size := uint64(len(abs) + 1)
	if size > uint64(nFull) {
		return uint32(CANTOPEN_FULLPATH)
	}
	mem := memory{mod}.view(zFull, size)
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
	return uint32(CANTOPEN_FULLPATH)
}

func vfsDelete(ctx context.Context, mod api.Module, pVfs, zPath, syncDir uint32) uint32 {
	path := memory{mod}.readString(zPath, _MAX_PATHNAME)
	err := os.Remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return uint32(IOERR_DELETE_NOENT)
	}
	if err != nil {
		return uint32(IOERR_DELETE)
	}
	if runtime.GOOS != "windows" && syncDir != 0 {
		f, err := os.Open(filepath.Dir(path))
		if err != nil {
			return _OK
		}
		defer f.Close()
		err = vfsOS.Sync(f, false, false)
		if err != nil {
			return uint32(IOERR_DIR_FSYNC)
		}
	}
	return _OK
}

func vfsAccess(ctx context.Context, mod api.Module, pVfs, zPath uint32, flags _AccessFlag, pResOut uint32) uint32 {
	path := memory{mod}.readString(zPath, _MAX_PATHNAME)
	ok, rc := vfsOS.Access(path, flags)
	var res uint32
	if ok {
		res = 1
	}
	memory{mod}.writeUint32(pResOut, res)
	return uint32(rc)
}

func vfsOpen(ctx context.Context, mod api.Module, pVfs, zName, pFile uint32, flags OpenFlag, pOutFlags uint32) uint32 {
	var oflags int
	if flags&OPEN_EXCLUSIVE != 0 {
		oflags |= os.O_EXCL
	}
	if flags&OPEN_CREATE != 0 {
		oflags |= os.O_CREATE
	}
	if flags&OPEN_READONLY != 0 {
		oflags |= os.O_RDONLY
	}
	if flags&OPEN_READWRITE != 0 {
		oflags |= os.O_RDWR
	}

	var err error
	var file *os.File
	if zName == 0 {
		file, err = os.CreateTemp("", "*.db")
	} else {
		name := memory{mod}.readString(zName, _MAX_PATHNAME)
		file, err = vfsOS.OpenFile(name, oflags, 0600)
	}
	if err != nil {
		return uint32(CANTOPEN)
	}

	if flags&OPEN_DELETEONCLOSE != 0 {
		os.Remove(file.Name())
	}

	vfsFile.Open(ctx, mod, pFile, file)

	if pOutFlags != 0 {
		memory{mod}.writeUint32(pOutFlags, uint32(flags))
	}
	return _OK
}

func vfsClose(ctx context.Context, mod api.Module, pFile uint32) uint32 {
	err := vfsFile.Close(ctx, mod, pFile)
	if err != nil {
		return uint32(IOERR_CLOSE)
	}
	return _OK
}

func vfsRead(ctx context.Context, mod api.Module, pFile, zBuf, iAmt uint32, iOfst uint64) uint32 {
	buf := memory{mod}.view(zBuf, uint64(iAmt))

	file := vfsFile.GetOS(ctx, mod, pFile)
	n, err := file.ReadAt(buf, int64(iOfst))
	if n == int(iAmt) {
		return _OK
	}
	if n == 0 && err != io.EOF {
		return uint32(IOERR_READ)
	}
	for i := range buf[n:] {
		buf[n+i] = 0
	}
	return uint32(IOERR_SHORT_READ)
}

func vfsWrite(ctx context.Context, mod api.Module, pFile, zBuf, iAmt uint32, iOfst uint64) uint32 {
	buf := memory{mod}.view(zBuf, uint64(iAmt))

	file := vfsFile.GetOS(ctx, mod, pFile)
	_, err := file.WriteAt(buf, int64(iOfst))
	if err != nil {
		return uint32(IOERR_WRITE)
	}
	return _OK
}

func vfsTruncate(ctx context.Context, mod api.Module, pFile uint32, nByte uint64) uint32 {
	file := vfsFile.GetOS(ctx, mod, pFile)
	err := file.Truncate(int64(nByte))
	if err != nil {
		return uint32(IOERR_TRUNCATE)
	}
	return _OK
}

func vfsSync(ctx context.Context, mod api.Module, pFile uint32, flags _SyncFlag) uint32 {
	dataonly := (flags & _SYNC_DATAONLY) != 0
	fullsync := (flags & 0x0f) == _SYNC_FULL
	file := vfsFile.GetOS(ctx, mod, pFile)
	err := vfsOS.Sync(file, fullsync, dataonly)
	if err != nil {
		return uint32(IOERR_FSYNC)
	}
	return _OK
}

func vfsFileSize(ctx context.Context, mod api.Module, pFile, pSize uint32) uint32 {
	file := vfsFile.GetOS(ctx, mod, pFile)
	off, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return uint32(IOERR_SEEK)
	}

	memory{mod}.writeUint64(pSize, uint64(off))
	return _OK
}

func vfsFileControl(ctx context.Context, pFile, op, pArg uint32) uint32 {
	// SQLite calls vfsFileControl with these opcodes:
	//  SQLITE_FCNTL_SIZE_HINT
	//  SQLITE_FCNTL_PRAGMA
	//  SQLITE_FCNTL_BUSYHANDLER
	//  SQLITE_FCNTL_HAS_MOVED
	//  SQLITE_FCNTL_SYNC
	//  SQLITE_FCNTL_COMMIT_PHASETWO
	//  SQLITE_FCNTL_PDB
	return uint32(NOTFOUND)
}
