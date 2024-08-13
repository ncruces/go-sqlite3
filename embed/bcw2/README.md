# Embeddable Wasm build of SQLite

This folder includes an embeddable Wasm build of SQLite 3.46.1, including the experimental
[`BEGIN CONCURRENT`](https://sqlite.org/src/doc/begin-concurrent/doc/begin_concurrent.md) and
[Wal2](https://www.sqlite.org/cgi/src/doc/wal2/doc/wal2.md) patches.

> [!IMPORTANT]  
> This package is experimental.
> It is built from the `bedrock` branch of SQLite,
> since that is _currently_ the most stable, maintained branch to include both features.

> [!CAUTION]
> The Wal2 journaling mode creates databases that other versions of SQLite cannot access.

The build is easily reproducible, and verifiable, using
[Artifact Attestations](https://github.com/ncruces/go-sqlite3/attestations).