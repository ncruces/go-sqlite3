// Package stats provides aggregate functions for statistics.
//
// Provided functions:
//   - stddev_pop: population standard deviation
//   - stddev_samp: sample standard deviation
//   - var_pop: population variance
//   - var_samp: sample variance
//   - covar_pop: population covariance
//   - covar_samp: sample covariance
//   - corr: correlation coefficient
//   - regr_r2: correlation coefficient squared
//   - regr_avgx: average of the independent variable
//   - regr_avgy: average of the dependent variable
//   - regr_sxx: sum of the squares of the independent variable
//   - regr_syy: sum of the squares of the dependent variable
//   - regr_sxy: sum of the products of each pair of variables
//   - regr_count: count non-null pairs of variables
//   - regr_slope: slope of the least-squares-fit linear equation
//   - regr_intercept: y-intercept of the least-squares-fit linear equation
//   - regr_json: all regr stats in a JSON object
//   - percentile_disc: discrete percentile
//   - percentile_cont: continuous percentile
//   - median: median value
//   - every: boolean and
//   - some: boolean or
//
// These join the [Built-in Aggregate Functions]:
//   - count: count rows/values
//   - sum: sum values
//   - avg: average value
//   - min: minimum value
//   - max: maximum value
//
// And the [Built-in Window Functions]:
//   - rank: rank of the current row with gaps
//   - dense_rank: rank of the current row without gaps
//   - percent_rank: relative rank of the row
//   - cume_dist: cumulative distribution
//
// See: [ANSI SQL Aggregate Functions]
//
// [Built-in Aggregate Functions]: https://sqlite.org/lang_aggfunc.html
// [Built-in Window Functions]: https://sqlite.org/windowfunctions.html#builtins
// [ANSI SQL Aggregate Functions]: https://www.oreilly.com/library/view/sql-in-a/9780596155322/ch04s02.html
package stats

import (
	"errors"

	"github.com/ncruces/go-sqlite3"
)

// Register registers statistics functions.
func Register(db *sqlite3.Conn) error {
	const flags = sqlite3.DETERMINISTIC | sqlite3.INNOCUOUS
	return errors.Join(
		db.CreateWindowFunction("var_pop", 1, flags, newVariance(var_pop)),
		db.CreateWindowFunction("var_samp", 1, flags, newVariance(var_samp)),
		db.CreateWindowFunction("stddev_pop", 1, flags, newVariance(stddev_pop)),
		db.CreateWindowFunction("stddev_samp", 1, flags, newVariance(stddev_samp)),
		db.CreateWindowFunction("covar_pop", 2, flags, newCovariance(var_pop)),
		db.CreateWindowFunction("covar_samp", 2, flags, newCovariance(var_samp)),
		db.CreateWindowFunction("corr", 2, flags, newCovariance(corr)),
		db.CreateWindowFunction("regr_r2", 2, flags, newCovariance(regr_r2)),
		db.CreateWindowFunction("regr_sxx", 2, flags, newCovariance(regr_sxx)),
		db.CreateWindowFunction("regr_syy", 2, flags, newCovariance(regr_syy)),
		db.CreateWindowFunction("regr_sxy", 2, flags, newCovariance(regr_sxy)),
		db.CreateWindowFunction("regr_avgx", 2, flags, newCovariance(regr_avgx)),
		db.CreateWindowFunction("regr_avgy", 2, flags, newCovariance(regr_avgy)),
		db.CreateWindowFunction("regr_slope", 2, flags, newCovariance(regr_slope)),
		db.CreateWindowFunction("regr_intercept", 2, flags, newCovariance(regr_intercept)),
		db.CreateWindowFunction("regr_count", 2, flags, newCovariance(regr_count)),
		db.CreateWindowFunction("regr_json", 2, flags, newCovariance(regr_json)),
		db.CreateWindowFunction("median", 1, flags, newPercentile(median)),
		db.CreateWindowFunction("percentile_cont", 2, flags, newPercentile(percentile_cont)),
		db.CreateWindowFunction("percentile_disc", 2, flags, newPercentile(percentile_disc)),
		db.CreateWindowFunction("every", 1, flags, newBoolean(every)),
		db.CreateWindowFunction("some", 1, flags, newBoolean(some)))
}

const (
	var_pop = iota
	var_samp
	stddev_pop
	stddev_samp
	corr
	regr_r2
	regr_sxx
	regr_syy
	regr_sxy
	regr_avgx
	regr_avgy
	regr_slope
	regr_intercept
	regr_count
	regr_json
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
	if a := arg[0]; a.NumericType() != sqlite3.NULL {
		fn.enqueue(a.Float())
	}
}

func (fn *variance) Inverse(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if a := arg[0]; a.NumericType() != sqlite3.NULL {
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
	case regr_r2:
		r = fn.regr_r2()
	case regr_sxx:
		r = fn.regr_sxx()
	case regr_syy:
		r = fn.regr_syy()
	case regr_sxy:
		r = fn.regr_sxy()
	case regr_avgx:
		r = fn.regr_avgx()
	case regr_avgy:
		r = fn.regr_avgy()
	case regr_slope:
		r = fn.regr_slope()
	case regr_intercept:
		r = fn.regr_intercept()
	case regr_count:
		ctx.ResultInt64(fn.regr_count())
		return
	case regr_json:
		ctx.ResultText(fn.regr_json())
		return
	}
	ctx.ResultFloat(r)
}

func (fn *covariance) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	b, a := arg[1], arg[0] // avoid a bounds check
	if a.NumericType() != sqlite3.NULL && b.NumericType() != sqlite3.NULL {
		fn.enqueue(a.Float(), b.Float())
	}
}

func (fn *covariance) Inverse(ctx sqlite3.Context, arg ...sqlite3.Value) {
	b, a := arg[1], arg[0] // avoid a bounds check
	if a.NumericType() != sqlite3.NULL && b.NumericType() != sqlite3.NULL {
		fn.dequeue(a.Float(), b.Float())
	}
}
