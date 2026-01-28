// Package sqlite3 wraps the C SQLite API.
package sqlite3

import (
	"bytes"
	"crypto/rand"
	"math"
	"math/bits"
	"time"
	_ "unsafe"

	sqlite3_wasm "github.com/ncruces/go-sqlite3-wasm"
	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
	"github.com/ncruces/julianday"
)

var _ sqlite3_wasm.Xenv = &env{}

type env struct{ *sqlite3_wrap.Wrapper }

func createWrapper() (*sqlite3_wrap.Wrapper, error) {
	mem := &sqlite3_wrap.Memory{Max: 4096} // 256MB
	if bits.UintSize < 64 {
		mem.Max = 512 // 32MB
	}
	env := &env{&sqlite3_wrap.Wrapper{Memory: mem}}
	sqlite3_wasm.New(mem, env)
	env.X_initialize()
	return env.Wrapper, nil
}

func (e *env) Init(m *sqlite3_wasm.Module) { e.Module = m }

// Math functions.
func (env) Xacos(x float64) float64     { return math.Acos(x) }
func (env) Xacosh(x float64) float64    { return math.Acosh(x) }
func (env) Xasin(x float64) float64     { return math.Asin(x) }
func (env) Xasinh(x float64) float64    { return math.Asinh(x) }
func (env) Xatan(x float64) float64     { return math.Atan(x) }
func (env) Xatan2(y, x float64) float64 { return math.Atan2(y, x) }
func (env) Xatanh(x float64) float64    { return math.Atanh(x) }
func (env) Xcos(x float64) float64      { return math.Cos(x) }
func (env) Xcosh(x float64) float64     { return math.Cosh(x) }
func (env) Xexp(x float64) float64      { return math.Exp(x) }
func (env) Xfmod(x, y float64) float64  { return math.Mod(x, y) }
func (env) Xlog(x float64) float64      { return math.Log(x) }
func (env) Xlog10(x float64) float64    { return math.Log10(x) }
func (env) Xlog2(x float64) float64     { return math.Log2(x) }
func (env) Xpow(x, y float64) float64   { return math.Pow(x, y) }
func (env) Xsin(x float64) float64      { return math.Sin(x) }
func (env) Xsinh(x float64) float64     { return math.Sinh(x) }
func (env) Xtan(x float64) float64      { return math.Tan(x) }
func (env) Xtanh(x float64) float64     { return math.Tanh(x) }

// String functions.

func (e env) Xstrlen(s int32) int32 {
	return int32(bytes.IndexByte(e.Buf[s:], 0))
}

func (e env) Xmemchr(s, c, n int32) int32 {
	m := e.Buf[s:]
	if len(m) > int(n) {
		m = m[:n]
	}
	if i := bytes.IndexByte(m, byte(c)); i >= 0 {
		return s + int32(i)
	}
	return 0
}

func (e env) Xmemcmp(s1, s2, n int32) int32 {
	e1, e2 := s1+n, s2+n
	m1 := e.Buf[s1:e1]
	m2 := e.Buf[s2:e2]
	return int32(bytes.Compare(m1, m2))
}

func (e env) Xstrchr(s, c int32) int32 {
	s = e.Xstrchrnul(s, c)
	if e.Buf[s] == byte(c) {
		return s
	}
	return 0
}

