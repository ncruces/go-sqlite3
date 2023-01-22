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
	_, err = env.Instantiate(ctx)
	return err
}

func vfsExit(ctx context.Context, mod api.Module, exitCode uint32) {
	// Ensure other callers see the exit code.
	_ = mod.CloseWithExitCode(ctx, exitCode)
	// Prevent any code from executing after this function.
	panic(sys.NewExitError(mod.Name(), exitCode))
}

func vfsRandomness(ctx context.Context, mod api.Module, pVfs, nByte, zByte uint32) uint32 {
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
	if ok := mod.Memory().WriteFloat64Le(prNow, day); !ok {
		panic(rangeErr)
	}
	return _OK
}

func vfsCurrentTime64(ctx context.Context, mod api.Module, pVfs, piNow uint32) uint32 {
	day, nsec := julianday.Date(time.Now())
	msec := day*86_400_000 + nsec/1_000_000
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

	siz := uint32(len(abs) + 1)
	if siz > nFull {
		return uint32(IOERR)
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

func vfsAccess(ctx context.Context, mod api.Module, pVfs, zPath, flags, pResOut uint32) uint32 {
	path := getString(mod.Memory(), zPath, _MAX_PATHNAME)
	fi, err := os.Stat(path)

	var res uint32
	if flags == uint32(ACCESS_EXISTS) {
		switch {
		case err == nil:
			res = 1
		case errors.Is(err, fs.ErrNotExist):
			res = 0
		default:
			return uint32(IOERR_ACCESS)
		}
	} else if err == nil {
		var want fs.FileMode = syscall.S_IRUSR
		if flags == uint32(ACCESS_READWRITE) {
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
	} else if errors.Is(err, fs.ErrPermission) {
		res = 0
	} else {
		return uint32(IOERR_ACCESS)
	}

	if ok := mod.Memory().WriteUint32Le(pResOut, res); !ok {
		panic(rangeErr)
	}
	return _OK
}

func vfsOpen(ctx context.Context, mod api.Module, pVfs, zName, pFile, flags, pOutFlags uint32) uint32 {
	name := getString(mod.Memory(), zName, _MAX_PATHNAME)
	c := ctx.Value(connContext{}).(*Conn)

	var oflags int
	if OpenFlag(flags)&OPEN_EXCLUSIVE != 0 {
		oflags |= os.O_EXCL
	}
	if OpenFlag(flags)&OPEN_CREATE != 0 {
		oflags |= os.O_CREATE
	}
	if OpenFlag(flags)&OPEN_READONLY != 0 {
		oflags |= os.O_RDONLY
	}
	if OpenFlag(flags)&OPEN_READWRITE != 0 {
		oflags |= os.O_RDWR
	}
	f, err := os.OpenFile(name, oflags, 0600)
	if err != nil {
		return uint32(CANTOPEN)
	}

	if ok := mod.Memory().WriteUint32Le(pFile+ptrSize, c.getFile(f)); !ok {
		panic(rangeErr)
	}
	if pOutFlags == 0 {
		return _OK
	}
	if ok := mod.Memory().WriteUint32Le(pOutFlags, flags); !ok {
		panic(rangeErr)
	}
	return _OK
}

func vfsClose(ctx context.Context, mod api.Module, pFile uint32) uint32 {
	id, ok := mod.Memory().ReadUint32Le(pFile + ptrSize)
	if !ok {
		panic(rangeErr)
	}

	c := ctx.Value(connContext{}).(*Conn)
	err := c.files[id].Close()
	c.files[id] = nil
	if err != nil {
		return uint32(IOERR_CLOSE)
	}
	return _OK
}

func vfsFile(ctx context.Context, mod api.Module, pFile uint32) *os.File {
	id, ok := mod.Memory().ReadUint32Le(pFile + ptrSize)
	if !ok {
		panic(rangeErr)
	}

	c := ctx.Value(connContext{}).(*Conn)
	return c.files[id]
}

func vfsRead(ctx context.Context, mod api.Module, pFile, zBuf, iAmt uint32, iOfst uint64) uint32 {
	mem, ok := mod.Memory().Read(zBuf, iAmt)
	if !ok {
		panic(rangeErr)
	}

	file := vfsFile(ctx, mod, pFile)
	n, err := file.ReadAt(mem, int64(iOfst))
	if n == int(iAmt) {
		return _OK
	}
	if n == 0 && err != io.EOF {
		return uint32(IOERR_READ)
	}
	for i := range mem[n:] {
		mem[i] = 0
	}
	return uint32(IOERR_SHORT_READ)
}

func vfsWrite(ctx context.Context, mod api.Module, pFile, zBuf, iAmt uint32, iOfst uint64) uint32 {
	mem, ok := mod.Memory().Read(zBuf, iAmt)
	if !ok {
		panic(rangeErr)
	}

	file := vfsFile(ctx, mod, pFile)
	_, err := file.WriteAt(mem, int64(iOfst))
	if err != nil {
		return uint32(IOERR_WRITE)
	}
	return _OK
}

func vfsTruncate(ctx context.Context, mod api.Module, pFile uint32, nByte uint64) uint32 {
	file := vfsFile(ctx, mod, pFile)
	err := file.Truncate(int64(nByte))
	if err != nil {
		return uint32(IOERR_TRUNCATE)
	}
	return _OK
}

func vfsSync(ctx context.Context, mod api.Module, pFile, flags uint32) uint32 {
	file := vfsFile(ctx, mod, pFile)
	err := file.Sync()
	if err != nil {
		return uint32(IOERR_FSYNC)
	}
	return _OK
}

func vfsFileSize(ctx context.Context, mod api.Module, pFile, pSize uint32) uint32 {
	file := vfsFile(ctx, mod, pFile)
	off, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return uint32(IOERR_SEEK)
	}

	if ok := mod.Memory().WriteUint64Le(pSize, uint64(off)); !ok {
		panic(rangeErr)
	}
	return _OK
}
