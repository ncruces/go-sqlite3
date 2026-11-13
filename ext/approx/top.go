package approx

import (
	"cmp"
	"encoding/binary"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"unsafe"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/errutil"
	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/sort/quick"
)

const (
	topHeader      = 1<<4 | 6 // HLL version 1, with an unassigned storage type
	topMaxK        = 1000
	topMinC        = 128
	topMaxC        = 4 * topMaxK
	topSummaryTag  = 0xff
	errInvalidTopK = errutil.ErrorString("approx: invalid top-K sketch")
)

func topCap(k int16) int16 {
	return max(topMinC, 4*k)
}

type topSketch struct {
	ptr    unsafe.Pointer // nil (empty/num), unsafe.StringData(str) (TEXT), or *topSummary
	num    uint64         // int64 bits (INTEGER), float64 bits (FLOAT), or len(str) (TEXT)
	weight float64        // inline weight in unit state (> 0)
	k      int16          // target K in [1, 1000] (0 if Step never ran)
	typ    uint8          // 0 (empty), INTEGER, FLOAT, TEXT, or topSummaryTag
}

func (b *topSketch) summary() *topSummary {
	return (*topSummary)(b.ptr)
}

func (b *topSketch) unitStr() string {
	if b.typ != uint8(sqlite3.TEXT) {
		return ""
	}
	return unsafe.String((*byte)(b.ptr), int(b.num))
}

func (b *topSketch) Value(ctx sqlite3.Context) {
	switch b.typ {
	case 0:
		return
	case topSummaryTag:
		b.summary().evalResult(ctx, int(b.summary().k))
	default:
		var buf [64]byte
		out, err := appendEntryJSON(append(buf[:0], '['), topEntry{
			str:    b.unitStr(),
			num:    b.num,
			weight: b.weight,
			typ:    b.typ,
		}, 0)
		if err != nil {
			ctx.ResultError(err)
			return
		}
		out = append(out, ']')
		ctx.ResultRawText(out)
		ctx.ResultSubtype('J')
	}
}

func (b *topSketch) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if b.k == 0 && b.typ != topSummaryTag {
		k := arg[len(arg)-1].Int()
		if k < 1 || k > topMaxK {
			ctx.ResultError(fmt.Errorf("approx: invalid k: %d", k))
			return
		}
		b.k = int16(k)
	}

	t := arg[0].Type()
	if t == sqlite3.BLOB {
		ctx.ResultError(errutil.ErrorString("approx: JSON cannot hold BLOB values"))
		return
	}

	w := 1.0
	if len(arg) == 3 {
		if arg[1].NumericType() > sqlite3.FLOAT {
			return
		}
		w = arg[1].Float()
		if w <= 0 || math.IsInf(w, 1) {
			if w < 0 || math.IsInf(w, 1) {
				ctx.ResultError(fmt.Errorf("approx: invalid weight: %g", w))
			}
			return
		}
	}

	switch t {
	case sqlite3.NULL:
		return
	case sqlite3.INTEGER:
		b.insertNum(uint8(sqlite3.INTEGER), uint64(arg[0].Int64()), w)
	case sqlite3.FLOAT:
		f := arg[0].Float()
		if i, ok := floatIsInt64(f); ok {
			b.insertNum(uint8(sqlite3.INTEGER), uint64(i), w)
		} else {
			b.insertNum(uint8(sqlite3.FLOAT), math.Float64bits(f), w)
		}
	case sqlite3.TEXT:
		b.insertRawText(arg[0].RawText(), w)
	}
}

func (b *topSketch) insertNum(typ uint8, num uint64, w float64) {
	switch b.typ {
	case 0:
		b.typ = typ
		b.num = num
		b.weight = w
	case topSummaryTag:
		b.summary().addNum(typ, num, w)
	default:
		if b.typ == typ && b.num == num {
			b.weight += w
			return
		}
		s := b.promote()
		s.addNum(typ, num, w)
	}
}

func (b *topSketch) insertRawText(raw []byte, w float64) {
	if b.typ == uint8(sqlite3.TEXT) && string(raw) == b.unitStr() {
		b.weight += w
		return
	}
	b.insertText(string(raw), w)
}

func (b *topSketch) insertText(str string, w float64) {
	switch b.typ {
	case 0:
		b.typ = uint8(sqlite3.TEXT)
		b.ptr = unsafe.Pointer(unsafe.StringData(str))
		b.num = uint64(len(str))
		b.weight = w
	case topSummaryTag:
		b.summary().addText(str, w)
	default:
		if b.typ == uint8(sqlite3.TEXT) && str == b.unitStr() {
			b.weight += w
			return
		}
		s := b.promote()
		s.addText(str, w)
	}
}

