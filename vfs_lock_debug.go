package sqlite3

const assert = true

type vfsDebugLocker struct {
	state uint32
}

var _ vfsLocker = &vfsDebugLocker{}

func (l *vfsDebugLocker) LockState() uint32 {
	return l.state
}

func (l *vfsDebugLocker) LockShared() uint32 {
	if assert && !(l.state == _NO_LOCK) {
		panic(assertErr + " [wz9dcw]")
	}
	l.state = _SHARED_LOCK
	return _OK
}

func (l *vfsDebugLocker) LockReserved() uint32 {
	if assert && !(l.state == _SHARED_LOCK) {
		panic(assertErr + " [m9hcil]")
	}
	l.state = _RESERVED_LOCK
	return _OK
}

func (l *vfsDebugLocker) LockPending() uint32 {
	if assert && !(l.state == _SHARED_LOCK || l.state == _RESERVED_LOCK) {
		panic(assertErr + " [wx8nk2]")
	}
	l.state = _PENDING_LOCK
	return _OK
}

func (l *vfsDebugLocker) LockExclusive() uint32 {
	if assert && !(l.state == _PENDING_LOCK) {
		panic(assertErr + " [84nbax]")
	}
	l.state = _EXCLUSIVE_LOCK
	return _OK
}

func (l *vfsDebugLocker) DowngradeLock() uint32 {
	if assert && !(l.state > _SHARED_LOCK) {
		panic(assertErr + " [je31i3]")
	}
	l.state = _SHARED_LOCK
	return _OK
}

func (l *vfsDebugLocker) ReleaseLock() uint32 {
	if assert && !(l.state > _NO_LOCK) {
		panic(assertErr + " [m6e9w5]")
	}
	l.state = _NO_LOCK
	return _OK
}
