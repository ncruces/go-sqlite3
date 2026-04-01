package testcfg

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TempDir(t testing.TB) string {
	t.Helper()

	if runtime.GOOS != "js" {
		return t.TempDir()
	}

	dir, err := os.MkdirTemp(".", sanitize(t.Name())+"-")
	if err != nil {
		t.Fatal(err)
	}
	dir = trimDotPrefix(dir)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}

func sanitize(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case 'a' <= r && r <= 'z':
			return r
		case 'A' <= r && r <= 'Z':
			return r
		case '0' <= r && r <= '9':
			return r
		case r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, name)
}

func TempFilename(t testing.TB, ext string) string {
	t.Helper()

	if runtime.GOOS != "js" {
		return filepath.Join(t.TempDir(), "test"+ext)
	}

	name := sanitize(t.Name()) + "-" + strconv.FormatInt(time.Now().UnixNano(), 36) + ext
	t.Cleanup(func() {
		_ = os.Remove(name)
	})
	return name
}

func trimDotPrefix(path string) string {
	prefix := "." + string(os.PathSeparator)
	return strings.TrimPrefix(path, prefix)
}