func (b *topSketch) promote() *topSummary {
	s := &topSummary{k: b.k, c: topCap(b.k)}
	if b.typ == uint8(sqlite3.TEXT) {
		s.addText(b.unitStr(), b.weight)
	} else {
		s.addNum(b.typ, b.num, b.weight)
	}
	*b = topSketch{
		ptr: unsafe.Pointer(s),
		typ: topSummaryTag,
	}
	return s
}

func (b *topSketch) Combine(other topSketch) {
	if other.typ == 0 {
		if b.k == 0 && b.typ != topSummaryTag {
			b.k = other.k
		}
		return
	}
	if b.k == 0 && b.typ != topSummaryTag {
		b.k = other.k
	}
	switch other.typ {
	case uint8(sqlite3.INTEGER), uint8(sqlite3.FLOAT):
		b.insertNum(other.typ, other.num, other.weight)
	case uint8(sqlite3.TEXT):
		b.insertText(other.unitStr(), other.weight)
	case topSummaryTag:
		os := other.summary()
		switch b.typ {
		case 0:
			*b = topSketch{
				ptr: unsafe.Pointer(os.clone()),
				typ: topSummaryTag,
			}
		case topSummaryTag:
			b.summary().merge(os)
		default:
			s := os.clone()
			if b.typ == uint8(sqlite3.TEXT) {
				s.addText(b.unitStr(), b.weight)
			} else {
				s.addNum(b.typ, b.num, b.weight)
			}
			*b = topSketch{
				ptr: unsafe.Pointer(s),
				typ: topSummaryTag,
			}
		}
	}
}

type topAccumulate struct{ topSketch }

func (b *topAccumulate) Value(ctx sqlite3.Context) {
	var buf [16]byte // fits empty and single-row numeric sketches
	ctx.ResultBlob(b.appendBytes(buf[:0]))
}

func (b *topAccumulate) Combine(other topAccumulate) {
	b.topSketch.Combine(other.topSketch)
}

type topCombine struct {
	sketch *topSummary
}

func (b *topCombine) Value(ctx sqlite3.Context) {
	if b.sketch != nil {
		var buf [16]byte
		ctx.ResultBlob(b.sketch.appendBytes(buf[:0]))
	}
}

func (b *topCombine) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	switch t := arg[0].Type(); t {
	case sqlite3.NULL:
		return
	case sqlite3.BLOB:
		other, err := topFromBytes(arg[0].RawBlob())
		if err != nil {
			ctx.ResultError(err)
			return
		}
		if b.sketch == nil {
			b.sketch = other
			return
		}
		b.sketch.merge(other)
	default:
		ctx.ResultError(fmt.Errorf("approx: invalid type: %v", t))
	}
}

func (b *topCombine) Combine(other topCombine) {
	if other.sketch == nil {
		return
	}
	if b.sketch == nil {
		b.sketch = other.sketch.clone()
		return
	}
	b.sketch.merge(other.sketch)
}

func topEstimate(ctx sqlite3.Context, arg ...sqlite3.Value) {
	switch t := arg[0].Type(); t {
	case sqlite3.NULL:
		return
	case sqlite3.BLOB:
		sketch, err := topFromBytes(arg[0].RawBlob())
		if err != nil {
			ctx.ResultError(err)
			return
		}
		var k int
		if len(arg) == 1 {
			if sketch.k == 0 {
				return
			}
			k = int(sketch.k)
		} else {
			k = arg[1].Int()
			maxK := topMaxC
			if sketch.c > 0 {
				maxK = int(sketch.c)
			}
			if k < 1 || k > maxK {
				ctx.ResultError(fmt.Errorf("approx: invalid k: %d", k))
				return
			}
			if sketch.k == 0 {
				return
			}
		}
		sketch.evalResult(ctx, k)
	default:
		ctx.ResultError(fmt.Errorf("approx: invalid type: %v", t))
	}
}

// topSummary implements a weighted mergeable Misra-Gries frequent items summary
// (Anderson et al., IMC 2017).
type topSummary struct {
	ints    map[int64]float64
	reals   map[float64]float64
	texts   map[string]float64
	scratch []float64
	offset  float64
	k, c    int16
}

func (s *topSummary) len() int {
	return len(s.ints) + len(s.reals) + len(s.texts)
}

