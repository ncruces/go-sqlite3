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
	lock int
}

var (
	vfsOpenFiles    []*vfsOpenFile
	vfsOpenFilesMtx sync.Mutex
)

func vfsGetOpenFileID(file *os.File) (uint32, error) {
	fi, err := file.Stat()
	if err != nil {
		return 0, err
	}

	vfsOpenFilesMtx.Lock()
	defer vfsOpenFilesMtx.Unlock()

	// Reuse an already opened file.
	for id, of := range vfsOpenFiles {
		if of == nil {
			continue
		}
		if os.SameFile(fi, of.info) {
			if err := file.Close(); err != nil {
				return 0, err
			}
			of.nref++
			return uint32(id), nil
		}
	}

	openFile := vfsOpenFile{
		file: file,
		info: fi,
		nref: 1,
	}

	// Find an empty slot.
	for id, of := range vfsOpenFiles {
		if of == nil {
			vfsOpenFiles[id] = &openFile
			return uint32(id), nil
		}
	}

	// Add a new slot.
	id := len(vfsOpenFiles)
	vfsOpenFiles = append(vfsOpenFiles, &openFile)
	return uint32(id), nil
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
	id, ok := p.Memory().ReadUint32Le(p.ptr + ptrSize)
	if !ok {
		panic(rangeErr)
	}
	return id
}

func (p vfsFilePtr) Lock() uint32 {
	lk, ok := p.Memory().ReadUint32Le(p.ptr + 2*ptrSize)
	if !ok {
		panic(rangeErr)
	}
	return lk
}

func (p vfsFilePtr) SetID(id uint32) vfsFilePtr {
	if ok := p.Memory().WriteUint32Le(p.ptr+ptrSize, id); !ok {
		panic(rangeErr)
	}
	return p
}

func (p vfsFilePtr) SetLock(lock uint32) vfsFilePtr {
	if ok := p.Memory().WriteUint32Le(p.ptr+2*ptrSize, lock); !ok {
		panic(rangeErr)
	}
	return p
}

const (
	_NO_LOCK        = 0
	_SHARED_LOCK    = 1
	_RESERVED_LOCK  = 2
	_PENDING_LOCK   = 3
	_EXCLUSIVE_LOCK = 4
)
