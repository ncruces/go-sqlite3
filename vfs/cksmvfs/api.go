package cksmvfs

import "github.com/ncruces/go-sqlite3/vfs"

func init() {
	Register("cksmvfs", vfs.Find(""))
}

func Register(name string, base vfs.VFS) {
	vfs.Register(name, &cksmVFS{
		VFS: base,
	})
}
