# Sketch BLOB storage format

Serialization format for the HLL (cardinality), REQ (quantile), and FI (top-k)
sketch BLOBs produced and consumed by the `approx` SQLite extension in
[go-sqlite3](https://github.com/ncruces/go-sqlite3).

## Header byte

Every sketch BLOB begins with a single header byte using the
[Aggregate Knowledge HLL Storage Specification](https://github.com/aggregateknowledge/hll-storage-spec)
nibble layout: high nibble = schema version (`1`), low nibble = type code.

| First byte    | Type | Format |
| ------------- | ---- | ------ |
| `0x11`–`0x14` | 1–4  | HLL (`EMPTY`, `EXPLICIT`, `SPARSE`, `FULL`) |
| `0x15`        | 5    | REQ (quantiles) |
| `0x16`        | 6    | FI (frequent items / top-k) |

Types `5` and `6` are private extensions in unassigned HLL v1 slots so sketch
types can be told apart by their first byte; everything after byte 0 is
format-specific.

## HLL sketch (cardinality)

HLL sketch BLOBs follow the
[Aggregate Knowledge HLL Storage Specification](https://github.com/aggregateknowledge/hll-storage-spec)
verbatim. `accumulate` emits `log2m = 11`, `regwidth = 5`, sparse
enabled, auto explicit threshold; `combine` requires inputs to share `log2m` and
`regwidth` and preserves them.

**Hashing.** Rows with no arguments or any `NULL` argument are ignored. Starting
from `seed = 0`, each argument updates `seed` to the first 64 bits (`h1`) of
`MurmurHash3_x64_128` initialized with `h1 = h2 = seed'`:

- **integer** (and **float** exactly representable as `int64`, incl. `±0.0`):
  `seed' = seed`, 8-byte little-endian `i64`;
- **text**: `seed' = seed`, raw bytes;
- **blob**: `seed' = seed ^ 1`, raw bytes;
- **float** (other): `seed' = seed ^ 2`, 8-byte little-endian `f64`.

If any step yields `0`, the row is ignored; otherwise the final `seed` is added
to the HLL as a raw 64-bit hash. For single integer (`bigint`) or text values,
this matches [`postgresql-hll`](https://github.com/citusdata/postgresql-hll).

## REQ & FI conventions

The remaining sections specify the `go-sqlite3` custom formats (`0x15` and `0x16`):

- Fixed-width integers (`u16`, `i64`) and floats (`f32`, `f64`) are **little-endian**;
- `uvarint` is an unsigned LEB128 varint (Go `encoding/binary` `Uvarint`, `uint64`);
- `f32`/`f64` are IEEE-754 binary32/binary64 bit patterns; `NaN` is never valid.

## REQ sketch (quantiles)

A [relative-error quantiles sketch](https://arxiv.org/abs/2004.01668) in
high-rank-accuracy mode, storing `f32` values across weighted levels.
An item at level `h` represents `2^h` input items.

```
byte     header    = 0x15
byte     k         = 40        // initial section size; must be 40
level*                         // levels h = 0, 1, ... until EOF (at most 53)
```

An empty sketch is just the 2 header bytes. Levels repeat until the end of the
BLOB (the level count is implicit; a 54th level is invalid), contiguous from
`h = 0` with no empty levels:

```
uvarint  state     // compaction counter for level h
uvarint  count     // items in buffer, 1 <= count < 2*s*k (see below)
f32[]    buf       // count values; ±Inf allowed, no NaNs
```

Writers sort `buf` ascending; readers must not rely on the order.

The section parameters `(s, k)` are a function of `state`:
starting at `s = 3`, `k = 40`, whenever `state >= 2^(s-1)` the number of
sections doubles and `k` is divided by `sqrt(2)`, rounded to the nearest even
integer (minimum 4):

| `state`   | `s` | `k` |
| --------- | --- | --- |
| `< 2^2`   | 3   | 40  |
| `< 2^5`   | 6   | 28  |
| `< 2^11`  | 12  | 20  |
| `< 2^23`  | 24  | 14  |
| `< 2^47`  | 48  | 10  |
| otherwise | 96  | 8   |

**Compaction.** When a level reaches `2*s*k` items, a prefix of its smallest
items is compacted in pairs into level `h + 1` (retaining the exact minimum at
`h = 0`; see the paper and `req.go`), and `state` is incremented. Cascading
keeps every level below `2*s*k` items.

**Estimates.** Total weight `N = Σ 2^h * count_h`. This implementation estimates
`quantile(p)` by linear interpolation at weighted rank `p * (N - 1)` (matching
`percentile_cont`), and `percent_rank(v)` as its inverse (interpolating across
the weight of the largest item `< v`, divided by `N - 1`).

**Merge.** Concatenate buffers level by level (bitwise-ORing `state`), then
compact from `h = 0`.

## FI sketch (top-k)

A weighted, mergeable [Misra-Gries](https://arxiv.org/abs/1705.07001)
frequent-items summary. A single `0x16` byte (`len == 1`) is a valid empty
sketch; otherwise:

```
byte     header    = 0x16
u16      k         // target top-K, 1 <= k <= 1000
u16      c         // capacity, k <= c <= 4000 (writers use max(128, 4*k))
u16      count     // number of entries, 0 <= count <= c
f64      total     // total input weight, total >= 0
f64      offset    // total weight purged by compaction, offset >= 0
entry[]  entries   // count entries; trailing bytes are invalid
```

Each entry:

```
byte     type      // 1 = integer, 2 = float, 3 = text
f64      weight    // weight > 0
value              // depends on type
```

- **integer (`1`)**: `i64`. Any numeric value exactly representable as `int64`
  (including `0.0` and `-0.0`) uses this type;
- **float (`2`)**: `f64`; `±Inf` allowed, must not be `NaN` or exactly
  representable as `int64`;
- **text (`3`)**: `uvarint` byte length, followed by raw bytes (UTF-8 is not
  validated on read).

Entries are unique by `(type, value)`. Writers sort entries by descending
`weight` (breaking ties by ascending SQLite value order); readers must not rely
on the order.

**Estimates.** Each entry's true weight lies in `[weight, weight + offset]`, and
this implementation reports the upper bound `weight + offset` (untracked items
have true weight at most `offset`). The top-k are the `k` heaviest entries.

**Compaction.** Writers compact to at most `c` entries before serializing. To
compact when entry count exceeds `c`, let `δ` be the `(c+1)`-th largest weight:
drop entries with `weight <= δ`, subtract `δ` from every remaining weight, and
add `δ` to `offset`. *(In memory, compaction is deferred until inserting a new
key at `2*c` entries or merging to `>= 2*c` entries.)*

**Merge.** The 1-byte empty sketch is the identity. Otherwise take
`k = min(k1, k2)`, `c = min(c1, c2)`, sum `total`s and `offset`s, sum weights of
matching entries, and compact.
