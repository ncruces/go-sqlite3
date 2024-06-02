# ANSI SQL Aggregate Functions

https://www.oreilly.com/library/view/sql-in-a/9780596155322/ch04s02.html

## Built in aggregates

- [x] `COUNT(*)`
- [x] `COUNT(expression)`
- [x] `SUM(expression)`
- [x] `AVG(expression)`
- [x] `MIN(expression)`
- [x] `MAX(expression)`

https://sqlite.org/lang_aggfunc.html

## Statistical aggregates

- [x] `STDDEV_POP(expression)`
- [x] `STDDEV_SAMP(expression)`
- [x] `VAR_POP(expression)`
- [x] `VAR_SAMP(expression)`
- [x] `COVAR_POP(dependent, independent)`
- [x] `COVAR_SAMP(dependent, independent)`
- [x] `CORR(dependent, independent)`

## Linear regression aggregates

- [X] `REGR_AVGX(dependent, independent)`
- [X] `REGR_AVGY(dependent, independent)`
- [X] `REGR_SXX(dependent, independent)`
- [X] `REGR_SYY(dependent, independent)`
- [X] `REGR_SXY(dependent, independent)`
- [X] `REGR_COUNT(dependent, independent)`
- [X] `REGR_SLOPE(dependent, independent)`
- [X] `REGR_INTERCEPT(dependent, independent)`
- [X] `REGR_R2(dependent, independent)`

## Set aggregates

- [X] `CUME_DIST() OVER window`
- [X] `RANK() OVER window`
- [X] `DENSE_RANK() OVER window`
- [X] `PERCENT_RANK() OVER window`
- [ ] `PERCENTILE_CONT(percentile) OVER window`
- [ ] `PERCENTILE_DISC(percentile) OVER window`

https://sqlite.org/windowfunctions.html#builtins

## Additional aggregates

- [X] `MEDIAN(expression)`
- [X] `QUANTILE_CONT(expression, quantile)`
- [X] `QUANTILE_DISC(expression, quantile)`

https://duckdb.org/docs/sql/aggregates.html