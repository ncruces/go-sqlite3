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
//   - approx_top_k(X, [W,] K): approximate top K elements (and counts/sums) as JSON using a FI sketch.
//   - approx_top_k_accumulate(X, [W,] K): accumulates elements into a FI sketch BLOB for up to K elements.
//   - approx_top_k_combine(B): merges FI sketch BLOBs into a single sketch BLOB.
//   - approx_top_k_estimate(B [, K]): estimates the top K elements as JSON from a FI sketch BLOB.
//
// [HyperLogLog] sketch BLOBs use the [Aggregate Knowledge HLL Storage Specification].
//
// [Relative Error Quantiles] (REQ) sketches use high-rank accuracy mode,
// prioritizing precision at the upper tail (e.g. p95, p99, p99.9).
// Rank error matches typical KLL sketches at the median,
// and is at most 2x worse at the lower tail.
//
// [Frequent Items] (FI) sketches use weighted mergeable Misra-Gries summaries.
//
// [HyperLogLog]: https://github.com/segmentio/go-hll
// [Relative Error Quantiles]: https://arxiv.org/abs/2004.01668
// [Frequent Items]: https://arxiv.org/abs/1705.07001
// [Aggregate Knowledge HLL Storage Specification]: https://github.com/aggregateknowledge/hll-storage-spec
package approx

import (
	"errors"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/util/sql3util"
)

// Register registers approximate aggregate functions.
func Register(db *sqlite3.Conn) error {
	const (
		flags = sqlite3.DETERMINISTIC | sqlite3.INNOCUOUS
		json  = sqlite3.RESULT_SUBTYPE | flags
	)
	return errors.Join(
		db.CreateWindowFunction("approx_count_distinct", -1, flags,
			sql3util.WindowFromCombiner[hllSketch](1e4)),
		db.CreateWindowFunction("approx_count_distinct_accumulate", -1, flags,
			sql3util.WindowFromCombiner[hllAccumulate](1e4)),
		db.CreateWindowFunction("approx_count_distinct_combine", 1, flags,
			sql3util.WindowFromCombiner[hllCombine](1e4)),
		db.CreateFunction("approx_count_distinct_estimate", 1, flags,
			hllEstimate),
		db.CreateWindowFunction("approx_percentile", 2, json,
			sql3util.WindowFromCombiner[reqSketch](1e4)),
		db.CreateWindowFunction("approx_percent_rank", 2, json,
			sql3util.WindowFromCombiner[reqRank](1e4)),
		db.CreateWindowFunction("approx_percentile_accumulate", 1, flags,
			sql3util.WindowFromCombiner[reqAccumulate](1e4)),
		db.CreateWindowFunction("approx_percentile_combine", 1, flags,
			sql3util.WindowFromCombiner[reqCombine](1e4)),
		db.CreateFunction("approx_percentile_estimate", 2, json,
			reqPercentileEstimate),
		db.CreateFunction("approx_percent_rank_estimate", 2, json,
			reqRankEstimate),
		db.CreateWindowFunction("approx_top_k", 2, json,
			sql3util.WindowFromCombiner[topSketch](1e4)),
		db.CreateWindowFunction("approx_top_k", 3, json,
			sql3util.WindowFromCombiner[topSketch](1e4)),
		db.CreateWindowFunction("approx_top_k_accumulate", 2, flags,
			sql3util.WindowFromCombiner[topAccumulate](1e4)),
		db.CreateWindowFunction("approx_top_k_accumulate", 3, flags,
			sql3util.WindowFromCombiner[topAccumulate](1e4)),
		db.CreateWindowFunction("approx_top_k_combine", 1, flags,
			sql3util.WindowFromCombiner[topCombine](1e4)),
		db.CreateFunction("approx_top_k_estimate", 1, json,
			topEstimate),
		db.CreateFunction("approx_top_k_estimate", 2, json,
			topEstimate),
	)
}
