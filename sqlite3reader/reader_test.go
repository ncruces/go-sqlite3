package sqlite3reader

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewSizeReaderAt(t *testing.T) {
	f, err := os.Create(filepath.Join(t.TempDir(), "abc.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	n, err := NewSizeReaderAt(f).Size()
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("got %d", n)
	}

	reader := strings.NewReader("abc")

	n, err = NewSizeReaderAt(reader).Size()
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("got %d", n)
	}

	n, err = NewSizeReaderAt(readlener{reader, reader.Len()}).Size()
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("got %d", n)
	}

	n, err = NewSizeReaderAt(readsizer{reader, reader.Size()}).Size()
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("got %d", n)
	}

	n, err = NewSizeReaderAt(readseeker{reader, reader}).Size()
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("got %d", n)
	}

	_, err = NewSizeReaderAt(readerat{reader}).Size()
	if err == nil {
		t.Error("want error")
	}
}

type readlener struct {
	io.ReaderAt
	len int
}

func (l readlener) Len() int { return l.len }

type readsizer struct {
	io.ReaderAt
	size int64
}

func (l readsizer) Size() (int64, error) { return l.size, nil }

type readseeker struct {
	io.ReaderAt
	io.Seeker
}

type readerat struct {
	io.ReaderAt
}