func (s *topSummary) addNum(typ uint8, num uint64, w float64) {
	if typ == uint8(sqlite3.INTEGER) {
		addMap(s, &s.ints, int64(num), w)
	} else {
		addMap(s, &s.reals, math.Float64frombits(num), w)
	}
}

func (s *topSummary) addText(k string, w float64) {
	addMap(s, &s.texts, k, w)
}

func addMap[K comparable](s *topSummary, m *map[K]float64, k K, w float64) {
	if *m == nil {
		*m = make(map[K]float64)
	} else if cur, ok := (*m)[k]; ok {
		(*m)[k] = cur + w
		return
	}
	if s.len() >= 2*int(s.c) {
		s.purge()
	}
	(*m)[k] = w
}

func (s *topSummary) purge() {
	n := s.len()
	if n <= int(s.c) {
		return
	}
	w := slices.Grow(s.scratch[:0], n)
	for _, v := range s.ints {
		w = append(w, v)
	}
	for _, v := range s.reals {
		w = append(w, v)
	}
	for _, v := range s.texts {
		w = append(w, v)
	}

	delta := quick.Select(w, len(w)-int(s.c)-1)
	s.offset += delta
	purgeMap(s.ints, delta)
	purgeMap(s.reals, delta)
	purgeMap(s.texts, delta)
	s.scratch = w[:0]
}

func purgeMap[K comparable](m map[K]float64, delta float64) {
	for k, v := range m {
		if v <= delta {
			delete(m, k)
		} else {
			m[k] = v - delta
		}
	}
}

func (s *topSummary) clone() *topSummary {
	return &topSummary{
		ints:   maps.Clone(s.ints),
		reals:  maps.Clone(s.reals),
		texts:  maps.Clone(s.texts),
		offset: s.offset,
		k:      s.k,
		c:      s.c,
	}
}

func (s *topSummary) merge(other *topSummary) {
	if other.k == 0 {
		return
	}
	if s.k == 0 {
		*s = *other.clone()
		return
	}
	s.k = min(s.k, other.k)
	s.c = min(s.c, other.c)
	s.offset += other.offset
	mergeMap(&s.ints, other.ints)
	mergeMap(&s.reals, other.reals)
	mergeMap(&s.texts, other.texts)
	if s.len() >= 2*int(s.c) {
		s.purge()
	}
}

func mergeMap[K comparable](dst *map[K]float64, src map[K]float64) {
	if len(src) == 0 {
		return
	}
	if *dst == nil {
		*dst = maps.Clone(src)
		return
	}
	for k, v := range src {
		(*dst)[k] += v
	}
}

type topEntry struct {
	str    string
	num    uint64
	weight float64
	typ    uint8
}

func cmpEntry(a, b topEntry) int {
	if c := cmp.Compare(b.weight, a.weight); c != 0 {
		return c
	}
	aText := a.typ == uint8(sqlite3.TEXT)
	bText := b.typ == uint8(sqlite3.TEXT)
	if aText != bText {
		if aText {
			return +1
		}
		return -1
	}
	if aText {
		return cmp.Compare(a.str, b.str)
	}
	if a.typ == b.typ {
		if a.typ == uint8(sqlite3.INTEGER) {
			return cmp.Compare(int64(a.num), int64(b.num))
		}
		return cmp.Compare(math.Float64frombits(a.num), math.Float64frombits(b.num))
	}
	// Mixed INTEGER and non-integral FLOAT.
	if a.typ == uint8(sqlite3.INTEGER) {
		fi := float64(int64(a.num))
		f := math.Float64frombits(b.num)
		if fi != f {
			return cmp.Compare(fi, f)
		}
		return -1 // f == 2^63 > MaxInt64
	}
	fi := float64(int64(b.num))
	f := math.Float64frombits(a.num)
	if fi != f {
		return cmp.Compare(f, fi)
	}
	return +1
}

func (s *topSummary) entries() ([]topEntry, float64) {
	n := s.len()
	if n == 0 {
		return nil, s.offset
	}
	out := make([]topEntry, 0, n)
	for k, v := range s.ints {
		out = append(out, topEntry{num: uint64(k), weight: v, typ: uint8(sqlite3.INTEGER)})
	}
	for k, v := range s.reals {
		out = append(out, topEntry{num: math.Float64bits(k), weight: v, typ: uint8(sqlite3.FLOAT)})
	}
	for k, v := range s.texts {
		out = append(out, topEntry{str: k, weight: v, typ: uint8(sqlite3.TEXT)})
	}
	offset := s.offset
	if len(out) > int(s.c) {
		w := slices.Grow(s.scratch[:0], len(out))
		for i := range out {
			w = append(w, out[i].weight)
		}
		delta := quick.Select(w, len(w)-int(s.c)-1)
		s.scratch = w[:0]
		offset += delta
		kept := 0
		for _, e := range out {
			if e.weight > delta {
				e.weight -= delta
				out[kept] = e
				kept++
			}
		}
		out = out[:kept]
	}
	slices.SortFunc(out, cmpEntry)
	return out, offset
}

