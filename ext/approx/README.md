# Approximate statistics SQLite extension

This package provides approximate aggregate functions for SQLite,
using mergeable data sketches:
- **distinct counts** (HyperLogLog), e.g. daily, weekly, monthly active users;
- **quantiles** (REQ), e.g. median and tail latencies;
- **top-K** (Frequent Items), e.g. most popular content, highest spenders.

Computing these exactly requires memory that grows with the input,
and exact results don't combine:
you can't sum seven daily counts of active users to get a weekly count,
or average daily medians to get a monthly median.

Sketches trade a small, bounded error for compact size
(about 1 KiB for distinct counts, tens of KiB for quantiles over millions of rows)
and mergeability: combining two sketches produces a sketch of the combined data.

### Roll-up tables

Sketches serialize to BLOBs, so you can store them in roll-up tables.
For example, you can accumulate daily sketches from raw events:

```sql
CREATE TABLE daily (day TEXT PRIMARY KEY, users BLOB, latency BLOB, pages BLOB);

INSERT INTO daily
SELECT date(time),
       approx_count_distinct_accumulate(user_id),
       approx_percentile_accumulate(latency),
       approx_top_k_accumulate(path, 10)
FROM events
GROUP BY 1;
```

You can then combine them to answer coarser queries without rescanning `events`:

```sql
SELECT strftime('%Y-%m', day) AS month,
       approx_count_distinct_estimate(approx_count_distinct_combine(users)) AS users,
       approx_percentile_estimate(approx_percentile_combine(latency), 0.99) AS p99,
       approx_top_k_estimate(approx_top_k_combine(pages), 3) AS top_pages
FROM daily
GROUP BY 1;
```

### Window functions

Every aggregate also works as a window function,
e.g. to compute a rolling count of weekly active users:

```sql
SELECT day,
       approx_count_distinct_estimate(
         approx_count_distinct_combine(users) OVER (ORDER BY day ROWS 6 PRECEDING)
       ) AS weekly_users
FROM daily;
```

### Storage format

The sketch BLOB storage format is specified in [`format.md`](format.md).
