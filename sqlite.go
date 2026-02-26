// Package sqlite3 wraps the C SQLite API.
package sqlite3

import (
	"bytes"
	"context"
	"math"
	"math/bits"
	"strings"

	"github.com/ncruces/go-sqlite3/internal/alloc"
	"github.com/ncruces/go-sqlite3/internal/sqlite3_wasm"
	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/vfs"
)

type sqlite struct {
	ctx context.Context
	cfn context.CancelFunc
	mem *alloc.Memory
	mod *sqlite3_wasm.Module
}

func instantiateSQLite() (sqlt *sqlite, err error) {
	sqlt = new(sqlite)
	sqlt.mem = &alloc.Memory{Max: 4096} // 256MB
	if bits.UintSize < 64 {
		sqlt.mem.Max = 512 // 32MB
	}
	sqlt.ctx, sqlt.cfn = util.NewContext(context.Background())
	sqlite3_wasm.New(&sqlite3_wasm.Memory{Grow: sqlt.mem.Grow}, sqlt)
	sqlt.mod.X_initialize()
	return sqlt, nil
}

func (sqlt *sqlite) Init(mod *sqlite3_wasm.Module) { sqlt.mod = mod }

func (sqlt *sqlite) close() error {
	sqlt.cfn()
	sqlt.mem.Free()
	sqlt.mod = nil
	return nil
}

func (sqlt *sqlite) error(rc res_t, handle ptr_t, sql ...string) error {
	if rc == _OK {
		return nil
	}

	if ErrorCode(rc) == NOMEM || xErrorCode(rc) == IOERR_NOMEM {
		panic(util.OOMErr)
	}

	var msg, query string
	if handle != 0 {
		if ptr := ptr_t(sqlt.mod.Xsqlite3_errmsg(int32(handle))); ptr != 0 {
			msg = util.ReadString(sqlt.mod, ptr, _MAX_LENGTH)
			msg = strings.TrimPrefix(msg, "sqlite3: ")
			msg = strings.TrimPrefix(msg, util.ErrorCodeString(rc)[len("sqlite3: "):])
			msg = strings.TrimPrefix(msg, ": ")
			if msg == "" || msg == "not an error" {
				msg = ""
			}
		}

		if len(sql) != 0 {
			if i := int32(sqlt.mod.Xsqlite3_error_offset(int32(handle))); i != -1 {
				query = sql[0][i:]
			}
		}
	}

	var sys error
	switch ErrorCode(rc) {
	case CANTOPEN, IOERR:
		sys = util.GetSystemError(sqlt.ctx)
	}

	if sys != nil || msg != "" || query != "" {
		return &Error{code: rc, sys: sys, msg: msg, sql: query}
	}
	return xErrorCode(rc)
}

func (sqlt *sqlite) free(ptr ptr_t) {
	if ptr == 0 {
		return
	}
	sqlt.mod.Xsqlite3_free(int32(ptr))
}

func (sqlt *sqlite) new(size int64) ptr_t {
	ptr := ptr_t(sqlt.mod.Xsqlite3_malloc64(size))
	if ptr == 0 && size != 0 {
		panic(util.OOMErr)
	}
	return ptr
}

func (sqlt *sqlite) realloc(ptr ptr_t, size int64) ptr_t {
	ptr = ptr_t(sqlt.mod.Xsqlite3_realloc64(int32(ptr), size))
	if ptr == 0 && size != 0 {
		panic(util.OOMErr)
	}
	return ptr
}

func (sqlt *sqlite) newBytes(b []byte) ptr_t {
	if len(b) == 0 {
		return 0
	}
	ptr := sqlt.new(int64(len(b)))
	util.WriteBytes(sqlt.mod, ptr, b)
	return ptr
}

func (sqlt *sqlite) newString(s string) ptr_t {
	ptr := sqlt.new(int64(len(s)) + 1)
	util.WriteString(sqlt.mod, ptr, s)
	return ptr
}

const arenaSize = 4096

func (sqlt *sqlite) newArena() arena {
	return arena{
		sqlt: sqlt,
		base: sqlt.new(arenaSize),
	}
}

type arena struct {
	sqlt *sqlite
	ptrs []ptr_t
	base ptr_t
	next int32
}

