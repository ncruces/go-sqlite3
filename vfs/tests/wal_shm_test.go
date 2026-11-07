package tests

import (
	"bytes"
	"context"
	"database/sql"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/vfs"
)

// TestWALShmRollbackCorruption reproduces deterministic stale reads and
// silent B-tree corruption in copy-based WAL shared memory (sqlite3_dotlk)
// when uncommitted pager cache spills are rolled back.
func TestWALShmRollbackCorruption(t *testing.T) {
	if !vfs.SupportsSharedMemory {
		t.Skip("skipping without shared memory")
	}
	t.Run("mxFrameZero", func(t *testing.T) {
		ctx := t.Context()
		dsn := (&url.URL{
			Scheme:   "file",
			OmitHost: true,
			Path:     filepath.Join(t.TempDir(), "rollback_zero.db"),
			RawQuery: url.Values{
				"_pragma": {
					"journal_mode(wal)",
					"synchronous(normal)",
				},
			}.Encode(),
		}).String()

		db, err := driver.Open(dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		db.SetMaxOpenConns(4)

		c1, err := db.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer c1.Close()

		c2, err := db.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer c2.Close()

		if _, err := c1.ExecContext(ctx, `
			CREATE TABLE spill(id INTEGER PRIMARY KEY, pad BLOB);
			CREATE TABLE target(id INTEGER PRIMARY KEY, tag TEXT NOT NULL, pad BLOB);
			CREATE INDEX idx_target_tag ON target(tag);
		`); err != nil {
			t.Fatal(err)
		}
		for i := 1; i <= 30; i++ {
			if _, err := c1.ExecContext(ctx, `INSERT INTO spill(id, pad) VALUES(?, ?)`,
				i, bytes.Repeat([]byte{'S'}, 3000)); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := c1.ExecContext(ctx, `INSERT INTO target(id, tag, pad) VALUES(1, 'v0', ?)`,
			bytes.Repeat([]byte{'T'}, 2000)); err != nil {
			t.Fatal(err)
		}

		// Reset WAL so mxFrame = 0.
		var cnt int
		if err := c2.QueryRowContext(ctx, `SELECT count(*) FROM target`).Scan(&cnt); err != nil {
			t.Fatal(err)
		}
		if _, err := c1.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			t.Fatal(err)
		}

		// Epoch 1: update unindexed column so Frame 1 = target table page.
		if _, err := c1.ExecContext(ctx, `UPDATE target SET pad = ? WHERE id = 1`,
			bytes.Repeat([]byte{'U'}, 2000)); err != nil {
			t.Fatal(err)
		}
		if _, err := c2.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			t.Fatal(err)
		}
		// Sync c1 header (mxFrame = 0); shared[0] and c1.shadow[0] still hold Epoch 1's hash entry.
		if err := c1.QueryRowContext(ctx, `SELECT count(*) FROM target`).Scan(&cnt); err != nil {
			t.Fatal(err)
		}

		// Spill uncommitted frames on c1 and ROLLBACK.
		// At mxFrame = 0, appending frame 1 zeroes c1's private hash table,
		// and walCleanupHash on ROLLBACK is a no-op because mxFrame == 0.
		if _, err := c1.ExecContext(ctx, `PRAGMA cache_size = 2`); err != nil {
			t.Fatal(err)
		}
		if _, err := c1.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
			t.Fatal(err)
		}
		for i := 1; i <= 20; i++ {
			if _, err := c1.ExecContext(ctx, `UPDATE spill SET pad = ? WHERE id = ?`,
				bytes.Repeat([]byte{'X'}, 3000), i); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := c1.ExecContext(ctx, `ROLLBACK`); err != nil {
			t.Fatal(err)
		}

		// Epoch 2: c2 updates indexed column tag = 'v2':
		// Frame 1 = target table page (matches c1.shadow[0] -> skipped by c1.shmAcquire),
		// Frame 2 = idx_target_tag index page (copied by c1.shmAcquire).
		if _, err := c2.ExecContext(ctx, `UPDATE target SET tag = 'v2' WHERE id = 1`); err != nil {
			t.Fatal(err)
		}

		// c1 reads stale 'v0', and updating aReadMark[1] causes c1.shmRelease
		// to overwrite shared[0]'s hash table without holding WAL_WRITE_LOCK.
		var got string
		if err := readInTx(ctx, c1, `SELECT tag FROM target WHERE id = 1`, &got); err != nil {
			t.Fatal(err)
		}
		if got != "v2" {
			t.Errorf("got stale tag %q after commit, want %q", got, "v2")
		}

		// Checkpoint uses poisoned shared[0], skipping Frame 1 while backfilling Frame 2.
		if _, err := c2.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			t.Fatal(err)
		}
		c1.Close()
		c2.Close()
		db.Close()

		checkIntegrity(t, dsn)
	})

	t.Run("crossPageBoundary", func(t *testing.T) {
		ctx := t.Context()
		dsn := (&url.URL{
			Scheme:   "file",
			OmitHost: true,
			Path:     filepath.Join(t.TempDir(), "rollback_page1.db"),
			RawQuery: url.Values{
				"_pragma": {
					"page_size(512)",
					"journal_mode(wal)",
					"synchronous(normal)",
					"wal_autocheckpoint(0)",
				},
			}.Encode(),
		}).String()

		db, err := driver.Open(dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		db.SetMaxOpenConns(4)

		c1, err := db.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer c1.Close()

		c2, err := db.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer c2.Close()

		if _, err := c1.ExecContext(ctx, `
			CREATE TABLE spill(id INTEGER PRIMARY KEY, pad BLOB);
			CREATE TABLE target(id INTEGER PRIMARY KEY, tag TEXT NOT NULL, pad BLOB);
			CREATE INDEX idx_target_tag ON target(tag);
			WITH RECURSIVE cnt(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM cnt WHERE x < 4200)
			INSERT INTO spill(id, pad) SELECT x, zeroblob(400) FROM cnt;
		`); err != nil {
			t.Fatal(err)
		}
		if _, err := c1.ExecContext(ctx, `INSERT INTO target(id, tag, pad) VALUES(1, 'v0', ?)`,
			bytes.Repeat([]byte{'T'}, 300)); err != nil {
			t.Fatal(err)
		}

		var cnt int
		if err := c2.QueryRowContext(ctx, `SELECT count(*) FROM target`).Scan(&cnt); err != nil {
			t.Fatal(err)
		}
		if _, err := c1.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			t.Fatal(err)
		}

		// WAL-index page 0 holds 4062 frames; frame 4063 is slot 0 on WAL-index page 1.
		if _, err := c1.ExecContext(ctx, `UPDATE spill SET pad = randomblob(400) WHERE id <= 4062`); err != nil {
			t.Fatal(err)
		}
		if _, err := c1.ExecContext(ctx, `UPDATE target SET pad = ? WHERE id = 1`,
			bytes.Repeat([]byte{'U'}, 300)); err != nil {
			t.Fatal(err)
		}

		// Reset WAL and commit 1 frame so mxFrame == 1 (> 0) while page 1 retains Epoch 1's frame 4063.
		if _, err := c2.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			t.Fatal(err)
		}
		if _, err := c2.ExecContext(ctx, `UPDATE spill SET pad = randomblob(400) WHERE id = 1`); err != nil {
			t.Fatal(err)
		}
		if err := c1.QueryRowContext(ctx, `SELECT count(*) FROM target`).Scan(&cnt); err != nil {
			t.Fatal(err)
		}

		// With mxFrame == 1 (> 0), spill > 4063 distinct pages on c1 so WAL-index page 1 is zeroed/dirtied,
		// then ROLLBACK. walCleanupHash only cleans page 0 (where mxFrame == 1 lives), leaving page 1 dirty.
		if _, err := c1.ExecContext(ctx, `PRAGMA cache_size = 10`); err != nil {
			t.Fatal(err)
		}
		if _, err := c1.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
			t.Fatal(err)
		}
		if _, err := c1.ExecContext(ctx, `UPDATE spill SET pad = randomblob(400) WHERE id <= 4150`); err != nil {
			t.Fatal(err)
		}
		if _, err := c1.ExecContext(ctx, `ROLLBACK`); err != nil {
			t.Fatal(err)
		}

		// Epoch 2: advance c2 from mxFrame == 1 to 4062, then commit target.tag = 'v2' at frames 4063..4064.
		if _, err := c2.ExecContext(ctx, `UPDATE spill SET pad = randomblob(400) WHERE id BETWEEN 2 AND 4062`); err != nil {
			t.Fatal(err)
		}
		if _, err := c2.ExecContext(ctx, `UPDATE target SET tag = 'v2' WHERE id = 1`); err != nil {
			t.Fatal(err)
		}

		var got string
		if err := readInTx(ctx, c1, `SELECT tag FROM target WHERE id = 1`, &got); err != nil {
			t.Fatal(err)
		}
		if got != "v2" {
			t.Errorf("got stale tag %q after commit, want %q", got, "v2")
		}

		if _, err := c2.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			t.Fatal(err)
		}
		c1.Close()
		c2.Close()
		db.Close()

		checkIntegrity(t, dsn)
	})

	t.Run("restartThenRollback", func(t *testing.T) {
		ctx := t.Context()
		dsn := (&url.URL{
			Scheme:   "file",
			OmitHost: true,
			Path:     filepath.Join(t.TempDir(), "rollback_restart.db"),
			RawQuery: url.Values{
				"_pragma": {
					"journal_mode(wal)",
					"synchronous(normal)",
				},
			}.Encode(),
		}).String()

		db, err := driver.Open(dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		db.SetMaxOpenConns(4)

		c1, err := db.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer c1.Close()

		c2, err := db.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer c2.Close()

		if _, err := c1.ExecContext(ctx, `
			CREATE TABLE spill(id INTEGER PRIMARY KEY, pad BLOB);
			CREATE TABLE target(id INTEGER PRIMARY KEY, tag TEXT NOT NULL, pad BLOB);
			CREATE INDEX idx_target_tag ON target(tag);
		`); err != nil {
			t.Fatal(err)
		}
		for i := 1; i <= 30; i++ {
			if _, err := c1.ExecContext(ctx, `INSERT INTO spill(id, pad) VALUES(?, ?)`,
				i, bytes.Repeat([]byte{'S'}, 3000)); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := c1.ExecContext(ctx, `INSERT INTO target(id, tag, pad) VALUES(1, 'v0', ?)`,
			bytes.Repeat([]byte{'T'}, 2000)); err != nil {
			t.Fatal(err)
		}

		var cnt int
		if err := c2.QueryRowContext(ctx, `SELECT count(*) FROM target`).Scan(&cnt); err != nil {
			t.Fatal(err)
		}
		if _, err := c1.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			t.Fatal(err)
		}

		// Commit Frame 1 = target.pad, then PASSIVE checkpoint so nBackfill == mxFrame == 1.
		if _, err := c1.ExecContext(ctx, `UPDATE target SET pad = ? WHERE id = 1`,
			bytes.Repeat([]byte{'U'}, 2000)); err != nil {
			t.Fatal(err)
		}
		if _, err := c2.ExecContext(ctx, `PRAGMA wal_checkpoint(PASSIVE)`); err != nil {
			t.Fatal(err)
		}

		// Spilling on c1 triggers walRestartLog (walRestartHdr publishes mxFrame = 0 mid-tx),
		// then spills uncommitted frames 1..N into c1.privat[0], and rolls back.
		if _, err := c1.ExecContext(ctx, `PRAGMA cache_size = 2`); err != nil {
			t.Fatal(err)
		}
		if _, err := c1.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
			t.Fatal(err)
		}
		for i := 1; i <= 20; i++ {
			if _, err := c1.ExecContext(ctx, `UPDATE spill SET pad = ? WHERE id = ?`,
				bytes.Repeat([]byte{'X'}, 3000), i); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := c1.ExecContext(ctx, `ROLLBACK`); err != nil {
			t.Fatal(err)
		}

		// c2 commits tag = 'v2' at Frames 1..2; c1 reads stale 'v0' and poisons shared[0].
		if _, err := c2.ExecContext(ctx, `UPDATE target SET tag = 'v2' WHERE id = 1`); err != nil {
			t.Fatal(err)
		}

		var got string
		if err := readInTx(ctx, c1, `SELECT tag FROM target WHERE id = 1`, &got); err != nil {
			t.Fatal(err)
		}
		if got != "v2" {
			t.Errorf("got stale tag %q after commit, want %q", got, "v2")
		}

		if _, err := c2.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			t.Fatal(err)
		}
		c1.Close()
		c2.Close()
		db.Close()

		checkIntegrity(t, dsn)
	})
}

func readInTx(ctx context.Context, conn *sql.Conn, query string, dest *string) error {
	tx, err := conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	return tx.QueryRowContext(ctx, query).Scan(dest)
}

func checkIntegrity(t *testing.T, dsn string) {
	t.Helper()
	db, err := driver.Open(dsn)
	if err != nil {
		t.Fatalf("cold reopen: %v", err)
	}
	defer db.Close()

	rows, err := db.Query(`PRAGMA integrity_check`)
	if err != nil {
		t.Fatalf("integrity_check query: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var msg string
		if err := rows.Scan(&msg); err != nil {
			t.Fatal(err)
		}
		if msg != "ok" {
			t.Errorf("integrity_check: %s", msg)
		}
	}
}
