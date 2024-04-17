package adiantum

import (
	"golang.org/x/crypto/argon2"
	"lukechampine.com/adiantum"
	"lukechampine.com/adiantum/hbsh"
)

const salt = "github.com/ncruces/go-sqlite3/vfs/adiantum"

type adiantumCreator struct{}

func (adiantumCreator) HBSH(key []byte) *hbsh.HBSH {
	return adiantum.New(key)
}

func (adiantumCreator) KDF(text string) (key []byte) {
	return argon2.IDKey([]byte(text), []byte(salt), 1, 64*1024, 4, 32)
}
