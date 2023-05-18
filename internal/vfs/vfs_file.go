package vfs

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/sqlite3vfs"
	"github.com/tetratelabs/wazero/api"
)

const vfsOS vfsOSAPI = false

type vfsOSAPI bool

func (vfsOSAPI) FullPathname(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	fi, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return path, nil
		}
		return "", err
	}
	if fi.Mode()&fs.ModeSymlink != 0 {
		err = _OK_SYMLINK
	}
	return path, err
}

func (vfsOSAPI) Delete(path string, syncDir bool) error {
	err := os.Remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return _IOERR_DELETE_NOENT
	}
	if err != nil {
		return err
	}
	if runtime.GOOS != "windows" && syncDir {
		f, err := os.Open(filepath.Dir(path))
		if err != nil {
			return _OK
		}
		defer f.Close()
		err = osSync(f, false, false)
		if err != nil {
			return _IOERR_DIR_FSYNC
		}
	}
	return nil
}

func (vfsOSAPI) Access(name string, flags sqlite3vfs.AccessFlag) (bool, error) {
	err := osAccess(name, flags)
	if flags == _ACCESS_EXISTS {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
	} else {
		if errors.Is(err, fs.ErrPermission) {
			return false, nil
		}
	}
	return err == nil, err
}

func (vfsOSAPI) Open(name string, flags sqlite3vfs.OpenFlag) (sqlite3vfs.File, sqlite3vfs.OpenFlag, error) {
	var oflags int
	if flags&_OPEN_EXCLUSIVE != 0 {
		oflags |= os.O_EXCL
	}
	if flags&_OPEN_CREATE != 0 {
		oflags |= os.O_CREATE
	}
	if flags&_OPEN_READONLY != 0 {
		oflags |= os.O_RDONLY
	}
	if flags&_OPEN_READWRITE != 0 {
		oflags |= os.O_RDWR
	}

	var err error
	var f *os.File
	if name == "" {
		f, err = os.CreateTemp("", "*.db")
	} else {
		f, err = osOpenFile(name, oflags, 0666)
	}
	if err != nil {
		return nil, flags, err
	}

	if flags&_OPEN_DELETEONCLOSE != 0 {
		os.Remove(f.Name())
	}

	file := vfsFile{
		File:     f,
		psow:     true,
		readOnly: flags&_OPEN_READONLY != 0,
		syncDir: runtime.GOOS != "windows" &&
			flags&(_OPEN_CREATE) != 0 &&
			flags&(_OPEN_MAIN_JOURNAL|_OPEN_SUPER_JOURNAL|_OPEN_WAL) != 0,
	}
	return &file, flags, nil
}

type vfsFile struct {
	*os.File
	lockTimeout time.Duration
	lock        _LockLevel
	psow        bool
	syncDir     bool
	readOnly    bool
}

func vfsFileNew(vfs *vfsState, file sqlite3vfs.File) uint32 {
	// Find an empty slot.
	for id, f := range vfs.files {
		if f == nil {
			vfs.files[id] = file
			return uint32(id)
		}
	}

	// Add a new slot.
	vfs.files = append(vfs.files, file)
	return uint32(len(vfs.files) - 1)
}

func vfsFileRegister(ctx context.Context, mod api.Module, pFile uint32, file sqlite3vfs.File) {
	id := vfsFileNew(ctx.Value(vfsKey{}).(*vfsState), file)
	util.WriteUint32(mod, pFile+4, id)
}

func vfsFileGet(ctx context.Context, mod api.Module, pFile uint32) sqlite3vfs.File {
	vfs := ctx.Value(vfsKey{}).(*vfsState)
	id := util.ReadUint32(mod, pFile+4)
	return vfs.files[id]
}

func vfsFileClose(ctx context.Context, mod api.Module, pFile uint32) error {
	vfs := ctx.Value(vfsKey{}).(*vfsState)
	id := util.ReadUint32(mod, pFile+4)
	file := vfs.files[id]
	vfs.files[id] = nil
	return file.Close()
}

func (f *vfsFile) Sync(flags sqlite3vfs.SyncFlag) error {
	dataonly := (flags & _SYNC_DATAONLY) != 0
	fullsync := (flags & 0x0f) == _SYNC_FULL

	err := osSync(f.File, fullsync, dataonly)
	if err != nil {
		return err
	}
	if runtime.GOOS != "windows" && f.syncDir {
		f.syncDir = false
		d, err := os.Open(filepath.Dir(f.File.Name()))
		if err != nil {
			return nil
		}
		defer d.Close()
		err = osSync(d, false, false)
		if err != nil {
			return _IOERR_DIR_FSYNC
		}
	}
	return nil
}

func (f *vfsFile) FileSize() (int64, error) {
	return f.Seek(0, io.SeekEnd)
}

func (*vfsFile) SectorSize() int {
	return _DEFAULT_SECTOR_SIZE
}
