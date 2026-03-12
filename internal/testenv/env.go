package testenv

import (
	"bytes"
	"io/fs"
	"sync"
	"testing"

	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
)

var (
	FS     fs.FS
	TB     testing.TB
	Exit   func(int32)
	System func(*sqlite3_wrap.Wrapper, int32) int32

	buf []byte
	mtx sync.Mutex
)

func WriteByte(c byte) error {
	TB.Helper()
	mtx.Lock()
	defer mtx.Unlock()

	if c == '\n' {
		TB.Logf("%s", buf)
		buf = buf[:0]
	} else {
		buf = append(buf, c)
	}
	return nil
}

func Write(p []byte) (n int, err error) {
	TB.Helper()
	mtx.Lock()
	defer mtx.Unlock()

	buf = append(buf, p...)
	for {
		before, after, found := bytes.Cut(buf, []byte("\n"))
		if !found {
			return len(p), nil
		}
		TB.Logf("%s", before)
		buf = after
	}
}

func WriteString(s string) (n int, err error) {
	return Write([]byte(s))
}
