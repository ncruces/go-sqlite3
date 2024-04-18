// Package adiantum wraps an SQLite VFS to offer encryption at rest.
//
// The "adiantum" [vfs.VFS] wraps the default VFS using the
// Adiantum tweakable length-preserving encryption.
//
// Importing package adiantum registers that VFS.
//
//	import _ "github.com/ncruces/go-sqlite3/vfs/adiantum"
//
// To open an encrypted database you need to provide key material.
// This is done through [URI] parameters:
//
//   - key: key material in binary (32 bytes)
//   - hexkey: key material in hex (64 hex digits)
//   - textkey: key material in text (any length)
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
	vfs.Register("adiantum", &hbshVFS{
		VFS:  base,
		hbsh: cipher,
	})
}

// HBSHCreator creates an [hbsh.HBSH] cipher,
// given key material.
type HBSHCreator interface {
	// KDF maps a secret (text) to a key of the appropriate size.
	KDF(text string) (key []byte)

	// HBSH creates an HBSH cipher given an appropriate key.
	HBSH(key []byte) *hbsh.HBSH
}
