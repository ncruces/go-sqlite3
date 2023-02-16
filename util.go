package sqlite3

import "sync"

type chanMutex chan struct{}

var _ sync.Locker = newChanMutex()

func newChanMutex() chanMutex {
	return make(chanMutex, 1)
}

func (mtx chanMutex) Lock() {
	mtx <- struct{}{}
}

func (mtx chanMutex) Unlock() {
	<-mtx
}
