// Package xts wraps an SQLite VFS to offer encryption at rest.
package xts

import (
	"github.com/ncruces/go-sqlite3/vfs"
	"golang.org/x/crypto/xts"
)

func init() {
	Register("xts", vfs.Find(""), nil)
}

// Register registers an encrypting VFS, wrapping a base VFS,
// and possibly using a custom XTS cipher construction.
// To use the default AES-XTS construction, set cipher to nil.
func Register(name string, base vfs.VFS, cipher XTSCreator) {
	if cipher == nil {
		cipher = aesCreator{}
	}
	vfs.Register(name, &xtsVFS{
		VFS:  base,
		init: cipher,
	})
}

// XTSCreator creates an [xts.Cipher]
// given key material.
type XTSCreator interface {
	// KDF derives an XTS key from a secret.
	// If no secret is given, a random key is generated.
	KDF(secret string) (key []byte)

	// XTS creates an XTS cipher given a key.
	// If key is not appropriate, nil is returned.
	XTS(key []byte) *xts.Cipher
}
