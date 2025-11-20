package litestream

import (
	"encoding/binary"
	"sync"

	"golang.org/x/sync/singleflight"

	"github.com/superfly/ltx"
)

type pageCache struct {
	single singleflight.Group
	pages  map[uint32]cachedPage // +checklocks:mtx
	size   int
	mtx    sync.Mutex
}

type cachedPage struct {
	data []byte
	txid ltx.TXID
}

func (c *pageCache) getOrFetch(pgno uint32, maxTXID ltx.TXID, fetch func() (any, error)) ([]byte, error) {
	if c.size >= 0 {
		c.mtx.Lock()
		if c.pages == nil {
			c.pages = map[uint32]cachedPage{}
		}
		page := c.pages[pgno]
		c.mtx.Unlock()

		if page.txid == maxTXID {
			return page.data, nil
		}
	}

	var key [12]byte
	binary.LittleEndian.PutUint32(key[0:], pgno)
	binary.LittleEndian.PutUint64(key[4:], uint64(maxTXID))
	v, err, _ := c.single.Do(string(key[:]), fetch)

	if err != nil {
		return nil, err
	}

	page := cachedPage{v.([]byte), maxTXID}
	if c.size >= 0 {
		c.mtx.Lock()
		c.evict(len(page.data))
		c.pages[pgno] = page
		c.mtx.Unlock()
	}
	return page.data, nil
}

// +checklocks:c.mtx
func (c *pageCache) evict(pageSize int) {
	// Evict random keys until we're under the maximum size.
	// SQLite has its own page cache, which it will use for each connection.
	// Since this is a second layer of shared cache,
	// random eviction is probably good enough.
	if pageSize*len(c.pages) < c.size {
		return
	}
	for key := range c.pages {
		delete(c.pages, key)
		if pageSize*len(c.pages) < c.size {
			return
		}
	}
}
