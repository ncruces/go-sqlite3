package xts

import (
	"crypto/aes"
	"crypto/rand"
	"crypto/sha512"

	"golang.org/x/crypto/pbkdf2"
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
		n, _ := rand.Read(key)
		return key[:n]
	}
	return pbkdf2.Key([]byte(text), []byte(pepper), 10_000, 32, sha512.New)
}
