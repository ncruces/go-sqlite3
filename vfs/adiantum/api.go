package adiantum

import "github.com/ncruces/go-sqlite3/vfs"

func init() {
	Register("adiantum", vfs.Find(""), nil)
}

func Register(name string, base vfs.VFS, cipher HBSHCreator) {
	if cipher == nil {
		cipher = adiantumCreator{}
	}
	vfs.Register("adiantum", &hbshVFS{
		VFS:  base,
		hbsh: cipher,
	})
}
