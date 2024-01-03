package ioutil

import (
	"strings"
	"testing"
)

func TestNewSeekingReaderAt(t *testing.T) {
	reader := NewSeekingReaderAt(strings.NewReader("abc"))
	defer reader.Close()

	n, err := reader.Size()
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("got %d", n)
	}

	var buf [3]byte
	r, err := reader.ReadAt(buf[:], 0)
	if err != nil {
		t.Fatal(err)
	}
	if r != 3 {
		t.Errorf("got %d", r)
	}
}
