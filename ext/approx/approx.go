// Package approx provides approximate aggregate functions.
//
// Provided functions:
//   - approx_count_distinct(X, ...): approximate count of distinct elements using HyperLogLog.
//   - approx_count_distinct_accumulate(X, ...): accumulates elements into a HyperLogLog sketch BLOB.
//   - approx_count_distinct_combine(B): merges HyperLogLog sketch BLOBs into a single sketch BLOB.
//   - approx_count_distinct_estimate(B): estimates the cardinality from a HyperLogLog sketch BLOB.
//   - approx_percentile(X, P): approximate continuous quantile(s) P in [0, 1] using a REQ sketch.
//   - approx_percent_rank(X, V): approximate relative rank(s) in [0, 1] of V using a REQ sketch.
//   - approx_percentile_accumulate(X): accumulates elements into a REQ sketch BLOB.
//   - approx_percentile_combine(B): merges REQ sketch BLOBs into a single sketch BLOB.
//   - approx_percentile_estimate(B, P): estimates quantile(s) P in [0, 1] from a REQ sketch BLOB.
//   - approx_percent_rank_estimate(B, V): estimates relative rank(s) in [0, 1] of V from a REQ sketch BLOB.
//
// HyperLogLog sketch BLOBs use the [Aggregate Knowledge HLL Storage Specification],
// compatible with PostgreSQL's postgresql-hll extension.
//
// [Aggregate Knowledge HLL Storage Specification]: https://github.com/aggregateknowledge/hll-storage-spec
package approx

import (
	"errors"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/util/sql3util"
)

// Register registers approximate aggregate functions.
func Register(db *sqlite3.Conn) error {
	const flags = sqlite3.DETERMINISTIC | sqlite3.INNOCUOUS
	return errors.Join(
		db.CreateWindowFunction("approx_count_distinct", -1, flags,
			sql3util.WindowFromCombiner[hllSketch](1e4)),
		db.CreateWindowFunction("approx_count_distinct_accumulate", -1, flags,
			sql3util.WindowFromCombiner[hllAccumulate](1e4)),
		db.CreateWindowFunction("approx_count_distinct_combine", 1, flags,
			sql3util.WindowFromCombiner[hllCombine](1e4)),
		db.CreateFunction("approx_count_distinct_estimate", 1, flags,
			hllEstimate),
		db.CreateWindowFunction("approx_percentile", 2, flags,
			sql3util.WindowFromCombiner[reqSketch](1e4)),
		db.CreateWindowFunction("approx_percent_rank", 2, flags,
			sql3util.WindowFromCombiner[reqRank](1e4)),
		db.CreateWindowFunction("approx_percentile_accumulate", 1, flags,
			sql3util.WindowFromCombiner[reqAccumulate](1e4)),
		db.CreateWindowFunction("approx_percentile_combine", 1, flags,
			sql3util.WindowFromCombiner[reqCombine](1e4)),
		db.CreateFunction("approx_percentile_estimate", 2, flags,
			reqPercentileEstimate),
		db.CreateFunction("approx_percent_rank_estimate", 2, flags,
			reqRankEstimate),
	)
}
