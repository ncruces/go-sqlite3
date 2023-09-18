//go:build sqlite3_nolock && unix && !(linux || darwin || freebsd || illumos)

package vfs

import (
	"os"
	"time"
)

func osUnlock(file *os.File, start, len int64) _ErrorCode {
	return _OK
}

func osLock(file *os.File, typ int16, start, len int64, timeout time.Duration, def _ErrorCode) _ErrorCode {
	return _OK
}

func osReadLock(file *os.File, start, len int64, timeout time.Duration) _ErrorCode {
	return _OK
}

func osWriteLock(file *os.File, start, len int64, timeout time.Duration) _ErrorCode {
	return _OK
}

func osCheckLock(file *os.File, start, len int64) (bool, _ErrorCode) {
	return false, _OK
}
