# Go `"adiantum"` SQLite VFS

This package wraps an SQLite VFS to offer encryption at rest.

> [!WARNING]
> This work was not certified by a cryptographer.
> If you need vetted encryption, you should purchase the
> [SQLite Encryption Extension](https://sqlite.org/see),
> and either wrap it, or seek assistance wrapping it.

The `"adiantum"` VFS wraps the default SQLite VFS using the
[Adiantum](https://github.com/lukechampine/adiantum)
tweakable and length-preserving encryption.

In general, any HBSH construction can be used to wrap any VFS.

The default Adiantum construction uses XChaCha12 for its stream cipher,
AES for its block cipher, and NH and Poly1305 for hashing.
It uses Argon2id to derive keys from plain text.

> [!IMPORTANT]
> Adiantum is typically used for disk encryption.
> The standard threat model for disk encryption considers an adversary
> that can read multiple snapshots of a disk.
> The security property that disk encryption provides is that
> the only information such an adversary can determine is
> whether the data in a sector has or has not changed over time.

The VFS encrypts database files, rollback and statement journals, and WAL files.