package sqlite3

import (
	"errors"
	"io/fs"
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

// OpenFile is a simplified copy of [os.openFileNolog]
// that uses syscall.FILE_SHARE_DELETE.
// https://go.dev/src/os/file_windows.go
//
// See: https://go.dev/issue/32088
func (vfsOSMethods) OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	if name == "" {
		return nil, &os.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
	}
	r, e := syscallOpen(name, flag, uint32(perm.Perm()))
	if e != nil {
		return nil, &os.PathError{Op: "open", Path: name, Err: e}
	}
	return os.NewFile(uintptr(r), name), nil
}

func (vfsOSMethods) Sync(file *os.File) error {
	return file.Sync()
}

func (vfsOSMethods) Access(path string, flags _AccessFlag) (bool, xErrorCode) {
	fi, err := os.Stat(path)

	switch {
	case flags == _ACCESS_EXISTS:
		switch {
		case err == nil:
			return true, _OK
		case errors.Is(err, fs.ErrNotExist):
			return false, _OK
		default:
			return false, IOERR_ACCESS
		}

	case err == nil:
		var want fs.FileMode = syscall.S_IRUSR
		if flags == _ACCESS_READWRITE {
			want |= syscall.S_IWUSR
		}
		if fi.IsDir() {
			want |= syscall.S_IXUSR
		}
		if fi.Mode()&want == want {
			return true, _OK
		} else {
			return false, _OK
		}

	case errors.Is(err, fs.ErrPermission):
		return false, _OK

	default:
		return false, IOERR_ACCESS
	}
}

func (vfsOSMethods) GetExclusiveLock(file *os.File) xErrorCode {
	// Release the SHARED lock.
	vfsOS.unlock(file, _SHARED_FIRST, _SHARED_SIZE)

	// Acquire the EXCLUSIVE lock.
	rc := vfsOS.writeLock(file, _SHARED_FIRST, _SHARED_SIZE)

	// Reacquire the SHARED lock.
	if rc != _OK {
		vfsOS.readLock(file, _SHARED_FIRST, _SHARED_SIZE)
	}
	return rc
}

func (vfsOSMethods) DowngradeLock(file *os.File, state vfsLockState) xErrorCode {
	if state >= _EXCLUSIVE_LOCK {
		// Release the SHARED lock.
		vfsOS.unlock(file, _SHARED_FIRST, _SHARED_SIZE)

		// Reacquire the SHARED lock.
		if rc := vfsOS.readLock(file, _SHARED_FIRST, _SHARED_SIZE); rc != _OK {
			// This should never happen.
			// We should always be able to reacquire the read lock.
			return IOERR_RDLOCK
		}
	}

	// Release the PENDING and RESERVED locks.
	if state >= _RESERVED_LOCK {
		vfsOS.unlock(file, _RESERVED_BYTE, 1)
	}
	if state >= _PENDING_LOCK {
		vfsOS.unlock(file, _PENDING_BYTE, 1)
	}
	return _OK
}

func (vfsOSMethods) ReleaseLock(file *os.File, state vfsLockState) xErrorCode {
	// Release all locks.
	if state >= _RESERVED_LOCK {
		vfsOS.unlock(file, _RESERVED_BYTE, 1)
	}
	if state >= _SHARED_LOCK {
		vfsOS.unlock(file, _SHARED_FIRST, _SHARED_SIZE)
	}
	if state >= _PENDING_LOCK {
		vfsOS.unlock(file, _PENDING_BYTE, 1)
	}
	return _OK
}

func (vfsOSMethods) unlock(file *os.File, start, len uint32) xErrorCode {
	err := windows.UnlockFileEx(windows.Handle(file.Fd()),
		0, len, 0, &windows.Overlapped{Offset: start})
	if err == windows.ERROR_NOT_LOCKED {
		return _OK
	}
	if err != nil {
		return IOERR_UNLOCK
	}
	return _OK
}

