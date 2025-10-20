// Package litestream implements a Litestream lightweight read-replica VFS.
package litestream

import (
	"log/slog"
	"sync"
	"time"

	"github.com/benbjohnson/litestream"
	"github.com/ncruces/go-sqlite3/vfs"
)

// The default poll interval.
const DefaultPollInterval = 1 * time.Second

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
	// Minimum compaction level to track.
	MinLevel int
	// Replica poll interval. Must be less than the compaction interval
	// used by the replica at MinLevel+1.
	PollInterval time.Duration
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
	options.MinLevel = max(0, min(options.MinLevel, litestream.SnapshotLevel))

	liteMtx.Lock()
	defer liteMtx.Unlock()
	liteDBs[name] = &liteDB{
		client: client,
		opts:   &options,
	}
}

// RemoveReplica removes a replica by name.
func RemoveReplica(name string) {
	liteMtx.Lock()
	defer liteMtx.Unlock()
	delete(liteDBs, name)
}
