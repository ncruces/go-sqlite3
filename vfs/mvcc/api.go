// Package mvcc implements the "mvcc" SQLite VFS.
//
// The "mvcc" [vfs.VFS] allows the same in-memory database to be shared
// among multiple database connections in the same process,
// as long as the database name begins with "/".
//
// Importing package mvcc registers the VFS:
//
//	import _ "github.com/ncruces/go-sqlite3/vfs/mvcc"
package mvcc

import (
	"crypto/rand"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/ncruces/wbt"
)

func init() {
	vfs.Register("mvcc", mvccVFS{})
}

var (
	memoryMtx sync.Mutex
	// +checklocks:memoryMtx
	memoryDBs = map[string]*mvccDB{}
)

// Create creates a shared memory database,
// using a snapshot as its initial contents.
func Create(name string, snapshot Snapshot) {
	memoryMtx.Lock()
	defer memoryMtx.Unlock()

	memoryDBs[name] = &mvccDB{
		refs: 1,
		name: name,
		data: snapshot.Tree,
	}
}

// Delete deletes a shared memory database.
func Delete(name string) {
	name = getName(name)

	memoryMtx.Lock()
	defer memoryMtx.Unlock()
	delete(memoryDBs, name)
}

// Snapshot represents a database snapshot.
type Snapshot struct {
	*wbt.Tree[int64, string]
}

// NewSnapshot creates a snapshot from data.
func NewSnapshot(data string) Snapshot {
	var tree *wbt.Tree[int64, string]

	// Convert data from WAL/2 to rollback journal.
	if len(data) >= 20 && (false ||
		data[18] == 2 && data[19] == 2 ||
		data[18] == 3 && data[19] == 3) {
		tree = tree.
			Put(0, data[:18]).
			Put(18, "\001\001").
			Put(20, data[20:])
	} else if len(data) > 0 {
		tree = tree.Put(0, data)
	}

	return Snapshot{tree}
}

// TakeSnapshot takes a snapshot of a database.
// Name may be a URI filename.
func TakeSnapshot(name string) Snapshot {
	name = getName(name)

	memoryMtx.Lock()
	defer memoryMtx.Unlock()
	db := memoryDBs[name]
	if db == nil {
		return Snapshot{}
	}

	db.mtx.Lock()
	defer db.mtx.Unlock()
	return Snapshot{db.data}
}

// TestDB creates a shared database from a snapshot for the test to use.
// The database is automatically deleted when the test and all its subtests complete.
// Returns a URI filename appropriate to call Open with.
// Each subsequent call to TestDB returns a unique database.
//
//	func Test_something(t *testing.T) {
//		t.Parallel()
//		dsn := mvcc.TestDB(t, snapshot, url.Values{
//			"_pragma": {"busy_timeout(1000)"},
//		})
//
//		db, err := sql.Open("sqlite3", dsn)
//		if err != nil {
//			t.Fatal(err)
//		}
//		defer db.Close()
//
//		// ...
//	}
func TestDB(tb testing.TB, snapshot Snapshot, params ...url.Values) string {
	tb.Helper()

	name := fmt.Sprintf("%s_%s", tb.Name(), rand.Text())
	tb.Cleanup(func() { Delete(name) })
	Create(name, snapshot)

	p := url.Values{"vfs": {"mvcc"}}
	for _, v := range params {
		for k, v := range v {
			for _, v := range v {
				p.Add(k, v)
			}
		}
	}

	return (&url.URL{
		Scheme:   "file",
		OmitHost: true,
		Path:     "/" + name,
		RawQuery: p.Encode(),
	}).String()
}

func getName(dsn string) string {
	u, err := url.Parse(dsn)
	if err == nil &&
		u.Scheme == "file" &&
		strings.HasPrefix(u.Path, "/") &&
		u.Query().Get("vfs") == "mvcc" {
		return u.Path[1:]
	}
	return dsn
}
