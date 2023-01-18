package sqlite3

import (
	"context"
	"log"
	"math/rand"
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
	env.NewFunctionBuilder().WithFunc(vfsOpen).Export("go_open")
	env.NewFunctionBuilder().WithFunc(vfsFullPathname).Export("go_full_pathname")
	env.NewFunctionBuilder().WithFunc(vfsRandomness).Export("go_randomness")
	env.NewFunctionBuilder().WithFunc(vfsSleep).Export("go_sleep")
	env.NewFunctionBuilder().WithFunc(vfsCurrentTime).Export("go_current_time")
	env.NewFunctionBuilder().WithFunc(vfsCurrentTime64).Export("go_current_time_64")
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

func vfsOpen(ctx context.Context, mod api.Module, vfs, zName, file, flags, pOutFlags uint32) uint32 {
	name := getString(mod.Memory(), zName, _MAX_PATHNAME)
	log.Println("vfsOpen", name)
	return uint32(IOERR)
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
