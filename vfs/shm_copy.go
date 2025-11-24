//go:build (windows && (386 || arm || amd64 || arm64 || riscv64 || ppc64le || loong64)) || sqlite3_dotlk

package vfs

import (
	"unsafe"

	"github.com/ncruces/go-sqlite3/internal/util"
)

const (
	_WALINDEX_HDR_SIZE = 136
	_WALINDEX_PGSZ     = 32768
)

// This seems a safe way of keeping the WAL-index in sync.
//
// The WAL-index file starts with a header,
// and the index doesn't meaningfully change if the header doesn't change.
//
// The header starts with two 48 byte, checksummed, copies of the same information,
// which are accessed independently between memory barriers.
// The checkpoint information that follows uses 4 byte aligned words.
//
// Finally, we have the WAL-index hash tables,
// which are only modified holding the exclusive WAL_WRITE_LOCK.
//
// Since all the data is either redundant+checksummed,
// 4 byte aligned, or modified under an exclusive lock,
// the copies below should correctly keep memory in sync.
//
// https://sqlite.org/walformat.html#the_wal_index_file_format

func (s *vfsShm) shmAcquire(errp *error) {
	if errp != nil && *errp != nil {
		return
	}
	if len(s.ptrs) == 0 {
		return
	}
	if !shmCopyHeader(
		util.View(s.mod, s.ptrs[0], _WALINDEX_HDR_SIZE),
		s.shadow[0][:],
		s.shared[0][:]) {
		return
	}

	skip := _WALINDEX_HDR_SIZE
	for id := range s.ptrs {
		shmCopyTables(
			util.View(s.mod, s.ptrs[id], _WALINDEX_PGSZ)[skip:],
			s.shadow[id][skip:],
			s.shared[id][skip:])
		skip = 0
	}
}

func (s *vfsShm) shmRelease() {
	if len(s.ptrs) == 0 {
		return
	}
	if !shmCopyHeader(
		s.shared[0][:],
		s.shadow[0][:],
		util.View(s.mod, s.ptrs[0], _WALINDEX_HDR_SIZE)) {
		return
	}

	skip := _WALINDEX_HDR_SIZE
	for id := range s.ptrs {
		shmCopyTables(
			s.shared[id][skip:],
			s.shadow[id][skip:],
			util.View(s.mod, s.ptrs[id], _WALINDEX_PGSZ)[skip:])
		skip = 0
	}
}

func (s *vfsShm) shmBarrier() {
	s.Lock()
	s.shmAcquire(nil)
	s.shmRelease()
	s.Unlock()
}

func shmCopyTables(v1, v2, v3 []byte) {
	if string(v2) != string(v3) {
		copy(v1, v3)
		copy(v2, v3)
	}
}

func shmCopyHeader(s1, s2, s3 []byte) (ret bool) {
	// First copy of the WAL Index Information.
	if string(s2[:48]) != string(s3[:48]) {
		copy(s1, s3[:48])
		copy(s2, s3[:48])
		ret = true
	}
	// Second copy of the WAL Index Information.
	if string(s2[48:][:48]) != string(s3[48:][:48]) {
		copy(s1[48:], s3[48:][:48])
		copy(s2[48:], s3[48:][:48])
		ret = true
	}
	// Checkpoint Information and Locks.
	i1 := shmCheckpointInfo(s1)
	i2 := shmCheckpointInfo(s2)
	for i, i3 := range shmCheckpointInfo(s3) {
		if i2[i] != i3 {
			i1[i] = i3
			i2[i] = i3
			ret = true
		}
	}
	return
}

func shmCheckpointInfo(s []byte) *[10]uint32 {
	p := (*uint32)(unsafe.Pointer(&s[96]))
	return (*[10]uint32)(unsafe.Slice(p, 10))
}
