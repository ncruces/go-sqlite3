package stats

import (
	"unsafe"

	"github.com/ncruces/go-sqlite3"
)

func newMode() sqlite3.AggregateFunction {
	return &mode{}
}

type mode struct {
	ints  counter[int64]
	reals counter[float64]
	texts counter[string]
	blobs counter[string]
}

func (m mode) Value(ctx sqlite3.Context) {
	var (
		typ = sqlite3.NULL
		max uint
		i64 int64
		f64 float64
		str string
	)
	for k, v := range m.ints {
		if v > max || v == max && k < i64 {
			typ = sqlite3.INTEGER
			max = v
			i64 = k
		}
	}
	for k, v := range m.reals {
		if v > max || v == max && k < f64 {
			typ = sqlite3.FLOAT
			max = v
			f64 = k
		}
	}
	for k, v := range m.texts {
		if v > max || v == max && typ == sqlite3.TEXT && k < str {
			typ = sqlite3.TEXT
			max = v
			str = k
		}
	}
	for k, v := range m.blobs {
		if v > max || v == max && typ == sqlite3.BLOB && k < str {
			typ = sqlite3.BLOB
			max = v
			str = k
		}
	}
	switch typ {
	case sqlite3.INTEGER:
		ctx.ResultInt64(i64)
	case sqlite3.FLOAT:
		ctx.ResultFloat(f64)
	case sqlite3.TEXT:
		ctx.ResultText(str)
	case sqlite3.BLOB:
		ctx.ResultBlob(unsafe.Slice(unsafe.StringData(str), len(str)))
	}
}

func (m *mode) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	switch arg[0].Type() {
	case sqlite3.INTEGER:
		if m.reals == nil {
			m.ints.add(arg[0].Int64())
			break
		}
		fallthrough
	case sqlite3.FLOAT:
		m.reals.add(arg[0].Float())
		for k, v := range m.ints {
			m.reals[float64(k)] += v
		}
		m.ints = nil
	case sqlite3.TEXT:
		m.texts.add(arg[0].Text())
	case sqlite3.BLOB:
		m.blobs.add(string(arg[0].RawBlob()))
	}
}

func (m *mode) Inverse(ctx sqlite3.Context, arg ...sqlite3.Value) {
	switch arg[0].Type() {
	case sqlite3.INTEGER:
		if m.reals == nil {
			m.ints.del(arg[0].Int64())
			break
		}
		fallthrough
	case sqlite3.FLOAT:
		m.reals.del(arg[0].Float())
	case sqlite3.TEXT:
		m.texts.del(arg[0].Text())
	case sqlite3.BLOB:
		m.blobs.del(string(arg[0].RawBlob()))
	}
}

type counter[T comparable] map[T]uint

func (c *counter[T]) add(k T) {
	if (*c) == nil {
		(*c) = make(counter[T])
	}
	(*c)[k]++
}

func (c counter[T]) del(k T) {
	if n := c[k]; n == 1 {
		delete(c, k)
	} else {
		c[k] = n - 1
	}
}
