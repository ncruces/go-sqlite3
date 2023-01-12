package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"

	_ "embed"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed sqlite3.wasm
var binary []byte

func main() {
	var ctx = context.Background()

	wasm := wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, wasm)

	compiled, err := wasm.CompileModule(ctx, binary)
	if err != nil {
		panic(err)
	}

	cfg := wazero.NewModuleConfig()
	module, err := wasm.InstantiateModule(ctx, compiled, cfg)
	if err != nil {
		panic(err)
	}

	var db sqlite = sqlite{
		memory:       module.Memory(),
		_malloc:      module.ExportedFunction("malloc"),
		_free:        module.ExportedFunction("free"),
		_errmsg:      module.ExportedFunction("sqlite3_errmsg"),
		_open:        module.ExportedFunction("sqlite3_open_v2"),
		_close:       module.ExportedFunction("sqlite3_close"),
		_prepare:     module.ExportedFunction("sqlite3_prepare_v2"),
		_exec:        module.ExportedFunction("sqlite3_exec"),
		_step:        module.ExportedFunction("sqlite3_step"),
		_columnText:  module.ExportedFunction("sqlite3_column_text"),
		_columnInt:   module.ExportedFunction("sqlite3_column_int64"),
		_columnFloat: module.ExportedFunction("sqlite3_column_double"),
	}

	log.Println(err, db.memory.Size())

	err = db.Open(":memory:", SQLITE_OPEN_READWRITE|SQLITE_OPEN_CREATE, "")
	defer db.Close()

	log.Println(err, db.memory.Size())
}

type sqlite struct {
	handle       uint32
	memory       api.Memory
	_malloc      api.Function
	_free        api.Function
	_errmsg      api.Function
	_open        api.Function
	_close       api.Function
	_prepare     api.Function
	_exec        api.Function
	_step        api.Function
	_columnInt   api.Function
	_columnText  api.Function
	_columnFloat api.Function
}

func (s *sqlite) Errmsg() error {
	r, err := s._errmsg.Call(context.TODO(), uint64(s.handle))
	if err != nil {
		return err
	}
	return errors.New(s.getString(r[0]))
}

func (s *sqlite) Open(name string, flags uint64, vfs string) error {
	namePtr := s.newString(name)
	defer s.free(namePtr)

	handlePtr := s.newPtr()
	defer s.free(handlePtr)

	var vfsPtr uint32
	if vfs != "" {
		vfsPtr = s.newString(vfs)
		defer s.free(vfsPtr)
	}

	r, err := s._open.Call(context.TODO(), uint64(namePtr), uint64(handlePtr), flags, uint64(vfsPtr))
	if err != nil {
		return err
	}

	s.handle, _ = s.memory.ReadUint32Le(handlePtr)

	if r[0] != SQLITE_OK {
		err := fmt.Errorf("sqlite error (%d): %s", r[0], s.Errmsg())
		_ = s.Close()
		return err
	}
	return nil
}

func (s *sqlite) Close() error {
	r, err := s._close.Call(context.TODO(), uint64(s.handle))
	if err != nil {
		return err
	}

	if r[0] != SQLITE_OK {
		return fmt.Errorf("sqlite error (%d): %s", r[0], s.Errmsg())
	}
	return nil
}

func (s *sqlite) free(ptr uint32) {
	_, err := s._free.Call(context.TODO(), uint64(ptr))
	if err != nil {
		panic(err)
	}
}

func (s *sqlite) newPtr() uint32 {
	r, err := s._malloc.Call(context.TODO(), 4)
	if err != nil {
		panic(err)
	}
	return uint32(r[0])
}

func (s *sqlite) newString(str string) uint32 {
	r, err := s._malloc.Call(context.TODO(), uint64(len(str)+1))
	if err != nil {
		panic(err)
	}

	ptr := uint32(r[0])
	if ok := s.memory.Write(ptr, []byte(str)); !ok {
		panic("failed init string")
	}
	if ok := s.memory.WriteByte(ptr+uint32(len(str)), 0); !ok {
		panic("failed init string")
	}
	return ptr
}

func (s *sqlite) getString(ptr uint64) string {
	buf, ok := s.memory.Read(uint32(ptr), 64)
	if !ok {
		panic("failed read string")
	}
	if i := bytes.IndexByte(buf, 0); i < 0 {
		panic("failed read string")
	} else {
		return string(buf[:i])
	}
}

const (
	SQLITE_OK   = 0
	SQLITE_ROW  = 100
	SQLITE_DONE = 101

	SQLITE_OPEN_READWRITE = 0x00000002
	SQLITE_OPEN_CREATE    = 0x00000004
)
