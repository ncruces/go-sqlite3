package tests

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	"golang.org/x/sync/errgroup"
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
// it.
//
// On the fixed VFS — and on every platform whose VFS maps the -shm directly
// (unix, native) — every round stays clean. The test therefore goes red only
// on an unfixed Windows build and green everywhere else, which is exactly what
// a regression gate should do.
//
// BUSY is tolerated: under this many writers, busy_timeout exhaustion is
// ordinary backpressure, not the defect. IOERR and FULL are tolerated too:
// this drives GBs of writes through a scratch directory, and a full or
// slow temp filesystem is an environment limit, not corruption — the
// cold-reopen integrity_check remains the definitive corruption gate. The
// defect surfaces as SQLITE_PROTOCOL, "malformed", or "not a database",
// none of which are BUSY/IOERR/FULL. Excluded from sqlite3_dotlk builds
// (see the build constraint).
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
	)
	t.Logf("runner parallelism: NumCPU=%d GOMAXPROCS=%d", runtime.NumCPU(), runtime.GOMAXPROCS(0))
	blob := make([]byte, blobBytes)
	rand.Read(blob)
	round := 0
	for round < maxRounds {
		round++
		t.Run("", func(t *testing.T) {
			if err := walStressRound(t, workers, iters, ckptEvery, blob); err != nil {
				t.Fatalf("round %d: WAL corruption reproduced: %v", round, err)
			}
		})
	}
	t.Logf("%d concurrent-writer rounds clean (workers=%d iters=%d)", round, workers, iters)
}

// walStressRound runs one storm+checkpoint cycle in a fresh database and
// returns a non-empty description if the WAL defect was observed. It removes
// its own scratch directory so repeated rounds do not accumulate disk.
func walStressRound(t *testing.T, workers, iters, ckptEvery int, blob []byte) error {
	dsn := (&url.URL{
		Scheme:   "file",
		OmitHost: true,
		Path:     filepath.Join(t.TempDir(), "wal_stress.db"),
		RawQuery: url.Values{
			"_txlock": {"deferred"},
			"_pragma": {
				"busy_timeout(10000)",
				"journal_mode(wal)",
				"synchronous(normal)",
			},
		}.Encode(),
	}).String()

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

	eg, ctx := errgroup.WithContext(t.Context())
	for gid := range workers {
		eg.Go(func() error {
			for i := 0; i < iters; i++ {
				if err := doTx(ctx, db, gid, i, blob); err != nil {
					if isBackpressure(err) {
						continue // backpressure or environment limit, not the defect
					}
					return fmt.Errorf("worker %d iter %d: %w", gid, i, err)
				}
				if i%ckptEvery == 0 {
					if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil &&
						!isBackpressure(err) {
						return fmt.Errorf("checkpoint: %w", err)
					}
				}
			}
			return nil
		})
	}
	err = eg.Wait()
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// A worker error that is not backpressure/environment (SQLITE_PROTOCOL
	// "locking protocol", "malformed", "file is not a database") is a defect
	// symptom, not infrastructure.
	if err != nil {
		return fmt.Errorf("worker error: %w", err)
	}

	// Cold reopen + integrity gate — the definitive assertion.
	db, err = driver.Open(dsn)
	if err != nil {
		return fmt.Errorf("cold reopen failed: %w", err)
	}
	defer db.Close()

	var first string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&first); err != nil {
		return fmt.Errorf("integrity_check errored: %w", err)
	}
	if first != "ok" {
		return fmt.Errorf("integrity_check: %q", first)
	}
	return nil
}

// isBackpressure reports whether err is ordinary write backpressure (BUSY)
// or an environment limit (a full or failing scratch filesystem: IOERR,
// FULL) rather than a symptom of the WAL defect. The defect surfaces as
// SQLITE_PROTOCOL, CORRUPT ("malformed"), or NOTADB ("not a database"),
// none of which match these codes; the cold-reopen integrity_check is the
// definitive corruption gate regardless.
func isBackpressure(err error) bool {
	return errors.Is(err, sqlite3.BUSY) ||
		errors.Is(err, sqlite3.IOERR) ||
		errors.Is(err, sqlite3.FULL)
}

func doTx(ctx context.Context, db *sql.DB, gid, i int, blob []byte) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var pid int64
	if err := tx.QueryRowContext(ctx,
		"INSERT INTO parent(hash) VALUES(?) RETURNING id",
		fmt.Sprintf("%d-%d", gid, i)).Scan(&pid); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO child(parent_id, data) VALUES(?, ?)", pid, blob); err != nil {
		return err
	}
	return tx.Commit()
}
