// Package stats provides aggregate functions for statistics.
//
// Functions:
//   - stddev_pop: population standard deviation
//   - stddev_samp: sample standard deviation
//   - var_pop: population variance
//   - var_samp: sample variance
//   - covar_pop: population covariance
//   - covar_samp: sample covariance
//   - corr: correlation coefficient
//
// See: [ANSI SQL Aggregate Functions]
//
// [ANSI SQL Aggregate Functions]: https://www.oreilly.com/library/view/sql-in-a/9780596155322/ch04s02.html
package stats

import "github.com/ncruces/go-sqlite3"

// Register registers statistics functions.
func Register(db *sqlite3.Conn) {
	flags := sqlite3.DETERMINISTIC | sqlite3.INNOCUOUS
	db.CreateWindowFunction("var_pop", 1, flags, newVariance(var_pop))
	db.CreateWindowFunction("var_samp", 1, flags, newVariance(var_samp))
	db.CreateWindowFunction("stddev_pop", 1, flags, newVariance(stddev_pop))
	db.CreateWindowFunction("stddev_samp", 1, flags, newVariance(stddev_samp))
	db.CreateWindowFunction("covar_pop", 2, flags, newCovariance(var_pop))
	db.CreateWindowFunction("covar_samp", 2, flags, newCovariance(var_samp))
	db.CreateWindowFunction("corr", 2, flags, newCovariance(corr))
}

const (
	var_pop = iota
	var_samp
	stddev_pop
	stddev_samp
	corr
)

func newVariance(kind int) func() sqlite3.AggregateFunction {
	return func() sqlite3.AggregateFunction { return &variance{kind: kind} }
}

type variance struct {
	kind int
	welford
}

func (fn *variance) Value(ctx sqlite3.Context) {
	var r float64
	switch fn.kind {
	case var_pop:
		r = fn.var_pop()
	case var_samp:
		r = fn.var_samp()
	case stddev_pop:
		r = fn.stddev_pop()
	case stddev_samp:
		r = fn.stddev_samp()
	}
	ctx.ResultFloat(r)
}

func (fn *variance) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if a := arg[0]; a.Type() != sqlite3.NULL {
		fn.enqueue(a.Float())
	}
}

func (fn *variance) Inverse(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if a := arg[0]; a.Type() != sqlite3.NULL {
		fn.dequeue(a.Float())
	}
}

func newCovariance(kind int) func() sqlite3.AggregateFunction {
	return func() sqlite3.AggregateFunction { return &covariance{kind: kind} }
}

type covariance struct {
	kind int
	welford2
}

func (fn *covariance) Value(ctx sqlite3.Context) {
	var r float64
	switch fn.kind {
	case var_pop:
		r = fn.covar_pop()
	case var_samp:
		r = fn.covar_samp()
	case corr:
		r = fn.correlation()
	}
	ctx.ResultFloat(r)
}

func (fn *covariance) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	a, b := arg[0], arg[1]
	if a.Type() != sqlite3.NULL && b.Type() != sqlite3.NULL {
		fn.enqueue(a.Float(), b.Float())
	}
}

func (fn *covariance) Inverse(ctx sqlite3.Context, arg ...sqlite3.Value) {
	a, b := arg[0], arg[1]
	if a.Type() != sqlite3.NULL && b.Type() != sqlite3.NULL {
		fn.dequeue(a.Float(), b.Float())
	}
}
