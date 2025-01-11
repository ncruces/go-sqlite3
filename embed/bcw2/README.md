# Embeddable Wasm build of SQLite

This folder includes an alternative embeddable Wasm build of SQLite,
which includes the experimental
[`BEGIN CONCURRENT`](https://sqlite.org/src/doc/begin-concurrent/doc/begin_concurrent.md) and
[Wal2](https://sqlite.org/cgi/src/doc/wal2/doc/wal2.md) patches.

It also enables the optional
[`UPDATE … ORDER BY … LIMIT`](https://sqlite.org/lang_update.html#optional_limit_and_order_by_clauses) and
[`DELETE … ORDER BY … LIMIT`](https://sqlite.org/lang_delete.html#optional_limit_and_order_by_clauses) clauses,
and the [`WITHIN GROUP ORDER BY`](https://sqlite.org/compile.html#enable_ordered_set_aggregates) aggregate syntax.

> [!IMPORTANT]  
> This package is experimental.
> It is built from the `bedrock` branch of SQLite,
> since that is _currently_ the most stable, maintained branch to include these features.

> [!CAUTION]
> The Wal2 journaling mode creates databases that other versions of SQLite cannot access.

The build is easily reproducible, and verifiable, using
[Artifact Attestations](https://github.com/ncruces/go-sqlite3/attestations).