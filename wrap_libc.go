package sqlite3

import (
	"bytes"
	"math"
	"strconv"
	"unsafe"
)

// math.h

func (*env) Xacos(x float64) float64     { return math.Acos(x) }
func (*env) Xacosh(x float64) float64    { return math.Acosh(x) }
func (*env) Xasin(x float64) float64     { return math.Asin(x) }
func (*env) Xasinh(x float64) float64    { return math.Asinh(x) }
func (*env) Xatan(x float64) float64     { return math.Atan(x) }
func (*env) Xatan2(y, x float64) float64 { return math.Atan2(y, x) }
func (*env) Xatanh(x float64) float64    { return math.Atanh(x) }
func (*env) Xcos(x float64) float64      { return math.Cos(x) }
func (*env) Xcosh(x float64) float64     { return math.Cosh(x) }
func (*env) Xexp(x float64) float64      { return math.Exp(x) }
func (*env) Xfmod(x, y float64) float64  { return math.Mod(x, y) }
func (*env) Xlog(x float64) float64      { return math.Log(x) }
func (*env) Xlog10(x float64) float64    { return math.Log10(x) }
func (*env) Xlog2(x float64) float64     { return math.Log2(x) }
func (*env) Xpow(x, y float64) float64   { return math.Pow(x, y) }
func (*env) Xsin(x float64) float64      { return math.Sin(x) }
func (*env) Xsinh(x float64) float64     { return math.Sinh(x) }
func (*env) Xtan(x float64) float64      { return math.Tan(x) }
func (*env) Xtanh(x float64) float64     { return math.Tanh(x) }

// string.h

func (e *env) Xmemchr(s, c, n int32) int32 {
	m := e.Buf[s:]
	if len(m) > int(n) {
		m = m[:n]
	}
	if i := bytes.IndexByte(m, byte(c)); i >= 0 {
		return s + int32(i)
	}
	return 0
}

func (e *env) Xmemcmp(s1, s2, n int32) int32 {
	e1, e2 := s1+n, s2+n
	m1 := e.Buf[s1:e1]
	m2 := e.Buf[s2:e2]
	return int32(bytes.Compare(m1, m2))
}

func (e *env) Xstrlen(s int32) int32 {
	return int32(bytes.IndexByte(e.Buf[s:], 0))
}

func (e *env) Xstrchr(s, c int32) int32 {
	s = e.Xstrchrnul(s, c)
	if e.Buf[s] == byte(c) {
		return s
	}
	return 0
}

func (e *env) Xstrchrnul(s, c int32) int32 {
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

func (e *env) Xstrrchr(s, c int32) int32 {
	m := e.Buf[s:]
	m = m[:bytes.IndexByte(m, 0)+1]
	if i := bytes.LastIndexByte(m, byte(c)); i >= 0 {
		return s + int32(i)
	}
	return 0
}

func (e *env) Xstrcmp(s1, s2 int32) int32 {
	m1 := e.Buf[s1:]
	m2 := e.Buf[s2:]
	m1 = m1[:bytes.IndexByte(m1, 0)]
	m2 = m2[:bytes.IndexByte(m2, 0)]
	return int32(bytes.Compare(m1, m2))
}

func (e *env) Xstrncmp(s1, s2, n int32) int32 {
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

func (e *env) Xstrspn(s, accept int32) int32 {
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

func (e *env) Xstrcspn(s, reject int32) int32 {
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

func (e *env) Xstrstr(haystack, needle int32) int32 {
	h := e.Buf[haystack:]
	n := e.Buf[needle:]
	h = h[:bytes.IndexByte(h, 0)]
	n = n[:bytes.IndexByte(n, 0)]
	i := bytes.Index(h, n)
	if i < 0 {
		return 0
	}
	return haystack + int32(i)
}

func (e *env) Xstrcpy(d, s int32) int32 {
	m := e.Buf[s:]
	m = m[:bytes.IndexByte(m, 0)+1]
	copy(e.Buf[d:], m)
	return d
}

// stdlib.h

func (e *env) Xstrtod(s, endptr int32) float64 {
	m0 := e.Buf[s:]
	m1 := bytes.TrimLeft(m0, " \t\n\v\f\r")
	m2 := bytes.TrimLeft(m1, "+-.0123456789abcdefinptxyABCDEFINPTXY")
	spaces := len(m0) - len(m1)
	digits := len(m1) - len(m2)

	var val float64
	for ; digits > 0; digits-- {
		var err error
		str := unsafe.String(&m1[0], digits)
		val, err = strconv.ParseFloat(str, 64)
		if err == nil {
			break
		}
	}

	if endptr != 0 {
		if digits > 0 {
			s += int32(spaces + digits)
		}
		e.Memory.Write32(ptr_t(endptr), uint32(s))
	}
	return val
}

func (e *env) Xstrtol(s, endptr, base int32) int32 {
	m0 := e.Buf[s:]
	m1 := bytes.TrimLeft(m0, " \t\n\v\f\r")
	m2 := bytes.TrimLeft(m1, "+-0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	spaces := len(m0) - len(m1)
	digits := len(m1) - len(m2)

	var val int64
	for ; digits > 0; digits-- {
		var err error
		str := unsafe.String(&m1[0], digits)
		val, err = strconv.ParseInt(str, int(base), 32)
		if e, ok := err.(*strconv.NumError); !ok || e.Err == strconv.ErrRange {
			break
		}
	}

	if endptr != 0 {
		if digits > 0 {
			s += int32(spaces + digits)
		}
		e.Memory.Write32(ptr_t(endptr), uint32(s))
	}
	return int32(val)
}
