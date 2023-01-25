package sqlite3

import (
	"os"
	"sync"

	"github.com/tetratelabs/wazero/api"
)

type vfsOpenFile struct {
	file *os.File
	info os.FileInfo
	nref int

	shared int
	vfsLocker
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
		file: file,
		info: info,
		nref: 1,

		vfsLocker: &vfsFileLocker{file, _NO_LOCK},
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

func (p vfsFilePtr) ID() uint32 {
	if p.ptr == 0 {
		panic(nilErr)
	}
	id, ok := p.Memory().ReadUint32Le(p.ptr + ptrlen)
	if !ok {
		panic(rangeErr)
	}
	return id
}

func (p vfsFilePtr) Lock() vfsLockState {
	if p.ptr == 0 {
		panic(nilErr)
	}
	lk, ok := p.Memory().ReadUint32Le(p.ptr + 2*ptrlen)
	if !ok {
		panic(rangeErr)
	}
	return vfsLockState(lk)
}

func (p vfsFilePtr) SetID(id uint32) vfsFilePtr {
	if p.ptr == 0 {
		panic(nilErr)
	}
	if ok := p.Memory().WriteUint32Le(p.ptr+ptrlen, id); !ok {
		panic(rangeErr)
	}
	return p
}

func (p vfsFilePtr) SetLock(lock vfsLockState) vfsFilePtr {
	if p.ptr == 0 {
		panic(nilErr)
	}
	if ok := p.Memory().WriteUint32Le(p.ptr+2*ptrlen, uint32(lock)); !ok {
		panic(rangeErr)
	}
	return p
}
