package litestream

import (
	"log/slog"
	"sync"

	"github.com/benbjohnson/litestream"
	"github.com/ncruces/go-sqlite3/vfs"
)

func init() {
	vfs.Register("litestream", liteVFS{})
}

var (
	liteMtx sync.RWMutex
	// +checklocks:memoryMtx
	liteDBs = map[string]liteDB{}
)

type liteDB struct {
	client litestream.ReplicaClient
	logger *slog.Logger
}

// NewReplica creates a readonly replica from a client.
func NewReplica(name string, client litestream.ReplicaClient, logger *slog.Logger) {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	logger = logger.With("name", name)

	liteMtx.Lock()
	defer liteMtx.Unlock()
	liteDBs[name] = liteDB{
		client: client,
		logger: logger,
	}
}

// RemoveReplica removes a replica by name.
func RemoveReplica(name string) {
	liteMtx.Lock()
	defer liteMtx.Unlock()
	delete(liteDBs, name)
}
