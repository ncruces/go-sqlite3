package hash

import (
	"crypto"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
)

func sha224Func(ctx sqlite3.Context, arg ...sqlite3.Value) {
	hashFunc(ctx, arg[0], crypto.SHA224)
}

func sha384Func(ctx sqlite3.Context, arg ...sqlite3.Value) {
	hashFunc(ctx, arg[0], crypto.SHA384)
}

func sha256Func(ctx sqlite3.Context, arg ...sqlite3.Value) {
	size := 256
	if len(arg) > 1 {
		size = arg[1].Int()
	}

	switch size {
	case 224:
		hashFunc(ctx, arg[0], crypto.SHA224)
	case 256:
		hashFunc(ctx, arg[0], crypto.SHA256)
	default:
		ctx.ResultError(util.ErrorString("sha256: size must be 224, 256"))
	}
}

func sha512Func(ctx sqlite3.Context, arg ...sqlite3.Value) {
	size := 512
	if len(arg) > 1 {
		size = arg[1].Int()
	}

	switch size {
	case 224:
		hashFunc(ctx, arg[0], crypto.SHA512_224)
	case 256:
		hashFunc(ctx, arg[0], crypto.SHA512_256)
	case 384:
		hashFunc(ctx, arg[0], crypto.SHA384)
	case 512:
		hashFunc(ctx, arg[0], crypto.SHA512)
	default:
		ctx.ResultError(util.ErrorString("sha512: size must be 224, 256, 384, 512"))
	}

}
