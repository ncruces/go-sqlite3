//go:build js && wasm

package vfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFullPathnameJSDoesNotRequireReadlink(t *testing.T) {
	t.Parallel()

	name := filepath.Join(t.TempDir(), "test.db")

	got, err := (vfsOS{}).FullPathname(name)
	if err != nil {
		t.Fatal(err)
	}

	want, err := filepath.Abs(name)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFullPathnameJSKeepsRelativePathsRelative(t *testing.T) {
	t.Parallel()

	dir, err := os.MkdirTemp(".", "fullpath-js-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	name := filepath.Join(dir, "test.db")

	got, err := (vfsOS{}).FullPathname(name)
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Clean(name)
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
