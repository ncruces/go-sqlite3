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

type topTag uint8

const (
	topHeader = 1<<4 | 6 // HLL version 1, with an unassigned storage type
	topMaxK   = 1000
	topMinC   = 128
	topMaxC   = 4 * topMaxK

	topIntTag     topTag = 1
	topRealTag    topTag = 2
	topTextTag    topTag = 3
	topSummaryTag topTag = 0xff

	errInvalidTopK = errutil.ErrorString("approx: invalid top-K sketch")
)

func topCap(k int16) int16 {
	return max(topMinC, 4*k)
}

type topSketch struct {
	ptr    unsafe.Pointer // nil, unsafe.StringData(str), or *topSummary
	bits   uint64         // int64, float64, or len(str)
	weight float64        // inline weight in unit state (> 0)
	k      int16          // target K, or 0 if Step never ran
	typ    topTag
}

func (b *topSketch) summary() *topSummary {
	return (*topSummary)(b.ptr)
}

func (b *topSketch) string() string {
	if b.typ != topTextTag {
		return ""
	}
	return unsafe.String((*byte)(b.ptr), int(b.bits))
}

func (b *topSketch) Value(ctx sqlite3.Context) {
	switch b.typ {
	case 0:
		return
	case topSummaryTag:
		s := b.summary()
		entries, offset := s.entries()
		evalEntries(ctx, entries, offset, int(s.k))
	default:
		evalEntries(ctx, []topEntry{{
			str:    b.string(),
			num:    b.bits,
			weight: b.weight,
			typ:    b.typ,
		}}, 0, 1)
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
		b.insert(topIntTag, uint64(arg[0].Int64()), "", w)
	case sqlite3.FLOAT:
		f := arg[0].Float()
		if i, ok := floatIsInt(f); ok {
			b.insert(topIntTag, uint64(i), "", w)
		} else {
			b.insert(topRealTag, math.Float64bits(f), "", w)
		}
	case sqlite3.TEXT:
		raw := arg[0].RawText()
		if b.typ == topTextTag && string(raw) == b.string() {
			b.weight += w
		} else {
			b.insert(topTextTag, uint64(len(raw)), string(raw), w)
		}
	}
}

func (b *topSketch) insert(typ topTag, num uint64, str string, w float64) {
	switch b.typ {
	case 0:
		b.typ = typ
		b.bits = num
		if typ == topTextTag {
			b.ptr = unsafe.Pointer(unsafe.StringData(str))
		}
		b.weight = w
	case topSummaryTag:
		b.summary().add(typ, num, str, w)
	default:
		if b.typ == typ && (typ == topTextTag && str == b.string() || typ != topTextTag && num == b.bits) {
			b.weight += w
			return
		}
		s := &topSummary{k: b.k, c: topCap(b.k)}
		s.add(b.typ, b.bits, b.string(), b.weight)
		s.add(typ, num, str, w)
		*b = topSketch{ptr: unsafe.Pointer(s), typ: topSummaryTag}
	}
}

func (b *topSketch) Combine(other topSketch) {
	if b.k == 0 && b.typ != topSummaryTag {
		b.k = other.k
	}
	switch other.typ {
	case 0:
		return
	case topSummaryTag:
		os := other.summary()
		switch b.typ {
		case 0:
			*b = topSketch{ptr: unsafe.Pointer(os.clone()), typ: topSummaryTag}
		case topSummaryTag:
			b.summary().merge(os)
		default:
			s := os.clone()
			s.add(b.typ, b.bits, b.string(), b.weight)
			*b = topSketch{ptr: unsafe.Pointer(s), typ: topSummaryTag}
		}
	default:
		b.insert(other.typ, other.bits, other.string(), other.weight)
	}
}

type topAccumulate struct{ topSketch }

