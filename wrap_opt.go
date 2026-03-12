package sqlite3

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
)

// These libc functions are only used by tests,
// and need to be further locked down.

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
	if _, err := os.Stdout.WriteString(s); err != nil {
		return -1
	}
	if _, err := os.Stdout.WriteString("\n"); err != nil {
		return -1
	}
	return 0
}

func (e env) Xfclose(h int32) int32 {
	var err error
	switch h {
	case 0:
		//
	case 1:
		err = os.Stdout.Sync()
	case 2:
		err = os.Stderr.Sync()
	default:
		err = e.DelHandle(ptr_t(h))
	}
	if err != nil {
		return -1
	}
	return 0
}

func (e env) Xfopen(path, mode int32) int32 {
	p := e.ReadString(ptr_t(path), _MAX_NAME)
	m := e.ReadString(ptr_t(mode), _MAX_NAME)

	var flag int
	if len(m) > 0 {
		switch m[0] {
		case 'r':
			flag = os.O_RDONLY
		case 'w':
			flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		case 'a':
			flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
		}
		if m[len(m)-1] == '+' {
			flag &^= os.O_RDONLY | os.O_WRONLY
			flag |= os.O_RDWR
		}
	}

	f, err := os.OpenFile(string(p), flag, 0666)
	if err != nil {
		return 0
	}
	return int32(e.AddHandle(f))
}

func (e env) getf(h int32) *os.File {
	switch h {
	case 0:
		return os.Stdin
	case 1:
		return os.Stdout
	case 2:
		return os.Stderr
	default:
		return e.GetHandle(ptr_t(h)).(*os.File)
	}
}

func (e env) Xfflush(h int32) int32 {
	f := e.getf(h)
	if err := f.Sync(); err != nil {
		return -1
	}
	return 0
}

func (e env) Xfputc(c, h int32) int32 {
	f := e.getf(h)
	var b [1]byte
	b[0] = byte(c)
	if _, err := f.Write(b[:]); err != nil {
		return -1
	}
	return 0
}

func (e env) Xfread(ptr, sz, cnt, h int32) int32 {
	f := e.getf(h)
	b := e.Buf[ptr:][:sz*cnt]
	n, _ := f.Read(b)
	return int32(n / int(sz))
}

func (e env) Xfwrite(ptr, sz, cnt, h int32) int32 {
	f := e.getf(h)
	b := e.Buf[ptr:][:sz*cnt]
	n, _ := f.Write(b)
	return int32(n / int(sz))
}

func (e env) Xftell(h int32) int32 {
	f := e.getf(h)
	if n, err := f.Seek(0, io.SeekEnd); err != nil {
		return -1
	} else {
		return int32(n)
	}
}

func (e env) Xfseek(h, offset, whence int32) int32 {
	f := e.getf(h)
	if _, err := f.Seek(int64(offset), int(whence)); err != nil {
		return -1
	}
	return 0
}
