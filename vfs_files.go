package sqlite3

import (
	"os"
	"sync"

	"github.com/tetratelabs/wazero/api"
)

var (
	vfsOpenFiles    []*os.File
	vfsOpenFilesMtx sync.Mutex
)

func vfsGetFileID(file *os.File) uint32 {
	vfsOpenFilesMtx.Lock()
	defer vfsOpenFilesMtx.Unlock()

	// Find an empty slot.
	for id, ptr := range vfsOpenFiles {
		if ptr == nil {
			vfsOpenFiles[id] = file
			return uint32(id)
		}
	}

	// Add a new slot.
	vfsOpenFiles = append(vfsOpenFiles, file)
	return uint32(len(vfsOpenFiles) - 1)
}

func vfsCloseFile(id uint32) error {
	vfsOpenFilesMtx.Lock()
	defer vfsOpenFilesMtx.Unlock()

	file := vfsOpenFiles[id]
	vfsOpenFiles[id] = nil
	return file.Close()
}

type vfsFilePtr struct {
	api.Module
	ptr uint32
}

func (p vfsFilePtr) OSFile() *os.File {
	id := p.ID()
	vfsOpenFilesMtx.Lock()
	defer vfsOpenFilesMtx.Unlock()
	return vfsOpenFiles[id]
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
