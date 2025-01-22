// Package stats provides aggregate functions for statistics.
//
// Provided functions:
//   - var_pop: population variance
//   - var_samp: sample variance
//   - stddev_pop: population standard deviation
//   - stddev_samp: sample standard deviation
//   - skewness_pop: Pearson population skewness
//   - skewness_samp: Pearson sample skewness
//   - kurtosis_pop: Fisher population excess kurtosis
//   - kurtosis_samp: Fisher sample excess kurtosis
//   - covar_pop: population covariance
//   - covar_samp: sample covariance
//   - corr: Pearson correlation coefficient
//   - regr_r2: correlation coefficient squared
//   - regr_avgx: average of the independent variable
//   - regr_avgy: average of the dependent variable
//   - regr_sxx: sum of the squares of the independent variable
//   - regr_syy: sum of the squares of the dependent variable
//   - regr_sxy: sum of the products of each pair of variables
//   - regr_count: count non-null pairs of variables
//   - regr_slope: slope of the least-squares-fit linear equation
//   - regr_intercept: y-intercept of the least-squares-fit linear equation
//   - regr_json: all regr stats as a JSON object
//   - percentile_disc: discrete quantile
//   - percentile_cont: continuous quantile
//   - percentile: continuous percentile
//   - median: middle value
//   - mode: most frequent value
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
	const order = sqlite3.SELFORDER1 | flags
	return errors.Join(
		db.CreateWindowFunction("var_pop", 1, flags, newVariance(var_pop)),
		db.CreateWindowFunction("var_samp", 1, flags, newVariance(var_samp)),
		db.CreateWindowFunction("stddev_pop", 1, flags, newVariance(stddev_pop)),
		db.CreateWindowFunction("stddev_samp", 1, flags, newVariance(stddev_samp)),
		db.CreateWindowFunction("skewness_pop", 1, flags, newMoments(skewness_pop)),
		db.CreateWindowFunction("skewness_samp", 1, flags, newMoments(skewness_samp)),
		db.CreateWindowFunction("kurtosis_pop", 1, flags, newMoments(kurtosis_pop)),
		db.CreateWindowFunction("kurtosis_samp", 1, flags, newMoments(kurtosis_samp)),
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
		db.CreateWindowFunction("median", 1, order, newPercentile(median)),
		db.CreateWindowFunction("percentile", 2, order, newPercentile(percentile_100)),
		db.CreateWindowFunction("percentile_cont", 2, order, newPercentile(percentile_cont)),
		db.CreateWindowFunction("percentile_disc", 2, order, newPercentile(percentile_disc)),
		db.CreateWindowFunction("every", 1, flags, newBoolean(every)),
		db.CreateWindowFunction("some", 1, flags, newBoolean(some)),
		db.CreateWindowFunction("mode", 1, order, newMode))
}

const (
	var_pop = iota
	var_samp
	stddev_pop
	stddev_samp
	skewness_pop
	skewness_samp
	kurtosis_pop
	kurtosis_samp
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

func special(kind int, n int64) (null, zero bool) {
	switch kind {
	case var_pop, stddev_pop, regr_sxx, regr_syy, regr_sxy:
		return n <= 0, n == 1
	case regr_avgx, regr_avgy:
		return n <= 0, false
	case kurtosis_samp:
		return n <= 3, false
	case skewness_samp:
		return n <= 2, false
	case skewness_pop:
		return n <= 1, n == 2
	default:
		return n <= 1, false
	}
}

func newVariance(kind int) func() sqlite3.AggregateFunction {
	return func() sqlite3.AggregateFunction { return &variance{kind: kind} }
}

type variance struct {
	kind int
	welford
}

func (fn *variance) Value(ctx sqlite3.Context) {
	switch null, zero := special(fn.kind, fn.n); {
	case zero:
		ctx.ResultFloat(0)
		return
	case null:
		return
	}

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
	a := arg[0]
	f := a.Float()
	if f != 0.0 || a.NumericType() != sqlite3.NULL {
		fn.enqueue(f)
	}
}

func (fn *variance) Inverse(ctx sqlite3.Context, arg ...sqlite3.Value) {
	a := arg[0]
	f := a.Float()
	if f != 0.0 || a.NumericType() != sqlite3.NULL {
		fn.dequeue(f)
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
	if fn.kind == regr_count {
		ctx.ResultInt64(fn.regr_count())
		return
	}
	switch null, zero := special(fn.kind, fn.n); {
	case zero:
		ctx.ResultFloat(0)
		return
	case null:
		return
	}

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
	case regr_json:
		var buf [128]byte
		ctx.ResultRawText(fn.regr_json(buf[:0]))
		return
	}
	ctx.ResultFloat(r)
}

func (fn *covariance) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	b, a := arg[1], arg[0] // avoid a bounds check
	fa := a.Float()
	fb := b.Float()
	if true &&
		(fa != 0.0 || a.NumericType() != sqlite3.NULL) &&
		(fb != 0.0 || b.NumericType() != sqlite3.NULL) {
		fn.enqueue(fa, fb)
	}
}

func (fn *covariance) Inverse(ctx sqlite3.Context, arg ...sqlite3.Value) {
	b, a := arg[1], arg[0] // avoid a bounds check
	fa := a.Float()
	fb := b.Float()
	if true &&
		(fa != 0.0 || a.NumericType() != sqlite3.NULL) &&
		(fb != 0.0 || b.NumericType() != sqlite3.NULL) {
		fn.dequeue(fa, fb)
	}
}

func newMoments(kind int) func() sqlite3.AggregateFunction {
	return func() sqlite3.AggregateFunction { return &momentfn{kind: kind} }
}

type momentfn struct {
	kind int
	moments
}

func (fn *momentfn) Value(ctx sqlite3.Context) {
	switch null, zero := special(fn.kind, fn.n); {
	case zero:
		ctx.ResultFloat(0)
		return
	case null:
		return
	}

	var r float64
	switch fn.kind {
	case skewness_pop:
		r = fn.skewness_pop()
	case skewness_samp:
		r = fn.skewness_samp()
	case kurtosis_pop:
		r = fn.kurtosis_pop()
	case kurtosis_samp:
		r = fn.kurtosis_samp()
	}
	ctx.ResultFloat(r)
}

func (fn *momentfn) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	a := arg[0]
	f := a.Float()
	if f != 0.0 || a.NumericType() != sqlite3.NULL {
		fn.enqueue(f)
	}
}

func (fn *momentfn) Inverse(ctx sqlite3.Context, arg ...sqlite3.Value) {
	a := arg[0]
	f := a.Float()
	if f != 0.0 || a.NumericType() != sqlite3.NULL {
		fn.dequeue(f)
	}
}
