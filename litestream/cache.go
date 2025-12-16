package litestream

import (
	"context"
	"fmt"
	"sync"

	"github.com/benbjohnson/litestream"
	"github.com/superfly/ltx"
)

type pageCache struct {
	pages map[uint32]cachedPage // +checklocks:mtx
	size  int
	mtx   sync.Mutex
}

type cachedPage struct {
	data []byte
	txid ltx.TXID
}

func (c *pageCache) getOrFetch(ctx context.Context, client ReplicaClient, pgno uint32, elem ltx.PageIndexElem) ([]byte, error) {
	if c.size > 0 {
		c.mtx.Lock()
		page := c.pages[pgno]
		c.mtx.Unlock()

		if page.txid == elem.MaxTXID {
			return page.data, nil
		}
	}

	h, data, err := litestream.FetchPage(ctx, client, elem.Level, elem.MinTXID, elem.MaxTXID, elem.Offset, elem.Size)
	if err != nil {
		return nil, fmt.Errorf("fetch page: %w", err)
	}
	if pgno != h.Pgno {
		return nil, fmt.Errorf("fetch page: want %d, got %d", pgno, h.Pgno)
	}

	if c.size > 0 {
		c.mtx.Lock()
		if c.pages != nil {
			c.evict(len(data))
		} else {
			c.pages = map[uint32]cachedPage{}
		}
		c.pages[pgno] = cachedPage{data, elem.MaxTXID}
		c.mtx.Unlock()
	}
	return data, nil
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
