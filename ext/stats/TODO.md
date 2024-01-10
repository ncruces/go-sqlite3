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

## Linear regression

- [X] `REGR_AVGX(dependent, independent)`
- [X] `REGR_AVGY(dependent, independent)`
- [X] `REGR_SXX(dependent, independent)`
- [X] `REGR_SYY(dependent, independent)`
- [X] `REGR_SXY(dependent, independent)`
- [X] `REGR_COUNT(dependent, independent)`
- [X] `REGR_SLOPE(dependent, independent)`
- [X] `REGR_INTERCEPT(dependent, independent)`
- [X] `REGR_R2(dependent, independent)`

## Ordered set aggregates

- [ ] `CUME_DIST(value_list) WITHIN GROUP (ORDER BY sort_list)`
- [ ] `RANK(value_list) WITHIN GROUP (ORDER BY sort_list)`
- [ ] `DENSE_RANK(value_list) WITHIN GROUP (ORDER BY sort_list)`
- [ ] `PERCENT_RANK(value_list) WITHIN GROUP (ORDER BY sort_list)`
- [ ] `PERCENTILE_CONT(percentile) WITHIN GROUP (ORDER BY sort_list)`
- [ ] `PERCENTILE_DISC(percentile) WITHIN GROUP (ORDER BY sort_list)`