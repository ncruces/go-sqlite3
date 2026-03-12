package sqlite3

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3/internal/testfs"
)

// These libc functions are only used by tests,
// they should only work under testing.

func (e env) Xsystem(ptr int32) int32 {
	if !testing.Testing() {
		return -1
	}
	if ptr == 0 {
		return 0
	}

	s := e.ReadString(ptr_t(ptr), _MAX_NAME)

	args := strings.Split(s, " ")
	for i := range args {
		args[i] = strings.Trim(args[i], `"`)
	}
	if args[0] != "mptest" || args[len(args)-1] != "&" {
		return -1
	}
	args = args[:len(args)-1]

	go func() {
		ctx := WithMaxMemory(context.TODO(), 32*1024*1024)
		wrp, err := createWrapper(ctx)
		if err != nil {
			panic(err)
		}
		defer wrp.Close()

		argv := wrp.New(int64(ptrlen * len(args)))
		for i, a := range args {
			wrp.Write32(argv+ptr_t(i)*ptrlen, uint32(wrp.NewString(a)))
		}

		defer func() { recover() }()
		wrp.Xmain_mptest(int32(len(args)), int32(argv))
	}()
	return 0
}

func (e env) Xexit(c int32) {
	if c != 0 || !testing.Testing() {
		panic(fmt.Sprint("exit error: ", c))
	}
	runtime.Goexit()
}

func (e env) Xputs(ptr int32) int32 {
	s := e.ReadString(ptr_t(ptr), _MAX_NAME)
	testfs.Stdout.WriteString(s)
	testfs.Stdout.WriteByte('\n')
	return 0
}

func (e env) Xfclose(h int32) int32 {
	switch h {
	case 0:
		return -1
	case 1, 2:
		return e.Xfflush(h)
	}
	if e.DelHandle(ptr_t(h)) != nil {
		return -1
	}
	return 0
}

func (e env) Xfopen(path, mode int32) int32 {
	if testfs.FS == nil {
		return 0
	}

	p := e.ReadString(ptr_t(path), _MAX_NAME)
	f, err := testfs.FS.Open(p)
	if err != nil {
		return 0
	}
	return int32(e.AddHandle(f))
}

func (e env) Xfflush(h int32) int32 {
	w := getw(h)
	if w == nil {
		return -1
	}
	if w.Flush() != nil {
		return -1
	}
	return 0
}

func (e env) Xfputc(c, h int32) int32 {
	w := getw(h)
	if w == nil {
		return -1
	}
	if w.WriteByte(byte(c)) != nil {
		return -1
	}
	return 0
}

func (e env) Xfwrite(ptr, sz, cnt, h int32) int32 {
	w := getw(h)
	if w == nil {
		return 0
	}
	b := e.Buf[ptr:][:sz*cnt]
	n, _ := w.Write(b)
	return int32(n / int(sz))
}

func (e env) Xfread(ptr, sz, cnt, h int32) int32 {
	f := e.GetHandle(ptr_t(h)).(io.Reader)
	b := e.Buf[ptr:][:sz*cnt]
	n, _ := f.Read(b)
	return int32(n / int(sz))
}

func (e env) Xftell(h int32) int32 {
	f := e.GetHandle(ptr_t(h)).(io.Seeker)
	n, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return -1
	}
	return int32(n)
}

func (e env) Xfseek(h, offset, whence int32) int32 {
	f := e.GetHandle(ptr_t(h)).(io.Seeker)
	_, err := f.Seek(int64(offset), int(whence))
	if err != nil {
		return -1
	}
	return 0
}

func getw(h int32) *bufio.Writer {
	switch h {
	case 1:
		return testfs.Stdout
	case 2:
		return testfs.Stderr
	default:
		return nil
	}
}
