package sqlite3

import (
	"bytes"
	"math"
	"strconv"
	"unsafe"
)

// math.h

func (*env) Xacos(x float64) float64           { return math.Acos(x) }
func (*env) Xacosh(x float64) float64          { return math.Acosh(x) }
func (*env) Xasin(x float64) float64           { return math.Asin(x) }
func (*env) Xasinh(x float64) float64          { return math.Asinh(x) }
func (*env) Xatan(x float64) float64           { return math.Atan(x) }
func (*env) Xatan2(y, x float64) float64       { return math.Atan2(y, x) }
func (*env) Xatanh(x float64) float64          { return math.Atanh(x) }
func (*env) Xcbrt(x float64) float64           { return math.Cbrt(x) }
func (*env) Xcopysign(x, y float64) float64    { return math.Copysign(x, y) }
func (*env) Xcos(x float64) float64            { return math.Cos(x) }
func (*env) Xcosh(x float64) float64           { return math.Cosh(x) }
func (*env) Xerf(x float64) float64            { return math.Erf(x) }
func (*env) Xerfc(x float64) float64           { return math.Erfc(x) }
func (*env) Xexp(x float64) float64            { return math.Exp(x) }
func (*env) Xexp2(x float64) float64           { return math.Exp2(x) }
func (*env) Xexpm1(x float64) float64          { return math.Expm1(x) }
func (*env) Xfabs(x float64) float64           { return math.Abs(x) }
func (*env) Xfdim(x, y float64) float64        { return math.Dim(x, y) }
func (*env) Xfma(x, y, z float64) float64      { return math.FMA(x, y, z) }
func (*env) Xfmod(x, y float64) float64        { return math.Mod(x, y) }
func (*env) Xhypot(x, y float64) float64       { return math.Hypot(x, y) }
func (*env) Xj0(x float64) float64             { return math.J0(x) }
func (*env) Xj1(x float64) float64             { return math.J1(x) }
func (*env) Xjn(n int32, x float64) float64    { return math.Jn(int(n), x) }
func (*env) Xldexp(x float64, n int32) float64 { return math.Ldexp(x, int(n)) }
func (*env) Xlog(x float64) float64            { return math.Log(x) }
func (*env) Xlog10(x float64) float64          { return math.Log10(x) }
func (*env) Xlog1p(x float64) float64          { return math.Log1p(x) }
func (*env) Xlog2(x float64) float64           { return math.Log2(x) }
func (*env) Xlogb(x float64) float64           { return math.Logb(x) }
func (*env) Xnextafter(x, y float64) float64   { return math.Nextafter(x, y) }
func (*env) Xpow(x, y float64) float64         { return math.Pow(x, y) }
func (*env) Xremainder(x, y float64) float64   { return math.Remainder(x, y) }
func (*env) Xround(x float64) float64          { return math.Round(x) }
func (*env) Xsin(x float64) float64            { return math.Sin(x) }
func (*env) Xsinh(x float64) float64           { return math.Sinh(x) }
func (*env) Xsqrt(x float64) float64           { return math.Sqrt(x) }
func (*env) Xtan(x float64) float64            { return math.Tan(x) }
func (*env) Xtanh(x float64) float64           { return math.Tanh(x) }
func (*env) Xtgamma(x float64) float64         { return math.Gamma(x) }
func (*env) Xy0(x float64) float64             { return math.Y0(x) }
func (*env) Xy1(x float64) float64             { return math.Y1(x) }
func (*env) Xyn(n int32, x float64) float64    { return math.Yn(int(n), x) }
func (*env) Xilogb(x float64) int32            { return int32(math.Ilogb(x)) }

func (*env) Xlgamma(x float64) float64 {
	x, _ = math.Lgamma(x)
	return x
}

func (e *env) Xfrexp(x float64, eptr int32) float64 {
	x, exp := math.Frexp(x)
	e.Memory.Write32(ptr_t(eptr), uint32(exp))
	return x
}

func (e *env) Xmodf(x float64, iptr int32) (f float64) {
	if math.IsInf(x, 0) {
		f = math.Copysign(0, x)
	} else {
		x, f = math.Modf(x)
	}
	e.Memory.Write64(ptr_t(iptr), math.Float64bits(x))
	return f
}

func (*env) Xfmax(x, y float64) float64 {
	switch r := max(x, y); {
	case !math.IsNaN(r):
		return r
	case math.IsNaN(x):
		return y
	default:
		return x
	}
}

func (*env) Xfmin(x, y float64) float64 {
	switch r := min(x, y); {
	case !math.IsNaN(r):
		return r
	case math.IsNaN(x):
		return y
	default:
		return x
	}
}

