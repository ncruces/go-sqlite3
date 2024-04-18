package adiantum

import (
	"sync"

	"golang.org/x/crypto/argon2"
	"lukechampine.com/adiantum"
	"lukechampine.com/adiantum/hbsh"
)

const pepper = "github.com/ncruces/go-sqlite3/vfs/adiantum"

type adiantumCreator struct{}

func (adiantumCreator) HBSH(key []byte) *hbsh.HBSH {
	return adiantum.New(key)
}

func (adiantumCreator) KDF(text string) []byte {
	if key := keyCacheGet(text); key != nil {
		return key[:]
	}

	key := argon2.IDKey([]byte(text), []byte(pepper), 1, 64*1024, 4, 32)
	keyCachePut(text, (*[32]byte)(key))
	return key
}

const keyCacheMaxEntries = 100

var (
	// +checklocks:keyCacheMtx
	keyCache    = map[string]*[32]byte{}
	keyCacheMtx sync.RWMutex
)

func keyCacheGet(text string) *[32]byte {
	keyCacheMtx.RLock()
	defer keyCacheMtx.RUnlock()
	return keyCache[text]
}

func keyCachePut(text string, key *[32]byte) {
	keyCacheMtx.Lock()
	defer keyCacheMtx.Unlock()
	if len(keyCache) >= keyCacheMaxEntries {
		for k := range keyCache {
			delete(keyCache, k)
		}
	}
	keyCache[text] = key
}
