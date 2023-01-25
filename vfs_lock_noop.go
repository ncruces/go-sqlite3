package sqlite3

const assert = true

type vfsNoopLocker struct {
	state vfsLockState
}

var _ vfsLocker = &vfsNoopLocker{}

func (l *vfsNoopLocker) LockState() vfsLockState {
	return l.state
}

func (l *vfsNoopLocker) LockShared() uint32 {
	if assert && !(l.state == _NO_LOCK) {
		panic(assertErr + " [wz9dcw]")
	}
	l.state = _SHARED_LOCK
	return _OK
}

func (l *vfsNoopLocker) LockReserved() uint32 {
	if assert && !(l.state == _SHARED_LOCK) {
		panic(assertErr + " [m9hcil]")
	}
	l.state = _RESERVED_LOCK
	return _OK
}

func (l *vfsNoopLocker) LockPending() uint32 {
	if assert && !(l.state == _SHARED_LOCK || l.state == _RESERVED_LOCK) {
		panic(assertErr + " [wx8nk2]")
	}
	l.state = _PENDING_LOCK
	return _OK
}

func (l *vfsNoopLocker) LockExclusive() uint32 {
	if assert && !(l.state == _PENDING_LOCK) {
		panic(assertErr + " [84nbax]")
	}
	l.state = _EXCLUSIVE_LOCK
	return _OK
}

func (l *vfsNoopLocker) DowngradeLock() uint32 {
	if assert && !(l.state > _SHARED_LOCK) {
		panic(assertErr + " [je31i3]")
	}
	l.state = _SHARED_LOCK
	return _OK
}

func (l *vfsNoopLocker) Unlock() uint32 {
	if assert && !(l.state > _NO_LOCK) {
		panic(assertErr + " [m6e9w5]")
	}
	l.state = _NO_LOCK
	return _OK
}

func (l *vfsNoopLocker) CheckReservedLock() (bool, uint32) {
	if l.state >= _RESERVED_LOCK {
		return true, _OK
	}
	return false, _OK
}
