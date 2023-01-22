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
	vfsMutex     sync.Mutex
	vfsOpenFiles []*vfsOpenFile
)

func vfsGetFileID(file *os.File) (uint32, error) {
	fi, err := file.Stat()
	if err != nil {
		return 0, err
	}

	vfsMutex.Lock()
	defer vfsMutex.Unlock()

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

func vfsReleaseFile(mod api.Module, pFile uint32) error {
	id, ok := mod.Memory().ReadUint32Le(pFile + ptrSize)
	if !ok {
		panic(rangeErr)
	}

	vfsMutex.Lock()
	defer vfsMutex.Unlock()

	of := vfsOpenFiles[id]
	if of.nref--; of.nref > 0 {
		return nil
	}
	err := of.file.Close()
	vfsOpenFiles[id] = nil
	return err
}

func vfsGetOSFile(mod api.Module, pFile uint32) *os.File {
	id, ok := mod.Memory().ReadUint32Le(pFile + ptrSize)
	if !ok {
		panic(rangeErr)
	}
	return vfsOpenFiles[id].file
}

func vfsGetFileData(mod api.Module, pFile uint32) (id, lock uint32) {
	var ok bool
	if id, ok = mod.Memory().ReadUint32Le(pFile + ptrSize); !ok {
		panic(rangeErr)
	}
	if lock, ok = mod.Memory().ReadUint32Le(pFile + 2*ptrSize); !ok {
		panic(rangeErr)
	}
	return
}

func vfsSetFileData(mod api.Module, pFile, id, lock uint32) {
	if ok := mod.Memory().WriteUint32Le(pFile+ptrSize, id); !ok {
		panic(rangeErr)
	}
	if ok := mod.Memory().WriteUint32Le(pFile+2*ptrSize, lock); !ok {
		panic(rangeErr)
	}
}

const (
	_NO_LOCK        = 0
	_SHARED_LOCK    = 1
	_RESERVED_LOCK  = 2
	_PENDING_LOCK   = 3
	_EXCLUSIVE_LOCK = 4
)
