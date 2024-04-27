// Package adiantum wraps an SQLite VFS to offer encryption at rest.
//
// The "adiantum" [vfs.VFS] wraps the default VFS using the
// Adiantum tweakable, length-preserving encryption.
//
// Importing package adiantum registers that VFS:
//
//	import _ "github.com/ncruces/go-sqlite3/vfs/adiantum"
//
// To open an encrypted database you need to provide key material.
//
// The simplest way to do that is to specify the key through an [URI] parameter:
//
//   - key: key material in binary (32 bytes)
//   - hexkey: key material in hex (64 hex digits)
//   - textkey: key material in text (any length)
//
// However, this makes your key easily accessible to other parts of
// your application (e.g. through [vfs.Filename.URIParameters]).
//
// To avoid this, use any of the following PRAGMAs:
//
//	PRAGMA key='D41d8cD98f00b204e9800998eCf8427e';
//	PRAGMA hexkey='e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855';
//	PRAGMA textkey='your-secret-key';
//
// [URI]: https://sqlite.org/uri.html
package adiantum

import (
	"github.com/ncruces/go-sqlite3/vfs"
	"lukechampine.com/adiantum/hbsh"
)

func init() {
	Register("adiantum", vfs.Find(""), nil)
}

// Register registers an encrypting VFS, wrapping a base VFS,
// and possibly using a custom HBSH cipher construction.
// To use the default Adiantum construction, set cipher to nil.
func Register(name string, base vfs.VFS, cipher HBSHCreator) {
	if cipher == nil {
		cipher = adiantumCreator{}
	}
	vfs.Register(name, &hbshVFS{
		VFS:  base,
		hbsh: cipher,
	})
}

// HBSHCreator creates an [hbsh.HBSH] cipher
// given key material.
type HBSHCreator interface {
	// KDF derives an HBSH key from a secret.
	// If no secret is given, a random key is generated.
	KDF(secret string) (key []byte)

	// HBSH creates an HBSH cipher given a key.
	// If key is not appropriate, nil is returned.
	HBSH(key []byte) *hbsh.HBSH
}