func (s *topSummary) evalResult(ctx sqlite3.Context, k int) {
	entries, offset := s.entries()
	if len(entries) == 0 && offset == 0 {
		return
	}
	entries = entries[:min(k, len(entries))]
	var buf [256]byte
	out := append(buf[:0], '[')
	for i, e := range entries {
		if i > 0 {
			out = append(out, ',')
		}
		var err error
		out, err = appendEntryJSON(out, e, offset)
		if err != nil {
			ctx.ResultError(err)
			return
		}
	}
	out = append(out, ']')
	ctx.ResultRawText(out)
	ctx.ResultSubtype('J')
}

func appendEntryJSON(dst []byte, e topEntry, offset float64) ([]byte, error) {
	var err error
	dst = append(dst, '[')
	switch sqlite3.Datatype(e.typ) {
	case sqlite3.INTEGER:
		dst = strconv.AppendInt(dst, int64(e.num), 10)
	case sqlite3.FLOAT:
		dst = appendJSONFloat(dst, math.Float64frombits(e.num))
	default:
		dst, err = util.AppendQuote(dst, e.str)
	}
	dst = append(dst, ',')
	dst = appendJSONFloat(dst, e.weight+offset)
	return append(dst, ']'), err
}

func appendJSONFloat(dst []byte, f float64) []byte {
	if i, ok := floatIsInt64(f); ok {
		return strconv.AppendInt(dst, i, 10)
	}
	return util.AppendNumber(dst, f)
}

const (
	topKeyInt   = 0
	topKeyFloat = 1
	topKeyText  = 2
	topWeightF  = 1 << 2
)

func (b *topSketch) appendBytes(out []byte) []byte {
	switch b.typ {
	case 0:
		if b.k == 0 {
			return append(out, topHeader, 0)
		}
		out = append(out, topHeader)
		out = binary.AppendUvarint(out, uint64(b.k))
		out = binary.AppendUvarint(out, uint64(topCap(b.k)))
		return binary.AppendUvarint(out, 0)
	case topSummaryTag:
		return b.summary().appendBytes(out)
	default:
		out = append(out, topHeader)
		out = binary.AppendUvarint(out, uint64(b.k))
		out = binary.AppendUvarint(out, uint64(topCap(b.k)))
		out = binary.AppendUvarint(out, 1<<1) // count=1, hasOffset=0
		return appendEntry(out, topEntry{
			str:    b.unitStr(),
			num:    b.num,
			weight: b.weight,
			typ:    b.typ,
		})
	}
}

func (s *topSummary) appendBytes(out []byte) []byte {
	if s.k == 0 {
		return append(out, topHeader, 0)
	}
	entries, offset := s.entries()
	out = slices.Grow(out, 6+12*len(entries))
	out = append(out, topHeader)
	out = binary.AppendUvarint(out, uint64(s.k))
	out = binary.AppendUvarint(out, uint64(s.c))
	hdr := uint64(len(entries)) << 1
	if offset > 0 {
		hdr |= 1
	}
	out = binary.AppendUvarint(out, hdr)
	if offset > 0 {
		out = binary.LittleEndian.AppendUint64(out, math.Float64bits(offset))
	}
	for _, e := range entries {
		out = appendEntry(out, e)
	}
	return out
}

func appendEntry(out []byte, e topEntry) []byte {
	var tag byte
	switch sqlite3.Datatype(e.typ) {
	case sqlite3.INTEGER:
		tag = topKeyInt
	case sqlite3.FLOAT:
		tag = topKeyFloat
	default:
		tag = topKeyText
	}
	wi, intWeight := floatIsInt64(e.weight)
	if !intWeight || wi <= 0 {
		tag |= topWeightF
	}
	out = append(out, tag)

	switch sqlite3.Datatype(e.typ) {
	case sqlite3.INTEGER:
		out = binary.AppendVarint(out, int64(e.num))
	case sqlite3.FLOAT:
		out = binary.LittleEndian.AppendUint64(out, e.num)
	default:
		out = binary.AppendUvarint(out, uint64(len(e.str)))
		out = append(out, e.str...)
	}

	if tag&topWeightF == 0 {
		out = binary.AppendUvarint(out, uint64(wi))
	} else {
		out = binary.LittleEndian.AppendUint64(out, math.Float64bits(e.weight))
	}
	return out
}

