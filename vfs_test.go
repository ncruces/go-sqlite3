package sqlite3

import (
	"bytes"
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/ncruces/julianday"
)

func Test_vfsLocaltime(t *testing.T) {
	memory := make(MockMemory, 128)
	module := &MockModule{&memory}

	memory.Write(0, []byte("zero"))

	rc := vfsLocaltime(context.TODO(), module, 0, 4)
	if rc != 0 {
		t.Fatal("returned", rc)
	}

	epoch := time.Unix(0, 0)
	if z, _ := memory.Read(0, 4); !bytes.Equal(z, []byte("zero")) {
		t.Fatal("overwrote zero address")
	}
	if s, _ := memory.ReadUint32Le(4 + 0*4); int(s) != epoch.Second() {
		t.Fatal("wrong second")
	}
	if m, _ := memory.ReadUint32Le(4 + 1*4); int(m) != epoch.Minute() {
		t.Fatal("wrong minute")
	}
	if h, _ := memory.ReadUint32Le(4 + 2*4); int(h) != epoch.Hour() {
		t.Fatal("wrong hour")
	}
	if d, _ := memory.ReadUint32Le(4 + 3*4); int(d) != epoch.Day() {
		t.Fatal("wrong day")
	}
	if m, _ := memory.ReadUint32Le(4 + 4*4); time.Month(1+m) != epoch.Month() {
		t.Fatal("wrong month")
	}
	if y, _ := memory.ReadUint32Le(4 + 5*4); 1900+int(y) != epoch.Year() {
		t.Fatal("wrong year")
	}
	if w, _ := memory.ReadUint32Le(4 + 6*4); time.Weekday(w) != epoch.Weekday() {
		t.Fatal("wrong weekday")
	}
	if d, _ := memory.ReadUint32Le(4 + 7*4); int(d) != epoch.YearDay()-1 {
		t.Fatal("wrong yearday")
	}
}

func Test_vfsRandomness(t *testing.T) {
	memory := make(MockMemory, 128)
	module := &MockModule{&memory}

	memory.Write(0, []byte("zero"))

	rand.Seed(0)
	rc := vfsRandomness(context.TODO(), module, 0, 16, 4)
	if rc != 16 {
		t.Fatal("returned", rc)
	}

	if z, _ := memory.Read(0, 4); !bytes.Equal(z, []byte("zero")) {
		t.Fatal("overwrote zero address")
	}

	var want [16]byte
	rand.Seed(0)
	rand.Read(want[:])

	if got, _ := memory.Read(4, 16); !bytes.Equal(got, want[:]) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func Test_vfsSleep(t *testing.T) {
	start := time.Now()

	rc := vfsSleep(context.TODO(), 0, 123456)
	if rc != 0 {
		t.Fatal("returned", rc)
	}

	want := 123456 * time.Microsecond
	if got := time.Since(start); got < want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func Test_vfsCurrentTime(t *testing.T) {
	memory := make(MockMemory, 128)
	module := &MockModule{&memory}

	memory.Write(0, []byte("zero"))

	now := time.Now()
	rc := vfsCurrentTime(context.TODO(), module, 0, 4)
	if rc != 0 {
		t.Fatal("returned", rc)
	}

	if z, _ := memory.Read(0, 4); !bytes.Equal(z, []byte("zero")) {
		t.Fatal("overwrote zero address")
	}
	want := julianday.Float(now)
	if got, _ := memory.ReadFloat64Le(4); float32(got) != float32(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func Test_vfsCurrentTime64(t *testing.T) {
	memory := make(MockMemory, 128)
	module := &MockModule{&memory}

	memory.Write(0, []byte("zero"))

	now := time.Now()
	time.Sleep(time.Millisecond)
	rc := vfsCurrentTime64(context.TODO(), module, 0, 4)
	if rc != 0 {
		t.Fatal("returned", rc)
	}

	if z, _ := memory.Read(0, 4); !bytes.Equal(z, []byte("zero")) {
		t.Fatal("overwrote zero address")
	}
	day, nsec := julianday.Date(now)
	want := day*86_400_000 + nsec/1_000_000
	if got, _ := memory.ReadUint64Le(4); int64(got)-want > 100 {
		t.Fatalf("got %v, want %v", got, want)
	}
}
