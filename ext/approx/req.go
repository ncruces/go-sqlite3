package approx

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"slices"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/errutil"
	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/sort/quick"
)

const (
	reqVersion = 1
	reqInitK   = 40
)

var (
	unitREQ     req
	emptyREQBin = []byte{reqVersion, reqInitK}
)

type reqSketch struct {
	sketch *req
	args   *[]float32
	arg    float32
	val    float32
}

func (b *reqSketch) Value(ctx sqlite3.Context) {
	b.value(ctx, true)
}

func (b *reqSketch) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if b.sketch == nil {
		b.arg, b.args = parseArg1(ctx, arg[1])
	}
	if arg[0].NumericType() <= sqlite3.FLOAT {
		b.insert(float32(arg[0].Float()))
	}
}

func (b *reqSketch) insert(v float32) {
	switch b.sketch {
	case nil:
		b.sketch = &unitREQ
		b.val = v
	case &unitREQ:
		b.sketch = &req{levels: []reqCompactor{{
			buf: []float32{b.val, v},
		}}}
		b.val = 0
	default:
		b.sketch.add(v)
	}
}

func (b *reqSketch) Combine(other reqSketch) {
	if other.sketch == nil {
		return
	}
	b.arg = other.arg
	b.args = other.args
	if other.sketch == &unitREQ {
		b.insert(other.val)
		return
	}
	switch b.sketch {
	case nil:
		b.sketch = other.sketch.clone()
	case &unitREQ:
		r := other.sketch.clone()
		r.add(b.val)
		b.sketch = r
		b.val = 0
	default:
		b.sketch.merge(other.sketch)
	}
}

func (b *reqSketch) value(ctx sqlite3.Context, isQuantile bool) {
	if b.sketch != nil {
		b.evalResult(ctx, isQuantile)
	}
}

func (b *reqSketch) evalResult(ctx sqlite3.Context, isQuantile bool) {
	if isQuantile {
		if b.args == nil && !(0 <= b.arg && b.arg <= 1) {
			ctx.ResultError(fmt.Errorf("approx: invalid quantile: %f", b.arg))
			return
		}
		if b.args != nil {
			for _, p := range *b.args {
				if !(0 <= p && p <= 1) {
					ctx.ResultError(fmt.Errorf("approx: invalid quantile: %f", p))
					return
				}
			}
		}
	} else if b.arg != b.arg {
		ctx.ResultError(errutil.ErrorString("approx: invalid rank"))
		return
	}
	if b.sketch != &unitREQ {
		if len(b.sketch.levels) == 0 {
			return
		}
		if isQuantile {
			b.sketch.sort()
		}
	}
	eval := b.evalQuantile
	if !isQuantile {
		eval = b.evalRank
	}
	if b.args == nil {
		ctx.ResultFloat(eval(b.arg))
		return
	}
	many := make([]float64, len(*b.args))
	for i, a := range *b.args {
		many[i] = eval(a)
	}
	ctx.ResultJSON(many)
}

func (b *reqSketch) evalQuantile(p float32) float64 {
	if b.sketch == &unitREQ {
		return float64(b.val)
	}
	return b.sketch.quantile(p)
}

func (b *reqSketch) evalRank(v float32) float64 {
	if b.sketch == &unitREQ {
		if v <= b.val {
			return 0
		}
		return 1
	}
	return b.sketch.percentRank(v)
}

type reqRank struct{ reqSketch }

func (b *reqRank) Value(ctx sqlite3.Context) {
	b.value(ctx, false)
}

func (b *reqRank) Combine(other reqRank) {
	b.reqSketch.Combine(other.reqSketch)
}

type reqAccumulate struct{ reqSketch }

func (b *reqAccumulate) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if arg[0].NumericType() <= sqlite3.FLOAT {
		b.insert(float32(arg[0].Float()))
	}
}

