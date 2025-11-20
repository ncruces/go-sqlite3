// Package litestream implements a Litestream lightweight read-replica VFS.
package litestream

import (
	"log/slog"
	"sync"
	"time"

	"github.com/benbjohnson/litestream"
	"github.com/ncruces/go-sqlite3/vfs"
)

const (
	// The default poll interval.
	DefaultPollInterval = 1 * time.Second

	// The default cache size: 10 MiB.
	DefaultCacheSize = 10 * 1024 * 1024
)

func init() {
	vfs.Register("litestream", liteVFS{})
}

var (
	liteMtx sync.RWMutex
	// +checklocks:liteMtx
	liteDBs = map[string]*liteDB{}
)

// ReplicaOptions represents options for [NewReplica].
type ReplicaOptions struct {
	// Where to log error messages. May be nil.
	Logger *slog.Logger

	// Replica poll interval.
	// Should be less than the compaction interval
	// used by the replica at MinLevel+1.
	PollInterval time.Duration

	// Minimum compaction level to track.
	MinLevel int

	// CacheSize is the maximum size of the page cache in bytes.
	// Zero means DefaultCacheSize, negative disables caching.
	CacheSize int
}

// NewReplica creates a read-replica from a Litestream client.
func NewReplica(name string, client litestream.ReplicaClient, options ReplicaOptions) {
	if options.Logger != nil {
		options.Logger = options.Logger.With("name", name)
	} else {
		options.Logger = slog.New(slog.DiscardHandler)
	}
	if options.PollInterval <= 0 {
		options.PollInterval = DefaultPollInterval
	}
	if options.CacheSize == 0 {
		options.CacheSize = DefaultCacheSize
	}

	liteMtx.Lock()
	defer liteMtx.Unlock()
	liteDBs[name] = &liteDB{
		client: client,
		opts:   options,
		cache:  pageCache{size: options.CacheSize},
	}
}

// RemoveReplica removes a replica by name.
func RemoveReplica(name string) {
	liteMtx.Lock()
	defer liteMtx.Unlock()
	delete(liteDBs, name)
}
