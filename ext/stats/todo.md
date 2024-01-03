# ANSI SQL Aggregate Functions

https://www.oreilly.com/library/view/sql-in-a/9780596155322/ch04s02.html

## Built in

- [x] `COUNT(*)`
- [x] `COUNT(expression)`
- [x] `SUM(expression)`
- [x] `AVG(expression)`
- [x] `MIN(expression)`
- [x] `MAX(expression)`

https://sqlite.org/lang_aggfunc.html

## Implemented

- [x] `STDDEV_POP(expression)`
- [x] `STDDEV_SAMP(expression)`
- [x] `VAR_POP(expression)`
- [x] `VAR_SAMP(expression)`
- [x] `COVAR_POP(dependent, independent)`
- [x] `COVAR_SAMP(dependent, independent)`
- [x] `CORR(dependent, independent)`

## Linear regression

- [ ] `REGR_AVGX(dependent, independent)`
- [ ] `REGR_AVGY(dependent, independent)`
- [ ] `REGR_COUNT(dependent, independent)`
- [ ] `REGR_INTERCEPT(dependent, independent)`
- [ ] `REGR_R2(dependent, independent)`
- [ ] `REGR_SLOPE(dependent, independent)`
- [ ] `REGR_SXX(dependent, independent)`
- [ ] `REGR_SXY(dependent, independent)`
- [ ] `REGR_SYY(dependent, independent)`

## Other

- [ ] `CUME_DIST(value_list) WITHIN GROUP (ORDER BY sort_list)`
- [ ] `RANK(value_list) WITHIN GROUP (ORDER BY sort_list)`
- [ ] `DENSE_RANK(value_list) WITHIN GROUP (ORDER BY sort_list)`
- [ ] `PERCENT_RANK(value_list) WITHIN GROUP (ORDER BY sort_list)`
- [ ] `PERCENTILE_CONT(percentile) WITHIN GROUP (ORDER BY sort_list)`
- [ ] `PERCENTILE_DISC(percentile) WITHIN GROUP (ORDER BY sort_list)`