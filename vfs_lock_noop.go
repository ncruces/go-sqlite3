package sqlite3

type vfsNoopLocker struct {
	state vfsLockState
}

var _ vfsLocker = &vfsNoopLocker{}

func (l *vfsNoopLocker) LockState() vfsLockState {
	return l.state
}

func (l *vfsNoopLocker) LockShared() xErrorCode {
	if assert && !(l.state == _NO_LOCK) {
		panic(assertErr + " [wz9dcw]")
	}
	l.state = _SHARED_LOCK
	return _OK
}

func (l *vfsNoopLocker) LockReserved() xErrorCode {
	if assert && !(l.state == _SHARED_LOCK) {
		panic(assertErr + " [m9hcil]")
	}
	l.state = _RESERVED_LOCK
	return _OK
}

func (l *vfsNoopLocker) LockPending() xErrorCode {
	if assert && !(l.state == _SHARED_LOCK || l.state == _RESERVED_LOCK) {
		panic(assertErr + " [wx8nk2]")
	}
	l.state = _PENDING_LOCK
	return _OK
}

func (l *vfsNoopLocker) LockExclusive() xErrorCode {
	if assert && !(l.state == _PENDING_LOCK) {
		panic(assertErr + " [84nbax]")
	}
	l.state = _EXCLUSIVE_LOCK
	return _OK
}

func (l *vfsNoopLocker) DowngradeLock() xErrorCode {
	if assert && !(l.state > _SHARED_LOCK) {
		panic(assertErr + " [je31i3]")
	}
	l.state = _SHARED_LOCK
	return _OK
}

func (l *vfsNoopLocker) Unlock() xErrorCode {
	if assert && !(l.state > _NO_LOCK) {
		panic(assertErr + " [m6e9w5]")
	}
	l.state = _NO_LOCK
	return _OK
}

func (l *vfsNoopLocker) CheckReservedLock() (bool, xErrorCode) {
	if l.state >= _RESERVED_LOCK {
		return true, _OK
	}
	return false, _OK
}
