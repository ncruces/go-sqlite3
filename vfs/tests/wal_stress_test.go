// The sqlite3_dotlk build keeps the copy-on-lock-boundary scheme, which
// cannot fully eliminate wal-index staleness: under this load it reliably
// fails with SQLITE_PROTOCOL (a documented limitation, not corruption —
// the database stays intact). Only the real-file VFS paths are exercised.
//
//go:build !sqlite3_dotlk

package tests

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	sqlite3 "github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
)

// TestWALConcurrentWriters drives many concurrent connections writing to a
// single WAL database through one *sql.DB, aggressively checkpoints, then
// cold-reopens the file and runs PRAGMA integrity_check. It repeats that whole
// cycle until either the defect is observed or a time budget elapses.
//
// It is a regression test for Windows WAL-index corruption. On the
// copy-on-lock-boundary shared-memory scheme (the Windows default before the
// -shm was mapped into wasm memory) this corrupts the database on Windows: a
// checkpointer acting on a stale wal-index view backfills stale pages and
// truncates not-yet-backfilled frames. The cold-reopen integrity_check catches
// it, and SQLITE_PROTOCOL during the run is the other symptom of the same bug.
//
// On the fixed VFS — and on every platform whose VFS maps the -shm directly
// (unix, native) — every round stays clean. The test therefore goes red only
// on an unfixed Windows build and green everywhere else, which is exactly what
// a regression gate should do.
//
// BUSY is tolerated: under this many writers, busy_timeout exhaustion is
// ordinary backpressure, not the defect. It is excluded from sqlite3_dotlk
// builds (see the build constraint).
//
// It is opt-in — several minutes of heavy I/O — so it does not tax the normal
// test run: set SQLITE3_TEST_WAL_STRESS=1 to enable it (the wal-repro workflow
// does this). On an unfixed Windows build it reproduces in the first round,
// usually within a minute.
func TestWALConcurrentWriters(t *testing.T) {
	if testing.Short() || os.Getenv("SQLITE3_TEST_WAL_STRESS") == "" {
		t.Skip("opt-in heavy WAL concurrent-writers regression; " +
			"set SQLITE3_TEST_WAL_STRESS=1 to run")
	}

	const (
		workers   = 64
		iters     = 2000
		blobBytes = 8192
		ckptEvery = 25
		maxRounds = 3
		budget    = 5 * time.Minute
	)
	t.Logf("runner parallelism: NumCPU=%d GOMAXPROCS=%d", runtime.NumCPU(), runtime.GOMAXPROCS(0))

	blob := make([]byte, blobBytes)
	if _, err := rand.Read(blob); err != nil {
		t.Fatalf("rand: %v", err)
	}

	deadline := time.Now().Add(budget)
	round := 0
	for round < maxRounds {
		round++
		t.Logf("round %d", round)
		if err := walStressRound(t, workers, iters, ckptEvery, blob); err != nil {
			t.Fatalf("round %d: WAL corruption reproduced: %v", round, err)
		}
		if time.Now().After(deadline) {
			break
		}
	}
	t.Logf("%d concurrent-writer rounds clean (workers=%d iters=%d)", round, workers, iters)
}

// walStressRound runs one storm+checkpoint cycle in a fresh database and
// returns a non-empty description if the WAL defect was observed. It removes
// its own scratch directory so repeated rounds do not accumulate disk.
func walStressRound(t *testing.T, workers, iters, ckptEvery int, blob []byte) error {
	t.Helper()
	dir, err := os.MkdirTemp("", "wal_stress")
	if err != nil {
		t.Fatalf("mkdtemp: %v", err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "wal_stress.db")
	dsn := "file:" + path +
		"?_pragma=busy_timeout(10000)&_pragma=journal_mode(wal)" +
		"&_pragma=synchronous(normal)&_txlock=deferred"

	db, err := driver.Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(workers)
	if _, err := db.Exec(`
		CREATE TABLE parent(id INTEGER PRIMARY KEY AUTOINCREMENT, hash TEXT NOT NULL UNIQUE);
		CREATE TABLE child(id INTEGER PRIMARY KEY AUTOINCREMENT,
		    parent_id INTEGER NOT NULL REFERENCES parent(id) ON DELETE CASCADE, data BLOB NOT NULL);
		CREATE INDEX child_parent_idx ON child(parent_id);
	`); err != nil {
		db.Close()
		t.Fatalf("schema: %v", err)
	}

	wg, ctx := errgroup.WithContext(t.Context())
	for gid := range workers {
		wg.Go(func() error {
			for i := 0; i < iters; i++ {
				if err := doTx(ctx, db, gid, i, blob); err != nil {
					if errors.Is(err, sqlite3.BUSY) || errors.Is(err, sqlite3.PROTOCOL) {
						continue // ordinary backpressure, not the defect
					}
					return fmt.Errorf("worker %d iter %d: %w", gid, i, err)
				}
				if i%ckptEvery == 0 {
					if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil &&
						!errors.Is(err, sqlite3.BUSY) && !errors.Is(err, sqlite3.PROTOCOL) {
						return fmt.Errorf("checkpoint: %w", err)
					}
				}
			}
			return nil
		})
	}
	workErr := wg.Wait()
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// A non-BUSY worker error ("malformed", "file is not a database")
	// is a defect symptom, not infrastructure.
	if workErr != nil {
		return fmt.Errorf("worker error: %w", workErr)
	}

	// Cold reopen + integrity gate — the definitive assertion.
	db2, err := driver.Open(dsn)
	if err != nil {
		return fmt.Errorf("cold reopen failed: %w", err)
	}
	defer db2.Close()
	db2.SetMaxOpenConns(1)
	var first string
	if err := db2.QueryRow("PRAGMA integrity_check").Scan(&first); err != nil {
		return fmt.Errorf("integrity_check errored: %w", err)
	}
	if first != "ok" {
		return fmt.Errorf("integrity_check = %q", first)
	}
	return nil
}

func doTx(ctx context.Context, db *sql.DB, gid, i int, blob []byte) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	var pid int64
	if err := tx.QueryRowContext(ctx,
		"INSERT INTO parent(hash) VALUES(?) RETURNING id",
		fmt.Sprintf("%d-%d", gid, i)).Scan(&pid); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO child(parent_id, data) VALUES(?, ?)", pid, blob); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
