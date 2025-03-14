package osutil

import (
	"io/fs"
	"os"
	. "syscall"
	"unsafe"
)

// OpenFile behaves the same as [os.OpenFile],
// except on Windows it sets [syscall.FILE_SHARE_DELETE].
//
// See: https://go.dev/issue/32088#issuecomment-502850674
func OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	if name == "" {
		return nil, &os.PathError{Op: "open", Path: name, Err: ENOENT}
	}
	r, err := syscallOpen(name, flag|O_CLOEXEC, uint32(perm.Perm()))
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: name, Err: err}
	}
	return os.NewFile(uintptr(r), name), nil
}

// syscallOpen is a copy of [syscall.Open]
// that uses [syscall.FILE_SHARE_DELETE],
// and supports [syscall.FILE_FLAG_OVERLAPPED].
//
// https://go.dev/src/syscall/syscall_windows.go
func syscallOpen(name string, flag int, perm uint32) (fd Handle, err error) {
	if len(name) == 0 {
		return InvalidHandle, ERROR_FILE_NOT_FOUND
	}
	namep, err := UTF16PtrFromString(name)
	if err != nil {
		return InvalidHandle, err
	}
	var access uint32
	switch flag & (O_RDONLY | O_WRONLY | O_RDWR) {
	case O_RDONLY:
		access = GENERIC_READ
	case O_WRONLY:
		access = GENERIC_WRITE
	case O_RDWR:
		access = GENERIC_READ | GENERIC_WRITE
	}
	if flag&O_CREAT != 0 {
		access |= GENERIC_WRITE
	}
	if flag&O_APPEND != 0 {
		// Remove GENERIC_WRITE unless O_TRUNC is set, in which case we need it to truncate the file.
		// We can't just remove FILE_WRITE_DATA because GENERIC_WRITE without FILE_WRITE_DATA
		// starts appending at the beginning of the file rather than at the end.
		if flag&O_TRUNC == 0 {
			access &^= GENERIC_WRITE
		}
		// Set all access rights granted by GENERIC_WRITE except for FILE_WRITE_DATA.
		access |= FILE_APPEND_DATA | FILE_WRITE_ATTRIBUTES | _FILE_WRITE_EA | STANDARD_RIGHTS_WRITE | SYNCHRONIZE
	}
	sharemode := uint32(FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE)
	var sa *SecurityAttributes
	if flag&O_CLOEXEC == 0 {
		sa = makeInheritSa()
	}
	// We don't use CREATE_ALWAYS, because when opening a file with
	// FILE_ATTRIBUTE_READONLY these will replace an existing file
	// with a new, read-only one. See https://go.dev/issue/38225.
	//
	// Instead, we ftruncate the file after opening when O_TRUNC is set.
	var createmode uint32
	switch {
	case flag&(O_CREAT|O_EXCL) == (O_CREAT | O_EXCL):
		createmode = CREATE_NEW
	case flag&O_CREAT == O_CREAT:
		createmode = OPEN_ALWAYS
	default:
		createmode = OPEN_EXISTING
	}
	var attrs uint32 = FILE_ATTRIBUTE_NORMAL
	if perm&S_IWRITE == 0 {
		attrs = FILE_ATTRIBUTE_READONLY
	}
	if flag&O_WRONLY == 0 && flag&O_RDWR == 0 {
		// We might be opening or creating a directory.
		// CreateFile requires FILE_FLAG_BACKUP_SEMANTICS
		// to work with directories.
		attrs |= FILE_FLAG_BACKUP_SEMANTICS
	}
	if flag&O_SYNC != 0 {
		const _FILE_FLAG_WRITE_THROUGH = 0x80000000
		attrs |= _FILE_FLAG_WRITE_THROUGH
	}
	if flag&O_NONBLOCK != 0 {
		attrs |= FILE_FLAG_OVERLAPPED
	}
	h, err := CreateFile(namep, access, sharemode, sa, createmode, attrs, 0)
	if h == InvalidHandle {
		if err == ERROR_ACCESS_DENIED && (flag&O_WRONLY != 0 || flag&O_RDWR != 0) {
			// We should return EISDIR when we are trying to open a directory with write access.
			fa, e1 := GetFileAttributes(namep)
			if e1 == nil && fa&FILE_ATTRIBUTE_DIRECTORY != 0 {
				err = EISDIR
			}
		}
		return h, err
	}
	// Ignore O_TRUNC if the file has just been created.
	if flag&O_TRUNC == O_TRUNC &&
		(createmode == OPEN_EXISTING || (createmode == OPEN_ALWAYS /*&& err == ERROR_ALREADY_EXISTS*/)) {
		err = Ftruncate(h, 0)
		if err != nil {
			CloseHandle(h)
			return InvalidHandle, err
		}
	}
	return h, nil
}

func makeInheritSa() *SecurityAttributes {
	var sa SecurityAttributes
	sa.Length = uint32(unsafe.Sizeof(sa))
	sa.InheritHandle = 1
	return &sa
}

const _FILE_WRITE_EA = 0x00000010
