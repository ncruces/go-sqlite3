package xts

import (
	"crypto/aes"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha512"

	"golang.org/x/crypto/xts"
)

// This variable can be replaced with -ldflags:
//
//	go build -ldflags="-X github.com/ncruces/go-sqlite3/vfs/xts.pepper=xts"
var pepper = "github.com/ncruces/go-sqlite3/vfs/xts"

type aesCreator struct{}

func (aesCreator) XTS(key []byte) *xts.Cipher {
	c, err := xts.NewCipher(aes.NewCipher, key)
	if err != nil {
		return nil
	}
	return c
}

func (aesCreator) KDF(text string) []byte {
	if text == "" {
		key := make([]byte, 32)
		rand.Read(key)
		return key
	}
	key, err := pbkdf2.Key(sha512.New, text, []byte(pepper), 10_000, 32)
	if err != nil {
		panic(err)
	}
	return key
}
