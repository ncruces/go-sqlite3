package sqlite3

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
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

func vfsRandomness(ctx context.Context, mod api.Module, vfs, nByte, zOut uint32) uint32 {
	mem, ok := mod.Memory().Read(zOut, nByte)
	if !ok {
		return 0
	}
	n, _ := rand.Read(mem)
	return uint32(n)
}

func vfsSleep(ctx context.Context, vfs, microseconds uint32) uint32 {
	time.Sleep(time.Duration(microseconds) * time.Microsecond)
	return _OK
}

func vfsCurrentTime(ctx context.Context, mod api.Module, vfs, out uint32) uint32 {
	day := julianday.Float(time.Now())
	ok := mod.Memory().WriteFloat64Le(out, day)
	if !ok {
		return uint32(ERROR)
	}
	return _OK
}

func vfsCurrentTime64(ctx context.Context, mod api.Module, vfs, out uint32) uint32 {
	day, nsec := julianday.Date(time.Now())
	msec := day*86_400_000 + nsec/1_000_000
	ok := mod.Memory().WriteUint64Le(out, uint64(msec))
	if !ok {
		return uint32(ERROR)
	}
	return _OK
}

func vfsFullPathname(ctx context.Context, mod api.Module, vfs, zName, nOut, zOut uint32) uint32 {
	name := getString(mod.Memory(), zName, _MAX_PATHNAME)
	s, err := filepath.Abs(name)
	if err != nil {
		return uint32(IOERR)
	}

	siz := uint32(len(s) + 1)
	if siz > zOut {
		return uint32(IOERR)
	}
	mem, ok := mod.Memory().Read(zOut, siz)
	if !ok {
		return uint32(IOERR)
	}

	mem[len(s)] = 0
	copy(mem, s)
	return _OK
}

func vfsDelete(vfs, zName, syncDir uint32) uint32 { panic("vfsDelete") }

func vfsAccess(vfs, zName, flags, pResOut uint32) uint32 { panic("vfsAccess") }

func vfsOpen(ctx context.Context, mod api.Module, vfs, zName, file, flags, pOutFlags uint32) uint32 {
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

	var id int
	for i := range c.files {
		if c.files[i] == nil {
			id = i
			c.files[i] = f
			goto found
		}
	}
	id = len(c.files)
	c.files = append(c.files, f)
found:

	mod.Memory().WriteUint32Le(file+ptrSize, uint32(id))
	return _OK
}

func vfsClose(ctx context.Context, mod api.Module, file uint32) uint32 {
	id, ok := mod.Memory().ReadUint32Le(file + ptrSize)
	if !ok {
		panic("sqlite: out-of-range")
	}

	c := ctx.Value(connContext{}).(*Conn)
	err := c.files[id].Close()
	c.files[id] = nil
	if err != nil {
		return uint32(IOERR)
	}
	return _OK
}

func vfsRead(file, buf, iAmt uint32, iOfst uint64) uint32 {
	return uint32(IOERR)
}

func vfsWrite(file, buf, iAmt uint32, iOfst uint64) uint32 { panic("vfsWrite") }

func vfsTruncate(file uint32, size uint64) uint32 { panic("vfsTruncate") }

func vfsSync(file, flags uint32) uint32 { panic("vfsSync") }

func vfsFileSize(file, pSize uint32) uint32 { panic("vfsFileSize") }