func topFromBytes(data []byte) (*topSummary, error) {
	if len(data) < 2 {
		return nil, errutil.ErrorString("approx: insufficient bytes to deserialize top-K sketch")
	}
	if data[0] != topHeader {
		return nil, errutil.ErrorString("approx: not a top-K sketch")
	}
	rest := data[1:]

	k, n := binary.Uvarint(rest)
	if n <= 0 {
		return nil, errInvalidTopK
	}
	rest = rest[n:]
	if k == 0 {
		if len(rest) != 0 {
			return nil, errInvalidTopK
		}
		return &topSummary{}, nil
	}
	if k > topMaxK {
		return nil, errInvalidTopK
	}

	c, n := binary.Uvarint(rest)
	if n <= 0 || c < k || c > topMaxC {
		return nil, errInvalidTopK
	}
	rest = rest[n:]

	hdr, n := binary.Uvarint(rest)
	if n <= 0 {
		return nil, errInvalidTopK
	}
	rest = rest[n:]

	count := hdr >> 1
	hasOffset := hdr&1 != 0
	if count > c {
		return nil, errInvalidTopK
	}

	s := &topSummary{k: int16(k), c: int16(c)}
	if hasOffset {
		if len(rest) < 8 {
			return nil, errInvalidTopK
		}
		off := math.Float64frombits(binary.LittleEndian.Uint64(rest))
		if !(off > 0) {
			return nil, errInvalidTopK
		}
		s.offset = off
		rest = rest[8:]
	}
	if uint64(len(rest))/3 < count {
		return nil, errInvalidTopK
	}

	for range count {
		if len(rest) == 0 {
			return nil, errInvalidTopK
		}
		tag := rest[0]
		rest = rest[1:]
		if tag & ^byte(0x07) != 0 {
			return nil, errInvalidTopK
		}

		var (
			iKey int64
			fKey float64
			sKey string
		)
		keyType := tag & 0x03
		switch keyType {
		case topKeyInt:
			v, n := binary.Varint(rest)
			if n <= 0 {
				return nil, errInvalidTopK
			}
			iKey = v
			rest = rest[n:]
		case topKeyFloat:
			if len(rest) < 8 {
				return nil, errInvalidTopK
			}
			f := math.Float64frombits(binary.LittleEndian.Uint64(rest))
			if math.IsNaN(f) {
				return nil, errInvalidTopK
			}
			if _, ok := floatIsInt64(f); ok {
				return nil, errInvalidTopK
			}
			fKey = f
			rest = rest[8:]
		case topKeyText:
			strLen, n := binary.Uvarint(rest)
			if n <= 0 || uint64(len(rest)-n) < strLen {
				return nil, errInvalidTopK
			}
			rest = rest[n:]
			sKey = string(rest[:strLen])
			rest = rest[strLen:]
		default:
			return nil, errInvalidTopK
		}

		var weight float64
		if tag&topWeightF == 0 {
			u, n := binary.Uvarint(rest)
			if n <= 0 || u == 0 {
				return nil, errInvalidTopK
			}
			weight = float64(u)
			rest = rest[n:]
		} else {
			if len(rest) < 8 {
				return nil, errInvalidTopK
			}
			w := math.Float64frombits(binary.LittleEndian.Uint64(rest))
			if !(w > 0) {
				return nil, errInvalidTopK
			}
			weight = w
			rest = rest[8:]
		}

		switch keyType {
		case topKeyInt:
			if s.ints == nil {
				s.ints = make(map[int64]float64)
			} else if _, dup := s.ints[iKey]; dup {
				return nil, errInvalidTopK
			}
			s.ints[iKey] = weight
		case topKeyFloat:
			if s.reals == nil {
				s.reals = make(map[float64]float64)
			} else if _, dup := s.reals[fKey]; dup {
				return nil, errInvalidTopK
			}
			s.reals[fKey] = weight
		case topKeyText:
			if s.texts == nil {
				s.texts = make(map[string]float64)
			} else if _, dup := s.texts[sKey]; dup {
				return nil, errInvalidTopK
			}
			s.texts[sKey] = weight
		}
	}

	if len(rest) != 0 {
		return nil, errInvalidTopK
	}
	return s, nil
}