func (b *reqAccumulate) Value(ctx sqlite3.Context) {
	switch b.sketch {
	case nil:
		ctx.ResultBlob(emptyREQBin)
	case &unitREQ:
		var buf [8]byte
		buf[0] = reqVersion
		buf[1] = reqInitK
		buf[2] = 0 // level 0 state
		buf[3] = 1 // level 0 count
		binary.LittleEndian.PutUint32(buf[4:], math.Float32bits(b.val))
		ctx.ResultBlob(buf[:])
	default:
		ctx.ResultBlob(b.sketch.toBytes())
	}
}

func (b *reqAccumulate) Combine(other reqAccumulate) {
	b.reqSketch.Combine(other.reqSketch)
}

type reqCombine struct {
	sketch *req
}

func (b *reqCombine) Value(ctx sqlite3.Context) {
	if b.sketch != nil {
		ctx.ResultBlob(b.sketch.toBytes())
	}
}

func (b *reqCombine) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	switch t := arg[0].Type(); t {
	case sqlite3.NULL:
		return
	case sqlite3.BLOB:
		other, err := reqFromBytes(arg[0].RawBlob())
		if err != nil {
			ctx.ResultError(fmt.Errorf("approx: %w", err))
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

func (b *reqCombine) Combine(other reqCombine) {
	if other.sketch == nil {
		return
	}
	if b.sketch == nil {
		b.sketch = other.sketch.clone()
		return
	}
	b.sketch.merge(other.sketch)
}

func reqPercentileEstimate(ctx sqlite3.Context, arg ...sqlite3.Value) {
	reqEstimate(ctx, arg, true)
}

func reqRankEstimate(ctx sqlite3.Context, arg ...sqlite3.Value) {
	reqEstimate(ctx, arg, false)
}

func reqEstimate(ctx sqlite3.Context, arg []sqlite3.Value, isQuantile bool) {
	switch t := arg[0].Type(); t {
	case sqlite3.NULL:
		return
	case sqlite3.BLOB:
		sketch, err := reqFromBytes(arg[0].RawBlob())
		if err != nil {
			ctx.ResultError(fmt.Errorf("approx: %w", err))
			return
		}
		var b reqSketch
		b.sketch = sketch
		b.arg, b.args = parseArg1(ctx, arg[1])
		b.evalResult(ctx, isQuantile)
	default:
		ctx.ResultError(fmt.Errorf("approx: invalid type: %v", t))
	}
}

func parseArg1(ctx sqlite3.Context, val sqlite3.Value) (float32, *[]float32) {
	if val.NumericType() <= sqlite3.FLOAT {
		return float32(val.Float()), nil
	}
	if a, ok := ctx.GetAuxData(1).(*[]float32); ok {
		return 0, a
	}
	var a []float32
	if json.Unmarshal(val.RawText(), &a) == nil && a != nil {
		ctx.SetAuxData(1, &a)
		return 0, &a
	}
	return float32(math.NaN()), nil
}

// req implements the Relative Error Quantiles (REQ) sketch
// in High-Rank Accuracy (HRA) mode (Cormode et al., PODS 2021).
type req struct {
	levels []reqCompactor
}

type reqCompactor struct {
	buf   []float32
	state uint64
}

// reqParams returns the number of sections s and section size k
// for a compactor with the given compaction state.
// Starts at s=3, k=40, doubling s and dividing k by sqrt(2) (to nearest even >= 4)
// whenever state >= 2^(s-1).
func reqParams(state uint64) (s, k int) {
	switch {
	case state < 1<<2:
		return 3, 40
	case state < 1<<5:
		return 6, 28
	case state < 1<<11:
		return 12, 20
	case state < 1<<23:
		return 24, 14
	case state < 1<<47:
		return 48, 10
	default:
		return 96, 8
	}
}

func compactOffset(level int, state uint64) int {
	// SplitMix64 on (level, state >> 1) XORed with (state & 1)
	// for deterministic antithetic (paired) compaction flips.
	x := uint64(level)*0x9e3779b97f4a7c15 + (state >> 1)
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	x = x ^ (x >> 31)
	return int((x ^ state) & 1)
}

func (r *req) add(v float32) {
	c := &r.levels[0]
	c.buf = append(c.buf, v)
	if s, k := reqParams(c.state); len(c.buf) >= 2*s*k {
		r.compact(0)
	}
}

func (r *req) compact(h int) {
	for ; h < len(r.levels); h++ {
		c := &r.levels[h]
		s, k := reqParams(c.state)
		if len(c.buf) < 2*s*k {
			continue
		}
		secs := min(bits.TrailingZeros64(^c.state)+1, s)
		nonCompact := (2*s - secs) * k
		compactLen := (len(c.buf) - nonCompact) &^ 1
		offset := compactOffset(h, c.state)
		c.state++

		quick.SortFirst(c.buf, compactLen)

		if h+1 >= len(r.levels) {
			r.levels = append(r.levels, reqCompactor{
				buf: make([]float32, 0, compactLen/2),
			})
			c = &r.levels[h]
		}
		next := &r.levels[h+1]
		for i := offset; i < compactLen; i += 2 {
			next.buf = append(next.buf, c.buf[i])
		}
		n := copy(c.buf, c.buf[compactLen:])
		c.buf = c.buf[:n]
	}
}

func (r *req) clone() *req {
	out := &req{
		levels: make([]reqCompactor, len(r.levels)),
	}
	for i, c := range r.levels {
		out.levels[i] = reqCompactor{
			buf:   slices.Clone(c.buf),
			state: c.state,
		}
	}
	return out
}

func (r *req) merge(other *req) {
	if len(other.levels) > len(r.levels) {
		r.levels = slices.Grow(r.levels, len(other.levels)-len(r.levels))
		r.levels = r.levels[:len(other.levels)]
	}
	for h := range other.levels {
		src := &other.levels[h]
		dst := &r.levels[h]
		dst.buf = append(dst.buf, src.buf...)
		dst.state |= src.state
	}
	r.compact(0)
}

func (r *req) sort() {
	for h := range r.levels {
		slices.Sort(r.levels[h].buf)
	}
}

func (r *req) count() uint64 {
	var total uint64
	for h := range r.levels {
		total += uint64(len(r.levels[h].buf)) << h
	}
	return total
}

func (r *req) quantile(p float32) float64 {
	if len(r.levels) == 1 {
		buf := r.levels[0].buf
		i, f := math.Modf(float64(p) * float64(len(buf)-1))
		if f == 0 {
			return float64(buf[int(i)])
		}
		return util.Lerp(float64(buf[int(i)]), float64(buf[int(i)+1]), f)
	}

	total := r.count()
	pos := float64(p) * float64(total-1)
	target := uint64(pos)

	var (
		lo, hi     [64]int
		iLess      [64]int
		iLessEq    [64]int
		baseWeight uint64
	)
	for h := range r.levels {
		hi[h] = len(r.levels[h].buf)
	}

	for {
		bestH := -1
		var maxW uint64
		for h := range r.levels {
			if w := uint64(hi[h]-lo[h]) << h; w > maxW {
				maxW = w
				bestH = h
			}
		}
		pivot := r.levels[bestH].buf[(lo[bestH]+hi[bestH])/2]

		wLess := baseWeight
		var wEq, w0 uint64
		for h := range r.levels {
			if lo[h] >= hi[h] {
				iLess[h] = lo[h]
				iLessEq[h] = lo[h]
				continue
			}
			buf := r.levels[h].buf
			i, found := slices.BinarySearch(buf[lo[h]:hi[h]], pivot)
			il := lo[h] + i
			ile := il
			if found {
				for ile < hi[h] && buf[ile] == pivot {
					ile++
				}
				if w0 == 0 {
					w0 = uint64(1) << h
				}
			}
			iLess[h] = il
			iLessEq[h] = ile
			wLess += uint64(il-lo[h]) << h
			wEq += uint64(ile-il) << h
		}
		wLessEq := wLess + wEq

		if target < wLess {
			hi = iLess
		} else if target >= wLessEq {
			lo = iLessEq
			baseWeight = wLessEq
		} else {
			m0 := float64(pivot)
			if pos <= float64(wLessEq-w0) || wLessEq >= total {
				return m0
			}
			nextVal := float32(math.Inf(+1))
			for h := range r.levels {
				buf := r.levels[h].buf
				if iLessEq[h] < len(buf) && buf[iLessEq[h]] < nextVal {
					nextVal = buf[iLessEq[h]]
				}
			}
			m1 := float64(nextVal)
			if m0 == m1 {
				return m0
			}
			f := (pos - float64(wLessEq-w0)) / float64(w0)
			return util.Lerp(m0, m1, f)
		}
	}
}

func (r *req) percentRank(v float32) float64 {
	total := r.count()
	var (
		wLess uint64
		wLo   uint64
		xLo   = float32(math.Inf(-1))
		xHi   = float32(math.Inf(+1))
	)
	for h := range r.levels {
		w := uint64(1) << h
		for _, x := range r.levels[h].buf {
			if x < v {
				wLess += w
				if x > xLo {
					xLo = x
					wLo = w
				}
			} else if x < xHi {
				xHi = x
			}
		}
	}
	if wLess == 0 {
		return 0
	}
	if wLess == total {
		return 1
	}
	if v == xHi || xLo == xHi {
		return float64(wLess) / float64(total-1)
	}
	f := float64(v-xLo) / float64(xHi-xLo)
	return (float64(wLess-wLo) + f*float64(wLo)) / float64(total-1)
}

func (r *req) toBytes() []byte {
	r.sort()
	size := 2
	for h := range r.levels {
		c := &r.levels[h]
		size += 2 + 4*len(c.buf)
	}
	out := make([]byte, 0, size)
	out = append(out, reqVersion, reqInitK)
	for h := range r.levels {
		c := &r.levels[h]
		out = binary.AppendUvarint(out, c.state)
		out = binary.AppendUvarint(out, uint64(len(c.buf)))
		for _, v := range c.buf {
			out = binary.LittleEndian.AppendUint32(out, math.Float32bits(v))
		}
	}
	return out
}

func reqFromBytes(data []byte) (*req, error) {
	if len(data) < 2 {
		return nil, errors.New("insufficient bytes to deserialize REQ sketch")
	}
	if data[0] != reqVersion {
		return nil, fmt.Errorf("unsupported REQ version: %d", data[0])
	}
	if data[1] != reqInitK {
		return nil, fmt.Errorf("incompatible REQ k: %d", data[1])
	}
	rest := data[2:]
	var r req
	for len(rest) > 0 {
		if len(r.levels) >= 60 {
			return nil, errors.New("invalid REQ sketch")
		}
		state, n := binary.Uvarint(rest)
		if n <= 0 {
			return nil, errors.New("invalid REQ sketch")
		}
		rest = rest[n:]

		count, n := binary.Uvarint(rest)
		if n <= 0 {
			return nil, errors.New("invalid REQ sketch")
		}
		rest = rest[n:]

		s, k := reqParams(state)
		if count == 0 || count > uint64(2*s*k) || uint64(len(rest))/4 < count {
			return nil, errors.New("invalid REQ sketch")
		}
		buf := make([]float32, count)
		for i := range buf {
			v := math.Float32frombits(binary.LittleEndian.Uint32(rest[4*i:]))
			if math.IsNaN(float64(v)) {
				return nil, errors.New("invalid REQ sketch")
			}
			buf[i] = v
		}
		rest = rest[4*count:]
		r.levels = append(r.levels, reqCompactor{
			buf:   buf,
			state: state,
		})
	}
	return &r, nil
}