func (a *arena) free() {
	if a.sqlt == nil {
		return
	}
	for _, ptr := range a.ptrs {
		a.sqlt.free(ptr)
	}
	a.sqlt.free(a.base)
	a.sqlt = nil
}

func (a *arena) mark() (reset func()) {
	ptrs := len(a.ptrs)
	next := a.next
	return func() {
		rest := a.ptrs[ptrs:]
		for _, ptr := range a.ptrs[:ptrs] {
			a.sqlt.free(ptr)
		}
		a.ptrs = rest
		a.next = next
	}
}

func (a *arena) new(size int64) ptr_t {
	// Align the next address, to 4 or 8 bytes.
	if size&7 != 0 {
		a.next = (a.next + 3) &^ 3
	} else {
		a.next = (a.next + 7) &^ 7
	}
	if size <= arenaSize-int64(a.next) {
		ptr := a.base + ptr_t(a.next)
		a.next += int32(size)
		return ptr_t(ptr)
	}
	ptr := a.sqlt.new(size)
	a.ptrs = append(a.ptrs, ptr)
	return ptr_t(ptr)
}

func (a *arena) bytes(b []byte) ptr_t {
	if len(b) == 0 {
		return 0
	}
	ptr := a.new(int64(len(b)))
	util.WriteBytes(a.sqlt.mod, ptr, b)
	return ptr
}

func (a *arena) string(s string) ptr_t {
	ptr := a.new(int64(len(s)) + 1)
	util.WriteString(a.sqlt.mod, ptr, s)
	return ptr
}

// Math functions.
func (sqlt *sqlite) Xacos(x float64) float64     { return math.Acos(x) }
func (sqlt *sqlite) Xacosh(x float64) float64    { return math.Acosh(x) }
func (sqlt *sqlite) Xasin(x float64) float64     { return math.Asin(x) }
func (sqlt *sqlite) Xasinh(x float64) float64    { return math.Asinh(x) }
func (sqlt *sqlite) Xatan(x float64) float64     { return math.Atan(x) }
func (sqlt *sqlite) Xatan2(y, x float64) float64 { return math.Atan2(y, x) }
func (sqlt *sqlite) Xatanh(x float64) float64    { return math.Atanh(x) }
func (sqlt *sqlite) Xcos(x float64) float64      { return math.Cos(x) }
func (sqlt *sqlite) Xcosh(x float64) float64     { return math.Cosh(x) }
func (sqlt *sqlite) Xexp(x float64) float64      { return math.Exp(x) }
func (sqlt *sqlite) Xfmod(x, y float64) float64  { return math.Mod(x, y) }
func (sqlt *sqlite) Xlog(x float64) float64      { return math.Log(x) }
func (sqlt *sqlite) Xlog10(x float64) float64    { return math.Log10(x) }
func (sqlt *sqlite) Xlog2(x float64) float64     { return math.Log2(x) }
func (sqlt *sqlite) Xpow(x, y float64) float64   { return math.Pow(x, y) }
func (sqlt *sqlite) Xsin(x float64) float64      { return math.Sin(x) }
func (sqlt *sqlite) Xsinh(x float64) float64     { return math.Sinh(x) }
func (sqlt *sqlite) Xtan(x float64) float64      { return math.Tan(x) }
func (sqlt *sqlite) Xtanh(x float64) float64     { return math.Tanh(x) }

// String functions.

func (sqlt *sqlite) Xstrlen(s int32) int32 {
	return int32(bytes.IndexByte(sqlt.mod.Data[s:], 0))
}

func (sqlt *sqlite) Xmemchr(s, c, n int32) int32 {
	m := sqlt.mod.Data[s:]
	if len(m) > int(n) {
		m = m[:n]
	}
	if i := bytes.IndexByte(m, byte(c)); i >= 0 {
		return s + int32(i)
	}
	return 0
}

func (sqlt *sqlite) Xmemcmp(s1, s2, n int32) int32 {
	e1, e2 := s1+n, s2+n
	m1 := sqlt.mod.Data[s1:e1]
	m2 := sqlt.mod.Data[s2:e2]
	return int32(bytes.Compare(m1, m2))
}

func (sqlt *sqlite) Xstrchr(s, c int32) int32 {
	s = sqlt.Xstrchrnul(s, c)
	if sqlt.mod.Data[s] == byte(c) {
		return s
	}
	return 0
}

