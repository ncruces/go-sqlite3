package stats

import (
	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/util/sql3util"
)

func newBitXor() sqlite3.AggregateFunction {
	return &bitXor{}
}

type bitXor struct {
	val   int64
	count int
}

func (b *bitXor) Value(ctx sqlite3.Context) {
	if b.count > 0 {
		ctx.ResultInt64(b.val)
	}
}

func (b *bitXor) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if arg[0].NumericType() <= sqlite3.FLOAT {
		b.val ^= arg[0].Int64()
		b.count++
	}
}

func (b *bitXor) Inverse(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if arg[0].NumericType() <= sqlite3.FLOAT {
		b.val ^= arg[0].Int64()
		b.count--
	}
}

func newBitOr() sqlite3.AggregateConstructor {
	return sql3util.WindowFromCombiner[bitOr](1e6)
}

type bitOr struct {
	val int64
	ok  bool
}

func (b *bitOr) Value(ctx sqlite3.Context) {
	if b.ok {
		ctx.ResultInt64(b.val)
	}
}

func (b *bitOr) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if arg[0].NumericType() <= sqlite3.FLOAT {
		b.val |= arg[0].Int64()
		b.ok = true
	}
}

func (b *bitOr) Combine(other bitOr) {
	if other.ok {
		b.val |= other.val
		b.ok = true
	}
}

func newBitAnd() sqlite3.AggregateConstructor {
	return sql3util.WindowFromCombiner[bitAnd](1e6)
}

type bitAnd struct {
	val int64
	ok  bool
}

func (b *bitAnd) Value(ctx sqlite3.Context) {
	if b.ok {
		ctx.ResultInt64(^b.val)
	}
}

func (b *bitAnd) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if arg[0].NumericType() <= sqlite3.FLOAT {
		b.val |= ^arg[0].Int64()
		b.ok = true
	}
}

func (b *bitAnd) Combine(other bitAnd) {
	if other.ok {
		b.val |= other.val
		b.ok = true
	}
}
