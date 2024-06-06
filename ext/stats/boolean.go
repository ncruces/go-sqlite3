package stats

import "github.com/ncruces/go-sqlite3"

const (
	every = iota
	some
)

func newBoolean(kind int) func() sqlite3.AggregateFunction {
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
	if arg[0].Type() == sqlite3.NULL {
		return
	}
	if arg[0].Bool() {
		b.count++
	}
	b.total++
}

func (b *boolean) Inverse(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if arg[0].Type() == sqlite3.NULL {
		return
	}
	if arg[0].Bool() {
		b.count--
	}
	b.total--
}
