// Package xts wraps an SQLite VFS to offer encryption at rest.
//
// The "xts" [vfs.VFS] wraps the default VFS using the
// AES-XTS tweakable, length-preserving encryption.
//
// Importing package xts registers that VFS:
//
//	import _ "github.com/ncruces/go-sqlite3/vfs/xts"
//
// To open an encrypted database you need to provide key material.
//
// The simplest way to do that is to specify the key through an [URI] parameter:
//
//   - key: key material in binary (32, 48 or 64 bytes)
//   - hexkey: key material in hex (64, 96 or 128 hex digits)
//   - textkey: key material in text (any length)
//
// However, this makes your key easily accessible to other parts of
// your application (e.g. through [vfs.Filename.URIParameters]).
//
// To avoid this, invoke any of the following PRAGMAs
// immediately after opening a connection:
//
//	PRAGMA key='D41d8cD98f00b204e9800998eCf8427e';
//	PRAGMA hexkey='e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855';
//	PRAGMA textkey='your-secret-key';
//
// For an ATTACH-ed database, you must specify the schema name:
//
//	ATTACH DATABASE 'demo.db' AS demo;
//	PRAGMA demo.textkey='your-secret-key';
//
// [URI]: https://sqlite.org/uri.html
package xts

import (
	"golang.org/x/crypto/xts"

	"github.com/ncruces/go-sqlite3/vfs"
)

func init() {
	Register("xts", vfs.Find(""), nil)
}

// Register registers an encrypting VFS, wrapping a base VFS,
// and possibly using a custom XTS cipher construction.
// To use the default AES-XTS construction, set cipher to nil.
//
// The default construction uses AES-128, AES-192, or AES-256
// if the key/hexkey is 32, 48, or 64 bytes, respectively.
// If a textkey is provided, the default KDF is PBKDF2-HMAC-SHA512
// with 10,000 iterations, always producing a 32 byte key.
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