func (vfsOSMethods) readLock(file *os.File, start, len uint32) xErrorCode {
	return vfsOS.lockErrorCode(windows.LockFileEx(windows.Handle(file.Fd()),
		windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, len, 0, &windows.Overlapped{Offset: start}),
		IOERR_RDLOCK)
}

func (vfsOSMethods) writeLock(file *os.File, start, len uint32) xErrorCode {
	return vfsOS.lockErrorCode(windows.LockFileEx(windows.Handle(file.Fd()),
		windows.LOCKFILE_FAIL_IMMEDIATELY|windows.LOCKFILE_EXCLUSIVE_LOCK,
		0, len, 0, &windows.Overlapped{Offset: start}),
		IOERR_LOCK)
}

func (vfsOSMethods) checkLock(file *os.File, start, len uint32) (bool, xErrorCode) {
	rc := vfsOS.readLock(file, start, len)
	if rc == _OK {
		vfsOS.unlock(file, start, len)
	}
	return rc != _OK, _OK
}

func (vfsOSMethods) lockErrorCode(err error, def xErrorCode) xErrorCode {
	if err == nil {
		return _OK
	}
	if errno, ok := err.(syscall.Errno); ok {
		// https://devblogs.microsoft.com/oldnewthing/20140905-00/?p=63
		switch errno {
		case
			windows.ERROR_LOCK_VIOLATION,
			windows.ERROR_IO_PENDING:
			return xErrorCode(BUSY)
		}
	}
	return def
}

// syscallOpen is a simplified copy of [syscall.Open]
// that uses syscall.FILE_SHARE_DELETE.
// https://go.dev/src/syscall/syscall_windows.go
func syscallOpen(path string, mode int, perm uint32) (fd syscall.Handle, err error) {
	if len(path) == 0 {
		return syscall.InvalidHandle, syscall.ERROR_FILE_NOT_FOUND
	}
	pathp, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return syscall.InvalidHandle, err
	}
	var access uint32
	switch mode & (syscall.O_RDONLY | syscall.O_WRONLY | syscall.O_RDWR) {
	case syscall.O_RDONLY:
		access = syscall.GENERIC_READ
	case syscall.O_WRONLY:
		access = syscall.GENERIC_WRITE
	case syscall.O_RDWR:
		access = syscall.GENERIC_READ | syscall.GENERIC_WRITE
	}
	if mode&syscall.O_CREAT != 0 {
		access |= syscall.GENERIC_WRITE
	}
	if mode&syscall.O_APPEND != 0 {
		access &^= syscall.GENERIC_WRITE
		access |= syscall.FILE_APPEND_DATA
	}
	sharemode := uint32(syscall.FILE_SHARE_READ | syscall.FILE_SHARE_WRITE | syscall.FILE_SHARE_DELETE)
	var createmode uint32
	switch {
	case mode&(syscall.O_CREAT|syscall.O_EXCL) == (syscall.O_CREAT | syscall.O_EXCL):
		createmode = syscall.CREATE_NEW
	case mode&(syscall.O_CREAT|syscall.O_TRUNC) == (syscall.O_CREAT | syscall.O_TRUNC):
		createmode = syscall.CREATE_ALWAYS
	case mode&syscall.O_CREAT == syscall.O_CREAT:
		createmode = syscall.OPEN_ALWAYS
	case mode&syscall.O_TRUNC == syscall.O_TRUNC:
		createmode = syscall.TRUNCATE_EXISTING
	default:
		createmode = syscall.OPEN_EXISTING
	}
	var attrs uint32 = syscall.FILE_ATTRIBUTE_NORMAL
	if perm&syscall.S_IWRITE == 0 {
		attrs = syscall.FILE_ATTRIBUTE_READONLY
	}
	if createmode == syscall.OPEN_EXISTING && access == syscall.GENERIC_READ {
		// Necessary for opening directory handles.
		attrs |= syscall.FILE_FLAG_BACKUP_SEMANTICS
	}
	return syscall.CreateFile(pathp, access, sharemode, nil, createmode, attrs, 0)
}
