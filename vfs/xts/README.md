# Go `xts` SQLite VFS

This package wraps an SQLite VFS to offer encryption at rest.

The `"xts"` VFS wraps the default SQLite VFS using the
[AES-XTS](https://pkg.go.dev/golang.org/x/crypto/xts)
tweakable and length-preserving encryption.\
In general, any XTS construction can be used to wrap any VFS.

The default AES-XTS construction uses AES-128, AES-192, or AES-256
for its block cipher.
Additionally, we use [PBKDF2-HMAC-SHA512](https://pkg.go.dev/golang.org/x/crypto/pbkdf2)
to derive AES-128 keys from plain text where needed.
File contents are encrypted in 512 byte sectors, matching the
[minimum](https://sqlite.org/fileformat.html#pages) SQLite page size.

The VFS encrypts all files _except_
[super journals](https://sqlite.org/tempfiles.html#super_journal_files):
these _never_ contain database data, only filenames,
and padding them to the sector size is problematic.
Temporary files _are_ encrypted with **random** AES-128 keys,
as they _may_ contain database data.
To avoid the overhead of encrypting temporary files,
keep them in memory:

    PRAGMA temp_store = memory;

> [!IMPORTANT]
> XTS is a cipher mode typically used for disk encryption.
> The standard threat model for disk encryption considers an adversary
> that can read multiple snapshots of a disk.
> The only security property that disk encryption provides
> is that all information such an adversary can obtain
> is whether the data in a sector has or has not changed over time.

The encryption offered by this package is fully deterministic.

This means that an adversary who can get ahold of multiple snapshots
(e.g. backups) of a database file can learn precisely:
which sectors changed, which ones didn't, which got reverted.

This is slightly weaker than other forms of SQLite encryption
that include *some* nondeterminism; with limited nondeterminism,
an adversary can't distinguish between
sectors that actually changed, and sectors that got reverted.

> [!CAUTION]
> This package does not claim protect databases against tampering or forgery.

The major practical consequence of the above point is that,
if you're keeping `"xts"` encrypted backups of your database,
and want to protect against forgery, you should sign your backups,
and verify signatures before restoring them.

This is slightly weaker than other forms of SQLite encryption
that include page-level [MACs](https://en.wikipedia.org/wiki/Message_authentication_code).
Page-level MACs can protect against forging individual pages,
but can't prevent them from being reverted to former versions of themselves.

> [!TIP]
> The [`"adiantum"`](../adiantum/README.md) package also offers encryption at rest.
> In general Adiantum performs significantly better,
> and as a "wide-block" cipher, _may_ offer improved security.