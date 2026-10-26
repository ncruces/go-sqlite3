package approx

import (
	"hash/maphash"
	"math"

	"github.com/axiomhq/hyperloglog"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/util/sql3util"
)

var (
	seed     = maphash.MakeSeed()
	blobSeed = maphash.MakeSeed()
)

func newHLL() sqlite3.AggregateConstructor {
	return sql3util.WindowFromCombiner[hllSketch](1e4)
}

type hllSketch struct {
	hash   uint64
	sketch *hyperloglog.Sketch
}

func (b *hllSketch) Value(ctx sqlite3.Context) {
	switch {
	case b.sketch != nil:
		ctx.ResultInt64(int64(b.sketch.Estimate()))
	case b.hash != 0:
		ctx.ResultInt64(1)
	default:
		ctx.ResultInt64(0)
	}
}

func (b *hllSketch) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	var hash uint64
	switch arg[0].Type() {
	case sqlite3.NULL:
		return
	case sqlite3.INTEGER:
		hash = maphash.Comparable(seed, arg[0].Int64())
	case sqlite3.FLOAT:
		f := arg[0].Float()
		if i := int64(f); float64(i) == f && i != math.MaxInt64 {
			hash = maphash.Comparable(seed, i)
		} else {
			hash = maphash.Comparable(seed, f)
		}
	case sqlite3.TEXT:
		hash = maphash.Bytes(seed, arg[0].RawText())
	case sqlite3.BLOB:
		hash = maphash.Bytes(blobSeed, arg[0].RawBlob())
	}
	b.insert(max(1, hash))
}

func (b *hllSketch) Combine(other hllSketch) {
	if other.hash != 0 {
		b.insert(other.hash)
		return
	}
	if other.sketch != nil {
		if b.sketch != nil {
			b.sketch.Merge(other.sketch)
			return
		}
		b.sketch = other.sketch.Clone()
		if b.hash != 0 {
			b.sketch.InsertHash(b.hash)
			b.hash = 0
		}
	}
}

func (b *hllSketch) insert(hash uint64) {
	switch {
	case b.sketch != nil:
		b.sketch.InsertHash(hash)
	case b.hash == 0 || b.hash == hash:
		b.hash = hash
	default:
		b.sketch = hyperloglog.New()
		b.sketch.InsertHash(b.hash)
		b.sketch.InsertHash(hash)
		b.hash = 0
	}
}
