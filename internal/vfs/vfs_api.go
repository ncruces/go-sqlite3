package vfs

import (
	"reflect"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/sqlite3vfs"
	"github.com/tetratelabs/wazero/api"
)

func vfsAPIGet(mod api.Module, pVfs uint32) sqlite3vfs.VFS {
	if pVfs == 0 {
		return nil
	}
	name := util.ReadString(mod, util.ReadUint32(mod, pVfs+16), _MAX_STRING)
	return sqlite3vfs.Find(name)
}

func vfsAPIErrorCode(err error, def _ErrorCode) _ErrorCode {
	if err == nil {
		return _OK
	}
	switch v := reflect.ValueOf(err); v.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return _ErrorCode(v.Uint())
	}
	return def
}