func (sqlt *sqlite) Xstrchrnul(s, c int32) int32 {
	m := sqlt.mod.Data[s:]
	m = m[:bytes.IndexByte(m, 0)]
	b := byte(c)
	l := len(m)
	if c != 0 {
		i := bytes.IndexByte(m, b)
		if i >= 0 {
			l = i
		}
	}
	return s + int32(l)
}

func (sqlt *sqlite) Xstrrchr(s, c int32) int32 {
	m := sqlt.mod.Data[s:]
	m = m[:bytes.IndexByte(m, 0)+1]
	if i := bytes.LastIndexByte(m, byte(c)); i >= 0 {
		return s + int32(i)
	}
	return 0
}

func (sqlt *sqlite) Xstrcmp(s1, s2 int32) int32 {
	m1 := sqlt.mod.Data[s1:]
	m2 := sqlt.mod.Data[s2:]
	m1 = m1[:bytes.IndexByte(m1, 0)]
	m2 = m2[:bytes.IndexByte(m2, 0)]
	return int32(bytes.Compare(m1, m2))
}

func (sqlt *sqlite) Xstrncmp(s1, s2, n int32) int32 {
	m1 := sqlt.mod.Data[s1:]
	m2 := sqlt.mod.Data[s2:]
	m1 = m1[:bytes.IndexByte(m1, 0)]
	m2 = m2[:bytes.IndexByte(m2, 0)]
	if len(m1) > int(n) {
		m1 = m1[:n]
	}
	if len(m2) > int(n) {
		m2 = m2[:n]
	}
	return int32(bytes.Compare(m1, m2))
}

func (sqlt *sqlite) Xstrspn(s, accept int32) int32 {
	m := sqlt.mod.Data[s:]
	a := sqlt.mod.Data[accept:]
	a = a[:bytes.IndexByte(a, 0)]

	i := int32(0)
	for _, b := range m {
		if bytes.IndexByte(a, b) == -1 {
			break
		}
		i++
	}
	return i
}

func (sqlt *sqlite) Xstrcspn(s, reject int32) int32 {
	m := sqlt.mod.Data[s:]
	r := sqlt.mod.Data[reject:]
	r = r[:bytes.IndexByte(r, 0)]

	i := int32(0)
	for _, b := range m {
		if b == 0 || bytes.IndexByte(r, b) != -1 {
			break
		}
		i++
	}
	return i
}

// VFS functions.

func (sqlt *sqlite) Xgo_vfs_find(v0 int32) int32 {
	return int32(vfs.VfsFind(sqlt.ctx, sqlt.mod, ptr_t(v0)))
}

func (sqlt *sqlite) Xgo_localtime(v0 int32, v1 int64) int32 {
	return int32(vfs.VfsLocaltime(sqlt.ctx, sqlt.mod, ptr_t(v0), v1))
}

func (sqlt *sqlite) Xgo_randomness(v0, v1, v2 int32) int32 {
	return int32(vfs.VfsRandomness(sqlt.ctx, sqlt.mod, ptr_t(v0), v1, ptr_t(v2)))
}

func (sqlt *sqlite) Xgo_sleep(v0, v1 int32) int32 {
	return int32(vfs.VfsSleep(sqlt.ctx, sqlt.mod, ptr_t(v0), v1))
}

func (sqlt *sqlite) Xgo_current_time_64(v0, v1 int32) int32 {
	return int32(vfs.VfsCurrentTime64(sqlt.ctx, sqlt.mod, ptr_t(v0), ptr_t(v1)))
}

func (sqlt *sqlite) Xgo_full_pathname(v0, v1, v2, v3 int32) int32 {
	return int32(vfs.VfsFullPathname(sqlt.ctx, sqlt.mod, ptr_t(v0), ptr_t(v1), v2, ptr_t(v3)))
}

func (sqlt *sqlite) Xgo_delete(v0, v1, v2 int32) int32 {
	return int32(vfs.VfsDelete(sqlt.ctx, sqlt.mod, ptr_t(v0), ptr_t(v1), v2))
}
func (sqlt *sqlite) Xgo_access(v0, v1, v2, v3 int32) int32 {
	return int32(vfs.VfsAccess(sqlt.ctx, sqlt.mod, ptr_t(v0), ptr_t(v1), vfs.AccessFlag(v2), ptr_t(v3)))
}