func (e env) Xstrchrnul(s, c int32) int32 {
	m := e.Buf[s:]
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

func (e env) Xstrrchr(s, c int32) int32 {
	m := e.Buf[s:]
	m = m[:bytes.IndexByte(m, 0)+1]
	if i := bytes.LastIndexByte(m, byte(c)); i >= 0 {
		return s + int32(i)
	}
	return 0
}

func (e env) Xstrcmp(s1, s2 int32) int32 {
	m1 := e.Buf[s1:]
	m2 := e.Buf[s2:]
	m1 = m1[:bytes.IndexByte(m1, 0)]
	m2 = m2[:bytes.IndexByte(m2, 0)]
	return int32(bytes.Compare(m1, m2))
}

func (e env) Xstrncmp(s1, s2, n int32) int32 {
	m1 := e.Buf[s1:]
	m2 := e.Buf[s2:]
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

func (e env) Xstrspn(s, accept int32) int32 {
	m := e.Buf[s:]
	a := e.Buf[accept:]
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

func (e env) Xstrcspn(s, reject int32) int32 {
	m := e.Buf[s:]
	r := e.Buf[reject:]
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

func (e env) Xgo_randomness(pVfs, nByte, zByte int32) int32 {
	mem := e.Slice(ptr_t(zByte), int64(nByte))
	n, _ := rand.Reader.Read(mem)
	return int32(n)
}

func (e env) Xgo_sleep(pVfs, nMicro int32) int32 {
	time.Sleep(time.Duration(nMicro) * time.Microsecond)
	return _OK
}

func (e env) Xgo_current_time_64(pVfs, nMicro int32) int32 {
	day, nsec := julianday.Date(time.Now())
	msec := day*86_400_000 + nsec/1_000_000
	e.Write64(ptr_t(nMicro), uint64(msec))
	return int32(_OK)
}

func (e env) Xgo_localtime(pTm int32, t int64) int32 {
	const size = 32 / 8
	mem := e.Memory
	tm := time.Unix(t, 0)
	// https://pubs.opengroup.org/onlinepubs/7908799/xsh/time.h.html
	mem.Write32(ptr_t(pTm+0*size), uint32(tm.Second()))
	mem.Write32(ptr_t(pTm+1*size), uint32(tm.Minute()))
	mem.Write32(ptr_t(pTm+2*size), uint32(tm.Hour()))
	mem.Write32(ptr_t(pTm+3*size), uint32(tm.Day()))
	mem.Write32(ptr_t(pTm+4*size), uint32(tm.Month()-time.January))
	mem.Write32(ptr_t(pTm+5*size), uint32(tm.Year()-1900))
	mem.Write32(ptr_t(pTm+6*size), uint32(tm.Weekday()-time.Sunday))
	mem.Write32(ptr_t(pTm+7*size), uint32(tm.YearDay()-1))
	mem.WriteBool(ptr_t(pTm+8*size), tm.IsDST())
	return _OK
}

//go:linkname vfsFind github.com/ncruces/go-sqlite3/vfs.vfsFind
func vfsFind(_ *sqlite3_wrap.Wrapper, v0 int32) int32

func (e env) Xgo_vfs_find(v0 int32) int32 {
	return vfsFind(e.Wrapper, v0)
}

//go:linkname vfsFullPathname github.com/ncruces/go-sqlite3/vfs.vfsFullPathname
func vfsFullPathname(_ *sqlite3_wrap.Wrapper, v0, v1, v2, v3 int32) int32

func (e env) Xgo_full_pathname(v0, v1, v2, v3 int32) int32 {
	return vfsFullPathname(e.Wrapper, v0, v1, v2, v3)
}

//go:linkname vfsDelete github.com/ncruces/go-sqlite3/vfs.vfsDelete
func vfsDelete(_ *sqlite3_wrap.Wrapper, v0, v1, v2 int32) int32

func (e env) Xgo_delete(v0, v1, v2 int32) int32 {
	return vfsDelete(e.Wrapper, v0, v1, v2)
}

//go:linkname vfsAccess github.com/ncruces/go-sqlite3/vfs.vfsAccess
func vfsAccess(_ *sqlite3_wrap.Wrapper, v0, v1, v2, v3 int32) int32

func (e env) Xgo_access(v0, v1, v2, v3 int32) int32 {
	return vfsAccess(e.Wrapper, v0, v1, v2, v3)
}

//go:linkname vfsOpen github.com/ncruces/go-sqlite3/vfs.vfsOpen
func vfsOpen(_ *sqlite3_wrap.Wrapper, v0, v1, v2, v3, v4, v5 int32) int32

func (e env) Xgo_open(v0, v1, v2, v3, v4, v5 int32) int32 {
	return vfsOpen(e.Wrapper, v0, v1, v2, v3, v4, v5)
}

//go:linkname vfsClose github.com/ncruces/go-sqlite3/vfs.vfsClose
func vfsClose(_ *sqlite3_wrap.Wrapper, v0 int32) int32

func (e env) Xgo_close(v0 int32) int32 {
	return vfsClose(e.Wrapper, v0)
}

//go:linkname vfsRead github.com/ncruces/go-sqlite3/vfs.vfsRead
func vfsRead(_ *sqlite3_wrap.Wrapper, v0, v1, v2 int32, v3 int64) int32

func (e env) Xgo_read(v0, v1, v2 int32, v3 int64) int32 {
	return vfsRead(e.Wrapper, v0, v1, v2, v3)
}

//go:linkname vfsWrite github.com/ncruces/go-sqlite3/vfs.vfsWrite
func vfsWrite(_ *sqlite3_wrap.Wrapper, v0, v1, v2 int32, v3 int64) int32

func (e env) Xgo_write(v0, v1, v2 int32, v3 int64) int32 {
	return vfsWrite(e.Wrapper, v0, v1, v2, v3)
}

//go:linkname vfsTruncate github.com/ncruces/go-sqlite3/vfs.vfsTruncate
func vfsTruncate(_ *sqlite3_wrap.Wrapper, v0 int32, v1 int64) int32

func (e env) Xgo_truncate(v0 int32, v1 int64) int32 {
	return vfsTruncate(e.Wrapper, v0, v1)
}

//go:linkname vfsSync github.com/ncruces/go-sqlite3/vfs.vfsSync
func vfsSync(_ *sqlite3_wrap.Wrapper, v0, v1 int32) int32

func (e env) Xgo_sync(v0, v1 int32) int32 {
	return vfsSync(e.Wrapper, v0, v1)
}

//go:linkname vfsFileSize github.com/ncruces/go-sqlite3/vfs.vfsFileSize
func vfsFileSize(_ *sqlite3_wrap.Wrapper, v0, v1 int32) int32

func (e env) Xgo_file_size(v0, v1 int32) int32 {
	return vfsFileSize(e.Wrapper, v0, v1)
}

//go:linkname vfsLock github.com/ncruces/go-sqlite3/vfs.vfsLock
func vfsLock(_ *sqlite3_wrap.Wrapper, v0, v1 int32) int32

func (e env) Xgo_lock(v0 int32, v1 int32) int32 {
	return vfsLock(e.Wrapper, v0, v1)
}

//go:linkname vfsUnlock github.com/ncruces/go-sqlite3/vfs.vfsUnlock
func vfsUnlock(_ *sqlite3_wrap.Wrapper, v0, v1 int32) int32

func (e env) Xgo_unlock(v0, v1 int32) int32 {
	return vfsUnlock(e.Wrapper, v0, v1)
}

//go:linkname vfsCheckReservedLock github.com/ncruces/go-sqlite3/vfs.vfsCheckReservedLock
func vfsCheckReservedLock(_ *sqlite3_wrap.Wrapper, v0, v1 int32) int32

func (e env) Xgo_check_reserved_lock(v0, v1 int32) int32 {
	return vfsCheckReservedLock(e.Wrapper, v0, v1)
}

//go:linkname vfsFileControl github.com/ncruces/go-sqlite3/vfs.vfsFileControl
func vfsFileControl(_ *sqlite3_wrap.Wrapper, v0, v1, v2 int32) int32

func (e env) Xgo_file_control(v0, v1, v2 int32) int32 {
	return vfsFileControl(e.Wrapper, v0, v1, v2)
}

//go:linkname vfsSectorSize github.com/ncruces/go-sqlite3/vfs.vfsSectorSize
func vfsSectorSize(_ *sqlite3_wrap.Wrapper, v0 int32) int32

func (e env) Xgo_sector_size(v0 int32) int32 {
	return vfsSectorSize(e.Wrapper, v0)
}

//go:linkname vfsDeviceCharacteristics github.com/ncruces/go-sqlite3/vfs.vfsDeviceCharacteristics
func vfsDeviceCharacteristics(_ *sqlite3_wrap.Wrapper, v0 int32) int32

func (e env) Xgo_device_characteristics(v0 int32) int32 {
	return vfsDeviceCharacteristics(e.Wrapper, v0)
}

//go:linkname vfsShmBarrier github.com/ncruces/go-sqlite3/vfs.vfsShmBarrier
func vfsShmBarrier(_ *sqlite3_wrap.Wrapper, v0 int32)

func (e env) Xgo_shm_barrier(v0 int32) {
	vfsShmBarrier(e.Wrapper, v0)
}

//go:linkname vfsShmMap github.com/ncruces/go-sqlite3/vfs.vfsShmMap
func vfsShmMap(_ *sqlite3_wrap.Wrapper, v0, v1, v2, v3, v4 int32) int32

func (e env) Xgo_shm_map(v0, v1, v2, v3, v4 int32) int32 {
	return vfsShmMap(e.Wrapper, v0, v1, v2, v3, v4)
}

//go:linkname vfsShmLock github.com/ncruces/go-sqlite3/vfs.vfsShmLock
func vfsShmLock(_ *sqlite3_wrap.Wrapper, v0, v1, v2, v3 int32) int32

func (e env) Xgo_shm_lock(v0, v1, v2, v3 int32) int32 {
	return vfsShmLock(e.Wrapper, v0, v1, v2, v3)
}

//go:linkname vfsShmUnmap github.com/ncruces/go-sqlite3/vfs.vfsShmUnmap
func vfsShmUnmap(_ *sqlite3_wrap.Wrapper, v0, v1 int32) int32

func (e env) Xgo_shm_unmap(v0, v1 int32) int32 {
	return vfsShmUnmap(e.Wrapper, v0, v1)
}
