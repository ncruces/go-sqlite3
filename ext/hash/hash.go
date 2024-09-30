// Package hash provides cryptographic hash functions.
//
// Provided functions:
//   - md4(data)
//   - md5(data)
//   - sha1(data)
//   - sha3(data, size) (default size 256)
//   - sha224(data)
//   - sha256(data, size) (default size 256)
//   - sha384(data)
//   - sha512(data, size) (default size 512)
//   - blake2s(data)
//   - blake2b(data, size) (default size 512)
//   - ripemd160(data)
//
// Each SQL function will only be registered if the corresponding
// [crypto.Hash] function is available.
// To ensure a specific hash function is available,
// import the implementing package.
package hash

import (
	"crypto"
	"errors"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
)

// Register registers cryptographic hash functions for a database connection.
func Register(db *sqlite3.Conn) error {
	const flags = sqlite3.DETERMINISTIC | sqlite3.INNOCUOUS

	var errs util.ErrorJoiner
	if crypto.MD4.Available() {
		errs.Join(
			db.CreateFunction("md4", 1, flags, md4Func))
	}
	if crypto.MD5.Available() {
		errs.Join(
			db.CreateFunction("md5", 1, flags, md5Func))
	}
	if crypto.SHA1.Available() {
		errs.Join(
			db.CreateFunction("sha1", 1, flags, sha1Func))
	}
	if crypto.SHA3_512.Available() {
		errs.Join(
			db.CreateFunction("sha3", 1, flags, sha3Func),
			db.CreateFunction("sha3", 2, flags, sha3Func))
	}
	if crypto.SHA256.Available() {
		errs.Join(
			db.CreateFunction("sha224", 1, flags, sha224Func),
			db.CreateFunction("sha256", 1, flags, sha256Func),
			db.CreateFunction("sha256", 2, flags, sha256Func))
	}
	if crypto.SHA512.Available() {
		errs.Join(
			db.CreateFunction("sha384", 1, flags, sha384Func),
			db.CreateFunction("sha512", 1, flags, sha512Func),
			db.CreateFunction("sha512", 2, flags, sha512Func))
	}
	if crypto.BLAKE2s_256.Available() {
		errs.Join(
			db.CreateFunction("blake2s", 1, flags, blake2sFunc))
	}
	if crypto.BLAKE2b_512.Available() {
		errs.Join(
			db.CreateFunction("blake2b", 1, flags, blake2bFunc),
			db.CreateFunction("blake2b", 2, flags, blake2bFunc))
	}
	if crypto.RIPEMD160.Available() {
		errs.Join(
			db.CreateFunction("ripemd160", 1, flags, ripemd160Func))
	}
	return errors.Join(errs...)
}

func md4Func(ctx sqlite3.Context, arg ...sqlite3.Value) {
	hashFunc(ctx, arg[0], crypto.MD4)
}

func md5Func(ctx sqlite3.Context, arg ...sqlite3.Value) {
	hashFunc(ctx, arg[0], crypto.MD5)
}

func sha1Func(ctx sqlite3.Context, arg ...sqlite3.Value) {
	hashFunc(ctx, arg[0], crypto.SHA1)
}

func ripemd160Func(ctx sqlite3.Context, arg ...sqlite3.Value) {
	hashFunc(ctx, arg[0], crypto.RIPEMD160)
}

func hashFunc(ctx sqlite3.Context, arg sqlite3.Value, fn crypto.Hash) {
	var data []byte
	switch arg.Type() {
	case sqlite3.NULL:
		return
	case sqlite3.BLOB:
		data = arg.RawBlob()
	default:
		data = arg.RawText()
	}

	h := fn.New()
	h.Write(data)
	ctx.ResultBlob(h.Sum(nil))
}
