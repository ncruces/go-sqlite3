package approx

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/segmentio/go-hll"
	"github.com/twmb/murmur3"

	"github.com/ncruces/go-sqlite3"
)

var emptyHLL, _ = hll.NewHll(hll.Settings{
	Log2m:             11,
	Regwidth:          5,
	ExplicitThreshold: hll.AutoExplicitThreshold,
	SparseEnabled:     true,
})

type hllSketch struct {
	sketch *hll.Hll
	hash   uint64
}

func (b *hllSketch) Value(ctx sqlite3.Context) {
	switch {
	case b.sketch != nil:
		ctx.ResultInt64(int64(b.sketch.Cardinality()))
	case b.hash != 0:
		ctx.ResultInt64(1)
	default:
		ctx.ResultInt64(0)
	}
}

func (b *hllSketch) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	var hash uint64
	for _, a := range arg {
		if hash = hashValue(hash, a); hash == 0 {
			return
		}
	}
	b.insert(hash)
}

func (b *hllSketch) Combine(other hllSketch) {
	if other.hash != 0 {
		b.insert(other.hash)
		return
	}
	if other.sketch != nil {
		if b.sketch != nil {
			b.sketch.Union(*other.sketch)
			return
		}
		h := emptyHLL
		h.Union(*other.sketch)
		if b.hash != 0 {
			h.AddRaw(b.hash)
			b.hash = 0
		}
		b.sketch = &h
	}
}

func (b *hllSketch) insert(hash uint64) {
	switch {
	case hash == 0 || hash == b.hash:
		return
	case b.sketch != nil:
		b.sketch.AddRaw(hash)
	case b.hash == 0:
		b.hash = hash
	default:
		h := emptyHLL
		h.AddRaw(b.hash)
		h.AddRaw(hash)
		b.sketch = &h
		b.hash = 0
	}
}

type hllAccumulate struct{ hllSketch }

func (b *hllAccumulate) Value(ctx sqlite3.Context) {
	switch {
	case b.hash != 0:
		h := emptyHLL
		h.AddRaw(b.hash)
		ctx.ResultBlob(h.ToBytes())
	case b.sketch != nil:
		ctx.ResultBlob(b.sketch.ToBytes())
	default:
		ctx.ResultBlob(emptyHLL.ToBytes())
	}
}

func (b *hllAccumulate) Combine(other hllAccumulate) {
	b.hllSketch.Combine(other.hllSketch)
}

type hllCombine struct {
	sketch *hll.Hll
	err    error
}

func (b *hllCombine) Value(ctx sqlite3.Context) {
	switch {
	case b.err != nil:
		ctx.ResultError(b.err)
	case b.sketch != nil:
		ctx.ResultBlob(b.sketch.ToBytes())
	}
}

func (b *hllCombine) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	switch t := arg[0].Type(); t {
	case sqlite3.NULL:
		return
	case sqlite3.BLOB:
		other, err := hll.FromBytes(arg[0].RawBlob())
		if err != nil {
			ctx.ResultError(fmt.Errorf("approx: %w", err))
			return
		}
		if b.sketch == nil {
			b.sketch = &other
			return
		}
		if err := b.sketch.StrictUnion(other); err != nil {
			ctx.ResultError(fmt.Errorf("approx: %w", err))
		}
	default:
		ctx.ResultError(fmt.Errorf("approx: invalid type: %v", t))
	}
}

func (b *hllCombine) Combine(other hllCombine) {
	if b.err != nil {
		return
	}
	if other.err != nil {
		b.err = other.err
		return
	}
	if other.sketch == nil {
		return
	}
	if b.sketch == nil {
		h := *other.sketch
		h.Clear()
		h.Union(*other.sketch)
		b.sketch = &h
		return
	}
	if err := b.sketch.StrictUnion(*other.sketch); err != nil {
		b.err = fmt.Errorf("approx: %w", err)
	}
}

func hllEstimate(ctx sqlite3.Context, arg ...sqlite3.Value) {
	switch t := arg[0].Type(); t {
	case sqlite3.NULL:
		return
	case sqlite3.BLOB:
		sketch, err := hll.FromBytes(arg[0].RawBlob())
		if err != nil {
			ctx.ResultError(fmt.Errorf("approx: %w", err))
			return
		}
		ctx.ResultInt64(int64(sketch.Cardinality()))
	default:
		ctx.ResultError(fmt.Errorf("approx: invalid type: %v", t))
	}
}

func hashValue(seed uint64, val sqlite3.Value) uint64 {
	var (
		buf  [8]byte
		data []byte
	)
	switch val.Type() {
	default: // sqlite3.NULL
		return 0
	case sqlite3.INTEGER:
		data = binary.LittleEndian.AppendUint64(buf[:0],
			uint64(val.Int64()))
	case sqlite3.FLOAT:
		var u uint64
		f := val.Float()
		if i, ok := floatIsInt64(f); ok {
			u = uint64(i)
		} else {
			seed ^= 2
			u = math.Float64bits(f)
		}
		data = binary.LittleEndian.AppendUint64(buf[:0], u)
	case sqlite3.TEXT:
		data = val.RawText()
	case sqlite3.BLOB:
		seed ^= 1
		data = val.RawBlob()
	}
	return murmur3.SeedSum64(seed, data)
}

func floatIsInt64(f float64) (int64, bool) {
	// math.MaxInt64 is not exactly representable as a float64 (it rounds to 2^63),
	// but int64(2^63) may saturate back to math.MaxInt64.
	i := int64(f)
	return i, float64(i) == f && i != math.MaxInt64
}
