package sqlite3

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ncruces/julianday"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/sys"
)

func vfsInstantiate(ctx context.Context, r wazero.Runtime) (err error) {
	wasi := r.NewHostModuleBuilder("wasi_snapshot_preview1")
	wasi.NewFunctionBuilder().WithFunc(vfsExit).Export("proc_exit")
	_, err = wasi.Instantiate(ctx)
	if err != nil {
		return err
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
	_, err = env.Instantiate(ctx)
	return err
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

	if pTm == 0 {
		panic(nilErr)
	}
	// https://pubs.opengroup.org/onlinepubs/7908799/xsh/time.h.html
	if mem := mod.Memory(); true &&
		mem.WriteUint32Le(pTm+0*wordSize, uint32(tm.Second())) &&
		mem.WriteUint32Le(pTm+1*wordSize, uint32(tm.Minute())) &&
		mem.WriteUint32Le(pTm+2*wordSize, uint32(tm.Hour())) &&
		mem.WriteUint32Le(pTm+3*wordSize, uint32(tm.Day())) &&
		mem.WriteUint32Le(pTm+4*wordSize, uint32(tm.Month()-time.January)) &&
		mem.WriteUint32Le(pTm+5*wordSize, uint32(tm.Year()-1900)) &&
		mem.WriteUint32Le(pTm+6*wordSize, uint32(tm.Weekday()-time.Sunday)) &&
		mem.WriteUint32Le(pTm+7*wordSize, uint32(tm.YearDay()-1)) &&
		mem.WriteUint32Le(pTm+8*wordSize, uint32(isdst)) {
		return _OK
	}
	panic(rangeErr)
}

func vfsRandomness(ctx context.Context, mod api.Module, pVfs, nByte, zByte uint32) uint32 {
	if zByte == 0 {
		panic(nilErr)
	}
	mem, ok := mod.Memory().Read(zByte, nByte)
	if !ok {
		panic(rangeErr)
	}
	n, _ := rand.Read(mem)
	return uint32(n)
}

func vfsSleep(ctx context.Context, pVfs, nMicro uint32) uint32 {
	time.Sleep(time.Duration(nMicro) * time.Microsecond)
	return _OK
}

func vfsCurrentTime(ctx context.Context, mod api.Module, pVfs, prNow uint32) uint32 {
	day := julianday.Float(time.Now())
	if prNow == 0 {
		panic(nilErr)
	}
	if ok := mod.Memory().WriteFloat64Le(prNow, day); !ok {
		panic(rangeErr)
	}
	return _OK
}

func vfsCurrentTime64(ctx context.Context, mod api.Module, pVfs, piNow uint32) uint32 {
	day, nsec := julianday.Date(time.Now())
	msec := day*86_400_000 + nsec/1_000_000
	if piNow == 0 {
		panic(nilErr)
	}
	if ok := mod.Memory().WriteUint64Le(piNow, uint64(msec)); !ok {
		panic(rangeErr)
	}
	return _OK
}

func vfsFullPathname(ctx context.Context, mod api.Module, pVfs, zRelative, nFull, zFull uint32) uint32 {
	rel := getString(mod.Memory(), zRelative, _MAX_PATHNAME)
	abs, err := filepath.Abs(rel)
	if err != nil {
		return uint32(IOERR)
	}

	// Consider either using [filepath.EvalSymlinks] to canonicalize the path (as the Unix VFS does).
	// Or using [os.Readlink] to resolve a symbolic link (as the Unix VFS did).
	// This might be buggy on Windows (the Windows VFS doesn't try).

	siz := uint32(len(abs) + 1)
	if siz > nFull {
		return uint32(CANTOPEN_FULLPATH)
	}
	if zFull == 0 {
		panic(nilErr)
	}
	mem, ok := mod.Memory().Read(zFull, siz)
	if !ok {
		panic(rangeErr)
	}

	mem[len(abs)] = 0
	copy(mem, abs)
	return _OK
}

func vfsDelete(ctx context.Context, mod api.Module, pVfs, zPath, syncDir uint32) uint32 {
	path := getString(mod.Memory(), zPath, _MAX_PATHNAME)
	err := os.Remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return _OK
	}
	if err != nil {
		return uint32(IOERR_DELETE)
	}
	if syncDir != 0 {
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

func vfsAccess(ctx context.Context, mod api.Module, pVfs, zPath uint32, flags AccessFlag, pResOut uint32) uint32 {
	// Consider using [syscall.Access] for [ACCESS_READWRITE]/[ACCESS_READ]
	// (as the Unix VFS does).

	path := getString(mod.Memory(), zPath, _MAX_PATHNAME)
	fi, err := os.Stat(path)

	var res uint32
	switch {
	case flags == ACCESS_EXISTS:
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
		if flags == ACCESS_READWRITE {
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

	if pResOut == 0 {
		panic(nilErr)
	}
	if ok := mod.Memory().WriteUint32Le(pResOut, res); !ok {
		panic(rangeErr)
	}
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
		name := getString(mod.Memory(), zName, _MAX_PATHNAME)
		file, err = os.OpenFile(name, oflags, 0600)
	}
	if err != nil {
		return uint32(CANTOPEN)
	}

	if flags&OPEN_DELETEONCLOSE != 0 {
		deleteOnClose(file)
	}

	info, err := file.Stat()
	if err != nil {
		return uint32(CANTOPEN)
	}
	if info.IsDir() {
		return uint32(CANTOPEN_ISDIR)
	}
	id := vfsGetOpenFileID(file, info)
	vfsFilePtr{mod, pFile}.SetID(id).SetLock(_NO_LOCK)

	if pOutFlags == 0 {
		return _OK
	}
	if ok := mod.Memory().WriteUint32Le(pOutFlags, uint32(flags)); !ok {
		panic(rangeErr)
	}
	return _OK
}

func vfsClose(ctx context.Context, mod api.Module, pFile uint32) uint32 {
	id := vfsFilePtr{mod, pFile}.ID()
	err := vfsReleaseOpenFile(id)
	if err != nil {
		return uint32(IOERR_CLOSE)
	}
	return _OK
}

func vfsRead(ctx context.Context, mod api.Module, pFile, zBuf, iAmt uint32, iOfst uint64) uint32 {
	if zBuf == 0 {
		panic(nilErr)
	}
	buf, ok := mod.Memory().Read(zBuf, iAmt)
	if !ok {
		panic(rangeErr)
	}

	file := vfsFilePtr{mod, pFile}.OSFile()
	n, err := file.ReadAt(buf, int64(iOfst))
	if n == int(iAmt) {
		return _OK
	}
	if n == 0 && err != io.EOF {
		return uint32(IOERR_READ)
	}
	for i := range buf[n:] {
		buf[i] = 0
	}
	return uint32(IOERR_SHORT_READ)
}

func vfsWrite(ctx context.Context, mod api.Module, pFile, zBuf, iAmt uint32, iOfst uint64) uint32 {
	if zBuf == 0 {
		panic(nilErr)
	}
	buf, ok := mod.Memory().Read(zBuf, iAmt)
	if !ok {
		panic(rangeErr)
	}

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
	// This uses [file.Seek] because we don't care about the offset for reading/writing.
	// But consider using [file.Stat] instead (as other VFSes do).

	file := vfsFilePtr{mod, pFile}.OSFile()
	off, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return uint32(IOERR_SEEK)
	}

	if pSize == 0 {
		panic(nilErr)
	}
	if ok := mod.Memory().WriteUint64Le(pSize, uint64(off)); !ok {
		panic(rangeErr)
	}
	return _OK
}
