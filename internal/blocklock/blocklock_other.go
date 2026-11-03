//go:build !linux || sqlite3_flock || sqlite3_dotlk

package blocklock

import "github.com/ncruces/go-sqlite3/internal/errutil"

// Other platforms only lock within the process.

func osSupported(fd uintptr) bool { return false }

func osReadLock(fd uintptr, start, end int64) error { panic(errutil.AssertErr()) }

func osWriteLock(fd uintptr, start, end int64) error { panic(errutil.AssertErr()) }

func osUnlock(fd uintptr, start, end int64) { panic(errutil.AssertErr()) }