func (sqlt *sqlite) Xgo_open(v0, v1, v2, v3, v4, v5 int32) int32 {
	return int32(vfs.VfsOpen(sqlt.ctx, sqlt.mod, ptr_t(v0), ptr_t(v1), ptr_t(v2), vfs.OpenFlag(v3), ptr_t(v4), ptr_t(v5)))
}

func (sqlt *sqlite) Xgo_close(v0 int32) int32 {
	return int32(vfs.VfsClose(sqlt.ctx, sqlt.mod, ptr_t(v0)))
}

func (sqlt *sqlite) Xgo_read(v0, v1, v2 int32, v3 int64) int32 {
	return int32(vfs.VfsRead(sqlt.ctx, sqlt.mod, ptr_t(v0), ptr_t(v1), v2, v3))
}

func (sqlt *sqlite) Xgo_write(v0, v1, v2 int32, v3 int64) int32 {
	return int32(vfs.VfsWrite(sqlt.ctx, sqlt.mod, ptr_t(v0), ptr_t(v1), v2, v3))
}

func (sqlt *sqlite) Xgo_truncate(v0 int32, v1 int64) int32 {
	return int32(vfs.VfsTruncate(sqlt.ctx, sqlt.mod, ptr_t(v0), v1))
}

func (sqlt *sqlite) Xgo_sync(v0, v1 int32) int32 {
	return int32(vfs.VfsSync(sqlt.ctx, sqlt.mod, ptr_t(v0), vfs.SyncFlag(v1)))
}

func (sqlt *sqlite) Xgo_file_size(v0, v1 int32) int32 {
	return int32(vfs.VfsFileSize(sqlt.ctx, sqlt.mod, ptr_t(v0), ptr_t(v1)))
}

func (sqlt *sqlite) Xgo_lock(v0 int32, v1 int32) int32 {
	return int32(vfs.VfsLock(sqlt.ctx, sqlt.mod, ptr_t(v0), vfs.LockLevel(v1)))
}

func (sqlt *sqlite) Xgo_unlock(v0, v1 int32) int32 {
	return int32(vfs.VfsUnlock(sqlt.ctx, sqlt.mod, ptr_t(v0), vfs.LockLevel(v1)))
}

func (sqlt *sqlite) Xgo_check_reserved_lock(v0, v1 int32) int32 {
	return int32(vfs.VfsCheckReservedLock(sqlt.ctx, sqlt.mod, ptr_t(v0), ptr_t(v1)))
}

func (sqlt *sqlite) Xgo_file_control(v0, v1, v2 int32) int32 {
	return int32(vfs.VfsFileControl(sqlt.ctx, sqlt.mod, ptr_t(v0), vfs.FcntlOpcode(v1), ptr_t(v2)))
}

func (sqlt *sqlite) Xgo_sector_size(v0 int32) int32 {
	return int32(vfs.VfsSectorSize(sqlt.ctx, sqlt.mod, ptr_t(v0)))
}

func (sqlt *sqlite) Xgo_device_characteristics(v0 int32) int32 {
	return int32(vfs.VfsDeviceCharacteristics(sqlt.ctx, sqlt.mod, ptr_t(v0)))
}

func (sqlt *sqlite) Xgo_shm_barrier(v0 int32) {
	vfs.VfsShmBarrier(sqlt.ctx, sqlt.mod, ptr_t(v0))
}

func (sqlt *sqlite) Xgo_shm_map(v0, v1, v2, v3, v4 int32) int32 {
	return int32(vfs.VfsShmMap(sqlt.ctx, sqlt.mod, ptr_t(v0), v1, v2, v3, ptr_t(v4)))
}

func (sqlt *sqlite) Xgo_shm_lock(v0, v1, v2, v3 int32) int32 {
	return int32(vfs.VfsShmLock(sqlt.ctx, sqlt.mod, ptr_t(v0), v1, v2, vfs.ShmFlag(v3)))
}

func (sqlt *sqlite) Xgo_shm_unmap(v0, v1 int32) int32 {
	return int32(vfs.VfsShmUnmap(sqlt.ctx, sqlt.mod, ptr_t(v0), v1))
}
