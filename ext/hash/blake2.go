package hash

import (
	"crypto"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
)

func blake2sFunc(ctx sqlite3.Context, arg ...sqlite3.Value) {
	hashFunc(ctx, arg[0], crypto.BLAKE2s_256)
}

func blake2bFunc(ctx sqlite3.Context, arg ...sqlite3.Value) {
	size := 512
	if len(arg) > 1 {
		size = arg[1].Int()
	}

	switch size {
	case 256:
		hashFunc(ctx, arg[0], crypto.BLAKE2b_256)
	case 384:
		hashFunc(ctx, arg[0], crypto.BLAKE2b_384)
	case 512:
		hashFunc(ctx, arg[0], crypto.BLAKE2b_512)
	default:
		ctx.ResultError(util.ErrorString("blake2b: size must be 256, 384, 512"))
	}
}
