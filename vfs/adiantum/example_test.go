//go:build ((linux || darwin || windows || freebsd || openbsd || netbsd || dragonfly || illumos) && !sqlite3_nosys) || sqlite3_flock || sqlite3_dotlk

package adiantum_test

import (
	"crypto/rand"
	"log"
	"os"

	"golang.org/x/crypto/argon2"
	"lukechampine.com/adiantum/hbsh"
	"lukechampine.com/adiantum/hpolyc"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/ncruces/go-sqlite3/vfs/adiantum"
)

func ExampleRegister_hpolyc() {
	vfs.Register("hpolyc", adiantum.Wrap(vfs.Find(""), hpolycCreator{}))

	db, err := sqlite3.Open("file:demo.db?vfs=hpolyc" +
		"&textkey=correct+horse+battery+staple")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove("./demo.db")
	defer db.Close()
	// Output:
}

type hpolycCreator struct{}

// HBSH creates an HBSH cipher given a key.
func (hpolycCreator) HBSH(key []byte) *hbsh.HBSH {
	if len(key) != 32 {
		// Key is not appropriate, return nil.
		return nil
	}
	return hpolyc.New(key)
}

// KDF gets a key from a secret.
func (hpolycCreator) KDF(secret string) []byte {
	if secret == "" {
		// No secret is given, generate a random key.
		key := make([]byte, 32)
		n, _ := rand.Read(key)
		return key[:n]
	}
	// Hash the secret with a KDF.
	return argon2.IDKey([]byte(secret), []byte("hpolyc"), 3, 64*1024, 4, 32)
}