// string.h

func (e *env) Xmemchr(s, c, n int32) int32 {
	m := e.Buf[uint32(s):]
	if uint(len(m)) > uint(uint32(n)) {
		m = m[:uint32(n)]
	}
	if i := bytes.IndexByte(m, byte(c)); i >= 0 {
		return s + int32(i)
	}
	return 0
}

func (e *env) Xmemcmp(s1, s2, n int32) int32 {
	e1, e2 := s1+n, s2+n
	m1 := e.Buf[uint32(s1):uint32(e1)]
	m2 := e.Buf[uint32(s2):uint32(e2)]
	return int32(bytes.Compare(m1, m2))
}

func (e *env) Xstrlen(s int32) int32 {
	return int32(bytes.IndexByte(e.Buf[uint32(s):], 0))
}

func (e *env) Xstrchr(s, c int32) int32 {
	s = e.Xstrchrnul(s, c)
	if e.Buf[uint32(s)] == byte(c) {
		return s
	}
	return 0
}

func (e *env) Xstrchrnul(s, c int32) int32 {
	m := e.Buf[uint32(s):]
	m = m[:bytes.IndexByte(m, 0)]
	l := len(m)
	if b := byte(c); b != 0 {
		if i := bytes.IndexByte(m, b); i >= 0 {
			l = i
		}
	}
	return s + int32(l)
}

func (e *env) Xstrrchr(s, c int32) int32 {
	m := e.Buf[uint32(s):]
	m = m[:bytes.IndexByte(m, 0)+1]
	if i := bytes.LastIndexByte(m, byte(c)); i >= 0 {
		return s + int32(i)
	}
	return 0
}

func (e *env) Xstrcmp(s1, s2 int32) int32 {
	m1 := e.Buf[uint32(s1):]
	m2 := e.Buf[uint32(s2):]
	sz := min(len(m1), len(m2))
	if i := bytes.IndexByte(m1[:sz], 0); i >= 0 {
		sz = i + 1
	}
	return int32(bytes.Compare(m1[:sz], m2[:sz]))
}

func (e *env) Xstrncmp(s1, s2, n int32) int32 {
	m1 := e.Buf[uint32(s1):]
	m2 := e.Buf[uint32(s2):]
	sz := int(min(uint(len(m1)), uint(len(m2)), uint(uint32(n))))
	if i := bytes.IndexByte(m1[:sz], 0); i >= 0 {
		sz = i + 1
	}
	return int32(bytes.Compare(m1[:sz], m2[:sz]))
}

func (e *env) Xstrspn(s, accept int32) int32 {
	m := e.Buf[uint32(s):]
	a := e.Buf[uint32(accept):]
	a = a[:bytes.IndexByte(a, 0)]

	for i, b := range m {
		if bytes.IndexByte(a, b) < 0 {
			return int32(i)
		}
	}
	return int32(len(m))
}

func (e *env) Xstrcspn(s, reject int32) int32 {
	m := e.Buf[uint32(s):]
	r := e.Buf[uint32(reject):]
	r = r[:bytes.IndexByte(r, 0)+1]

	for i, b := range m {
		if bytes.IndexByte(r, b) >= 0 {
			return int32(i)
		}
	}
	return int32(len(m))
}

func (e *env) Xstrstr(haystack, needle int32) int32 {
	h := e.Buf[uint32(haystack):]
	n := e.Buf[uint32(needle):]
	h = h[:bytes.IndexByte(h, 0)]
	n = n[:bytes.IndexByte(n, 0)]
	i := bytes.Index(h, n)
	if i < 0 {
		return 0
	}
	return haystack + int32(i)
}

func (e *env) Xstrcpy(d, s int32) int32 {
	m := e.Buf[uint32(s):]
	m = m[:bytes.IndexByte(m, 0)+1]
	copy(e.Buf[uint32(d):], m)
	return d
}

// stdlib.h

func (e *env) Xstrtod(s, endptr int32) float64 {
	m0 := e.Buf[uint32(s):]
	m1 := bytes.TrimLeft(m0, " \t\n\v\f\r")
	m2 := bytes.TrimLeft(m1, "+-.0123456789abcdefinptxyABCDEFINPTXY")
	spaces := len(m0) - len(m1)
	digits := len(m1) - len(m2)

	var val float64
	for ; digits > 0; digits-- {
		var err error
		str := unsafe.String(&m1[0], digits)
		val, err = strconv.ParseFloat(str, 64)
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
	return val
}

func (e *env) Xstrtol(s, endptr, base int32) int32 {
	m0 := e.Buf[uint32(s):]
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
