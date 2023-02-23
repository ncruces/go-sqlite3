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
	"syscall"
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

	env := r.NewHostModuleBuilder("env")
	env.NewFunctionBuilder().WithFunc(vfsLocaltime).Export("go_localtime")
	env.NewFunctionBuilder().WithFunc(vfsRandomness).Export("go_randomness")
	env.NewFunctionBuilder().WithFunc(vfsSleep).Export("go_sleep")
	env.NewFunctionBuilder().WithFunc(vfsCurrentTime).Export("go_current_time")
	env.NewFunctionBuilder().WithFunc(vfsCurrentTime64).Export("go_current_time_64")
	env.NewFunctionBuilder().WithFunc(vfsFullPathname).Export("go_full_pathname")
	env.NewFunctionBuilder().WithFunc(vfsDelete).Export("go_delete")
	env.NewFunctionBuilder().WithFunc(vfsAccess).Export("go_access")
	env.NewFunctionBuilder().WithFunc(vfsOpen).Export("go_open")
	env.NewFunctionBuilder().WithFunc(vfsClose).Export("go_close")
	env.NewFunctionBuilder().WithFunc(vfsRead).Export("go_read")
	env.NewFunctionBuilder().WithFunc(vfsWrite).Export("go_write")
	env.NewFunctionBuilder().WithFunc(vfsTruncate).Export("go_truncate")
	env.NewFunctionBuilder().WithFunc(vfsSync).Export("go_sync")
	env.NewFunctionBuilder().WithFunc(vfsFileSize).Export("go_file_size")
	env.NewFunctionBuilder().WithFunc(vfsLock).Export("go_lock")
	env.NewFunctionBuilder().WithFunc(vfsUnlock).Export("go_unlock")
	env.NewFunctionBuilder().WithFunc(vfsCheckReservedLock).Export("go_check_reserved_lock")
	env.NewFunctionBuilder().WithFunc(vfsFileControl).Export("go_file_control")
	_, err = env.Instantiate(ctx)
	if err != nil {
		panic(err)
	}
}

type vfsOSMethods bool

const vfsOS vfsOSMethods = false

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
	mem := memory{mod}.view(zByte, nByte)
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
		return uint32(IOERR)
	}

	// Consider either using [filepath.EvalSymlinks] to canonicalize the path (as the Unix VFS does).
	// Or using [os.Readlink] to resolve a symbolic link (as the Unix VFS did).
	// This might be buggy on Windows (the Windows VFS doesn't try).

	size := uint32(len(abs) + 1)
	if size > nFull {
		return uint32(CANTOPEN_FULLPATH)
	}
	mem := memory{mod}.view(zFull, size)

	mem[len(abs)] = 0
	copy(mem, abs)
	return _OK
}

func vfsDelete(ctx context.Context, mod api.Module, pVfs, zPath, syncDir uint32) uint32 {
	path := memory{mod}.readString(zPath, _MAX_PATHNAME)
	err := os.Remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return _OK
	}
	if err != nil {
		return uint32(IOERR_DELETE)
	}
	if runtime.GOOS != "windows" && syncDir != 0 {
		f, err := os.Open(filepath.Dir(path))
		if err == nil {
			err = f.Sync()
			f.Close()
		}
		if err != nil {
			return uint32(IOERR_DELETE)
		}
	}
	return _OK
}

func vfsAccess(ctx context.Context, mod api.Module, pVfs, zPath uint32, flags _AccessFlag, pResOut uint32) uint32 {
	// Consider using [syscall.Access] for [ACCESS_READWRITE]/[ACCESS_READ]
	// (as the Unix VFS does).

	path := memory{mod}.readString(zPath, _MAX_PATHNAME)
	fi, err := os.Stat(path)

	var res uint32
	switch {
	case flags == _ACCESS_EXISTS:
		switch {
		case err == nil:
			res = 1
		case errors.Is(err, fs.ErrNotExist):
			res = 0
		default:
			return uint32(IOERR_ACCESS)
		}

	case err == nil:
		var want fs.FileMode = syscall.S_IRUSR
		if flags == _ACCESS_READWRITE {
			want |= syscall.S_IWUSR
		}
		if fi.IsDir() {
			want |= syscall.S_IXUSR
		}
		if fi.Mode()&want == want {
			res = 1
		} else {
			res = 0
		}

	case errors.Is(err, fs.ErrPermission):
		res = 0

	default:
		return uint32(IOERR_ACCESS)
	}

	memory{mod}.writeUint32(pResOut, res)
	return _OK
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
		file, err = os.OpenFile(name, oflags, 0600)
	}
	if err != nil {
		return uint32(CANTOPEN)
	}

	if flags&OPEN_DELETEONCLOSE != 0 {
		vfsOS.DeleteOnClose(file)
	}

	id := vfsGetFileID(file)
	vfsFilePtr{mod, pFile}.SetID(id).SetLock(_NO_LOCK)

	if pOutFlags != 0 {
		memory{mod}.writeUint32(pOutFlags, uint32(flags))
	}
	return _OK
}

func vfsClose(ctx context.Context, mod api.Module, pFile uint32) uint32 {
	id := vfsFilePtr{mod, pFile}.ID()
	err := vfsCloseFile(id)
	if err != nil {
		return uint32(IOERR_CLOSE)
	}
	return _OK
}

func vfsRead(ctx context.Context, mod api.Module, pFile, zBuf, iAmt uint32, iOfst uint64) uint32 {
	buf := memory{mod}.view(zBuf, iAmt)

	file := vfsFilePtr{mod, pFile}.OSFile()
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
	buf := memory{mod}.view(zBuf, iAmt)

	file := vfsFilePtr{mod, pFile}.OSFile()
	_, err := file.WriteAt(buf, int64(iOfst))
	if err != nil {
		return uint32(IOERR_WRITE)
	}
	return _OK
}

func vfsTruncate(ctx context.Context, mod api.Module, pFile uint32, nByte uint64) uint32 {
	file := vfsFilePtr{mod, pFile}.OSFile()
	err := file.Truncate(int64(nByte))
	if err != nil {
		return uint32(IOERR_TRUNCATE)
	}
	return _OK
}

func vfsSync(ctx context.Context, mod api.Module, pFile, flags uint32) uint32 {
	file := vfsFilePtr{mod, pFile}.OSFile()
	err := file.Sync()
	if err != nil {
		return uint32(IOERR_FSYNC)
	}
	return _OK
}

func vfsFileSize(ctx context.Context, mod api.Module, pFile, pSize uint32) uint32 {
	// This uses [os.File.Seek] because we don't care about the offset for reading/writing.
	// But consider using [os.File.Stat] instead (as other VFSes do).

	file := vfsFilePtr{mod, pFile}.OSFile()
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
