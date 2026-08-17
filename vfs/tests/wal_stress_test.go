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

func TestWALConcurrentWriters(t *testing.T) {
	if testing.Short() || os.Getenv("SQLITE3_TEST_WAL_STRESS") == "" {
		t.Skip("skipping without SQLITE3_TEST_WAL_STRESS")
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
			for i := range iters {
				if err := doTx(ctx, db, gid, i, blob); err != nil {
					if isBackpressure(err) {
						continue
					}
					return fmt.Errorf("worker %d iter %d: %w", gid, i, err)
				}
				if i%ckptEvery != 0 {
					continue
				}
				if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
					if isBackpressure(err) {
						continue
					}
					return fmt.Errorf("checkpoint: %w", err)
				}
			}
			return nil
		})
	}
	err = eg.Wait()
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if err != nil {
		return fmt.Errorf("worker error: %w", err)
	}

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

func isBackpressure(err error) bool {
	var code sqlite3.ErrorCode
	return errors.As(err, &code) && (false ||
		code == sqlite3.BUSY ||
		code == sqlite3.IOERR ||
		code == sqlite3.FULL ||
		code == sqlite3.PROTOCOL)
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
