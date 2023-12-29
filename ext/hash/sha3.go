package hash

import (
	"crypto"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
)

func sha3Func(ctx sqlite3.Context, arg ...sqlite3.Value) {
	size := 256
	if len(arg) > 1 {
		size = arg[1].Int()
	}

	switch size {
	case 224:
		hashFunc(ctx, arg[0], crypto.SHA3_224)
	case 256:
		hashFunc(ctx, arg[0], crypto.SHA3_256)
	case 384:
		hashFunc(ctx, arg[0], crypto.SHA3_384)
	case 512:
		hashFunc(ctx, arg[0], crypto.SHA3_512)
	default:
		ctx.ResultError(util.ErrorString("sha3: size must be 224, 256, 384, 512"))
	}
}
