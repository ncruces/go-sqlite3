package sqlite3

import (
	"os"
	"sync"

	"github.com/tetratelabs/wazero/api"
)

type vfsOpenFile struct {
	file   *os.File
	info   os.FileInfo
	nref   int
	locker vfsFileLocker
}

var (
	vfsOpenFiles    []*vfsOpenFile
	vfsOpenFilesMtx sync.Mutex
)

func vfsGetOpenFileID(file *os.File, info os.FileInfo) uint32 {
	vfsOpenFilesMtx.Lock()
	defer vfsOpenFilesMtx.Unlock()

	// Reuse an already opened file.
	for id, of := range vfsOpenFiles {
		if of == nil {
			continue
		}
		if os.SameFile(info, of.info) {
			of.nref++
			_ = file.Close()
			return uint32(id)
		}
	}

	of := &vfsOpenFile{
		file:   file,
		info:   info,
		nref:   1,
		locker: vfsFileLocker{file: file},
	}

	// Find an empty slot.
	for id, ptr := range vfsOpenFiles {
		if ptr == nil {
			vfsOpenFiles[id] = of
			return uint32(id)
		}
	}

	// Add a new slot.
	id := len(vfsOpenFiles)
	vfsOpenFiles = append(vfsOpenFiles, of)
	return uint32(id)
}

func vfsReleaseOpenFile(id uint32) error {
	vfsOpenFilesMtx.Lock()
	defer vfsOpenFilesMtx.Unlock()

	of := vfsOpenFiles[id]
	if of.nref--; of.nref > 0 {
		return nil
	}
	err := of.file.Close()
	vfsOpenFiles[id] = nil
	return err
}

type vfsFilePtr struct {
	api.Module
	ptr uint32
}

func (p vfsFilePtr) OSFile() *os.File {
	id := p.ID()
	vfsOpenFilesMtx.Lock()
	defer vfsOpenFilesMtx.Unlock()
	return vfsOpenFiles[id].file
}

func (p vfsFilePtr) Locker() *vfsFileLocker {
	id := p.ID()
	vfsOpenFilesMtx.Lock()
	defer vfsOpenFilesMtx.Unlock()
	return &vfsOpenFiles[id].locker
}

func (p vfsFilePtr) ID() uint32 {
	return memory{p}.readUint32(p.ptr + ptrlen)
}

func (p vfsFilePtr) Lock() vfsLockState {
	return vfsLockState(memory{p}.readUint32(p.ptr + 2*ptrlen))
}

func (p vfsFilePtr) SetID(id uint32) vfsFilePtr {
	memory{p}.writeUint32(p.ptr+ptrlen, id)
	return p
}

func (p vfsFilePtr) SetLock(lock vfsLockState) vfsFilePtr {
	memory{p}.writeUint32(p.ptr+2*ptrlen, uint32(lock))
	return p
}
