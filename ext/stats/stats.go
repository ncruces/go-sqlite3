// Package stats provides aggregate functions for statistics.
//
// Functions:
//   - var_samp: sample variance
//   - var_pop: population variance
//   - stddev_samp: sample standard deviation
//   - stddev_pop: population standard deviation
//
// See: [ANSI SQL Aggregate Functions]
//
// [ANSI SQL Aggregate Functions]: https://www.oreilly.com/library/view/sql-in-a/9780596155322/ch04s02.html
package stats

import "github.com/ncruces/go-sqlite3"

// Register registers statistics functions.
func Register(db *sqlite3.Conn) {
	flags := sqlite3.DETERMINISTIC | sqlite3.INNOCUOUS
	db.CreateWindowFunction("var_pop", 1, flags, create(var_pop))
	db.CreateWindowFunction("var_samp", 1, flags, create(var_samp))
	db.CreateWindowFunction("stddev_pop", 1, flags, create(stddev_pop))
	db.CreateWindowFunction("stddev_samp", 1, flags, create(stddev_samp))
}

const (
	var_pop = iota
	var_samp
	stddev_pop
	stddev_samp
)

func create(kind int) func() sqlite3.AggregateFunction {
	return func() sqlite3.AggregateFunction { return &state{kind: kind} }
}

type state struct {
	kind int
	welford
}

func (f *state) Value(ctx sqlite3.Context) {
	var r float64
	switch f.kind {
	case var_pop:
		r = f.var_pop()
	case var_samp:
		r = f.var_samp()
	case stddev_pop:
		r = f.stddev_pop()
	case stddev_samp:
		r = f.stddev_samp()
	}
	ctx.ResultFloat(r)
}

func (f *state) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if a := arg[0]; a.Type() != sqlite3.NULL {
		f.enqueue(a.Float())
	}
}

func (f *state) Inverse(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if a := arg[0]; a.Type() != sqlite3.NULL {
		f.dequeue(a.Float())
	}
}
