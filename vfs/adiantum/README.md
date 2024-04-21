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
Additionally, we use Argon2id to derive 256-bit keys from plain text.

The VFS encrypts all files _except_
[super journals](https://sqlite.org/tempfiles.html#super_journal_files):
these _never_ contain database data, only filenames,
and padding them to the block size is problematic.

Temporary files _are_ encrypted with **random** keys,
as they _may_ contain database data.
To avoid the overhead of encrypting temporary files,
keep them in memory:

    PRAGMA temp_store = memory;

> [!IMPORTANT]
> Adiantum is typically used for disk encryption.
> The standard threat model for disk encryption considers an adversary
> that can read multiple snapshots of a disk.
> The only security property that disk encryption (and this package)
> provides is that the only information such an adversary can determine
> is whether the data in a sector has or has not changed over time.