func (b *topAccumulate) Value(ctx sqlite3.Context) {
	var buf [32]byte // fits empty and single-row numeric sketches
	switch b.typ {
	case 0:
		ctx.ResultBlob(appendSketchBytes(buf[:0], nil, 0, b.k, topCap(b.k)))
	case topSummaryTag:
		ctx.ResultBlob(b.summary().appendBytes(buf[:0]))
	default:
		ctx.ResultBlob(appendSketchBytes(buf[:0], []topEntry{{
			str:    b.string(),
			num:    b.bits,
			weight: b.weight,
			typ:    b.typ,
		}}, 0, b.k, topCap(b.k)))
	}
}

func (b *topAccumulate) Combine(other topAccumulate) {
	b.topSketch.Combine(other.topSketch)
}

type topCombine struct {
	sketch *topSummary
}

func (b *topCombine) Value(ctx sqlite3.Context) {
	if b.sketch != nil {
		var buf [32]byte
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
		entries, offset := sketch.entries()
		evalEntries(ctx, entries, offset, k)
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

func (s *topSummary) add(typ topTag, num uint64, str string, w float64) {
	switch typ {
	case topIntTag:
		addMap(s, &s.ints, int64(num), w)
	case topRealTag:
		addMap(s, &s.reals, math.Float64frombits(num), w)
	default:
		addMap(s, &s.texts, str, w)
	}
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
	s.scratch = w[:0]

	delta := quick.Select(w, len(w)-int(s.c)-1)
	purgeMap(s.ints, delta)
	purgeMap(s.reals, delta)
	purgeMap(s.texts, delta)
	s.offset += delta
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

func purgeMap[K comparable](m map[K]float64, delta float64) {
	for k, v := range m {
		if v <= delta {
			delete(m, k)
		} else {
			m[k] = v - delta
		}
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
	typ    topTag
}

func cmpEntry(a, b topEntry) int {
	if c := cmp.Compare(b.weight, a.weight); c != 0 {
		return c
	}
	if a.typ == b.typ {
		switch a.typ {
		case topIntTag:
			return cmp.Compare(int64(a.num), int64(b.num))
		case topRealTag:
			return cmp.Compare(math.Float64frombits(a.num), math.Float64frombits(b.num))
		default:
			return cmp.Compare(a.str, b.str)
		}
	}
	if a.typ == topTextTag || b.typ == topTextTag {
		return cmp.Compare(a.typ, b.typ)
	}
	if a.typ == topIntTag {
		return cmpIntFloat(int64(a.num), math.Float64frombits(b.num))
	}
	return -cmpIntFloat(int64(b.num), math.Float64frombits(a.num))
}

func (s *topSummary) entries() ([]topEntry, float64) {
	n := s.len()
	if n == 0 {
		return nil, s.offset
	}
	out := make([]topEntry, 0, n)
	for k, v := range s.ints {
		out = append(out, topEntry{num: uint64(k), weight: v, typ: topIntTag})
	}
	for k, v := range s.reals {
		out = append(out, topEntry{num: math.Float64bits(k), weight: v, typ: topRealTag})
	}
	for k, v := range s.texts {
		out = append(out, topEntry{str: k, weight: v, typ: topTextTag})
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

func evalEntries(ctx sqlite3.Context, entries []topEntry, offset float64, k int) {
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
		out = append(out, '[')
		switch e.typ {
		case topIntTag:
			out = strconv.AppendInt(out, int64(e.num), 10)
		case topRealTag:
			out = appendJSONFloat(out, math.Float64frombits(e.num))
		default:
			out, err = util.AppendQuote(out, e.str)
			if err != nil {
				ctx.ResultError(err)
				return
			}
		}
		out = append(out, ',')
		out = appendJSONFloat(out, e.weight+offset)
		out = append(out, ']')
	}
	out = append(out, ']')
	ctx.ResultRawText(out)
	ctx.ResultSubtype('J')
}

func appendJSONFloat(dst []byte, f float64) []byte {
	if i, ok := floatIsInt(f); ok {
		return strconv.AppendInt(dst, i, 10)
	}
	return util.AppendNumber(dst, f)
}

func (s *topSummary) appendBytes(out []byte) []byte {
	entries, offset := s.entries()
	return appendSketchBytes(out, entries, offset, s.k, s.c)
}

func appendSketchBytes(out []byte, entries []topEntry, offset float64, k, c int16) []byte {
	if k == 0 {
		return append(out, topHeader)
	}
	out = slices.Grow(out, 15+17*len(entries))
	out = append(out, topHeader)
	out = binary.LittleEndian.AppendUint16(out, uint16(k))
	out = binary.LittleEndian.AppendUint16(out, uint16(c))
	out = binary.LittleEndian.AppendUint16(out, uint16(len(entries)))
	out = binary.LittleEndian.AppendUint64(out, math.Float64bits(offset))
	for _, e := range entries {
		out = append(out, byte(e.typ))
		out = binary.LittleEndian.AppendUint64(out, math.Float64bits(e.weight))
		if e.typ == topTextTag {
			out = binary.AppendUvarint(out, uint64(len(e.str)))
			out = append(out, e.str...)
		} else {
			out = binary.LittleEndian.AppendUint64(out, e.num)
		}
	}
	return out
}

func topFromBytes(data []byte) (*topSummary, error) {
	if len(data) == 0 {
		return nil, errutil.ErrorString("approx: insufficient bytes to deserialize top-K sketch")
	}
	if data[0] != topHeader {
		return nil, errutil.ErrorString("approx: not a top-K sketch")
	}
	if len(data) == 1 {
		return &topSummary{}, nil
	}
	if len(data) < 15 {
		return nil, errInvalidTopK
	}

	k := binary.LittleEndian.Uint16(data[1:])
	c := binary.LittleEndian.Uint16(data[3:])
	count := binary.LittleEndian.Uint16(data[5:])
	off := math.Float64frombits(binary.LittleEndian.Uint64(data[7:]))
	if k == 0 || k > topMaxK || c < k || c > topMaxC || count > c || !(off >= 0) {
		return nil, errInvalidTopK
	}

	rest := data[15:]
	if len(rest)/10 < int(count) {
		return nil, errInvalidTopK
	}

	s := &topSummary{k: int16(k), c: int16(c)}
	if off > 0 {
		s.offset = off
	}
	for range count {
		if len(rest) < 10 {
			return nil, errInvalidTopK
		}
		typ := topTag(rest[0])
		w := math.Float64frombits(binary.LittleEndian.Uint64(rest[1:]))
		if !(w > 0) {
			return nil, errInvalidTopK
		}
		rest = rest[9:]

		var ok bool
		switch typ {
		case topIntTag:
			if len(rest) < 8 {
				return nil, errInvalidTopK
			}
			ok = putUnique(&s.ints, int64(binary.LittleEndian.Uint64(rest)), w)
			rest = rest[8:]
		case topRealTag:
			if len(rest) < 8 {
				return nil, errInvalidTopK
			}
			f := math.Float64frombits(binary.LittleEndian.Uint64(rest))
			if math.IsNaN(f) {
				return nil, errInvalidTopK
			}
			if _, isInt := floatIsInt(f); isInt {
				return nil, errInvalidTopK
			}
			ok = putUnique(&s.reals, f, w)
			rest = rest[8:]
		case topTextTag:
			strLen, n := binary.Uvarint(rest)
			if n <= 0 || uint64(len(rest)-n) < strLen {
				return nil, errInvalidTopK
			}
			rest = rest[n:]
			ok = putUnique(&s.texts, string(rest[:strLen]), w)
			rest = rest[strLen:]
		default:
			return nil, errInvalidTopK
		}
		if !ok {
			return nil, errInvalidTopK
		}
	}
	if len(rest) != 0 {
		return nil, errInvalidTopK
	}
	return s, nil
}

func putUnique[K comparable](m *map[K]float64, k K, w float64) bool {
	if *m == nil {
		*m = make(map[K]float64)
	} else if _, dup := (*m)[k]; dup {
		return false
	}
	(*m)[k] = w
	return true
}
