package stats

import "github.com/ncruces/go-sqlite3"

const (
	every = iota
	some
)

func newBoolean(kind int) sqlite3.AggregateConstructor {
	return func() sqlite3.AggregateFunction { return &boolean{kind: kind} }
}

type boolean struct {
	count int
	total int
	kind  int
}

func (b *boolean) Value(ctx sqlite3.Context) {
	if b.kind == every {
		ctx.ResultBool(b.count == b.total)
	} else {
		ctx.ResultBool(b.count > 0)
	}
}

func (b *boolean) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	a := arg[0]
	if a.Bool() {
		b.count++
	}
	if a.Type() != sqlite3.NULL {
		b.total++
	}
}

func (b *boolean) Inverse(ctx sqlite3.Context, arg ...sqlite3.Value) {
	a := arg[0]
	if a.Bool() {
		b.count--
	}
	if a.Type() != sqlite3.NULL {
		b.total--
	}
}
