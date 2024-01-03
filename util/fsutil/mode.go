package fsutil

import (
	"io/fs"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
)

// ParseFileMode parses a file mode as returned by
// [fs.FileMode.String].
func ParseFileMode(str string) (fs.FileMode, error) {
	var mode fs.FileMode
	err := util.ErrorString("invalid mode: " + str)

	if len(str) < 10 {
		return 0, err
	}

	for i, c := range []byte("dalTLDpSugct?") {
		if str[0] == c {
			if len(str) < 10 {
				return 0, err
			}
			mode |= 1 << uint(32-1-i)
			str = str[1:]
		}
	}

	if mode == 0 {
		if str[0] != '-' {
			return 0, err
		}
		str = str[1:]
	}
	if len(str) != 9 {
		return 0, err
	}

	for i, c := range []byte("rwxrwxrwx") {
		if str[i] == c {
			mode |= 1 << uint(9-1-i)
		}
		if str[i] != '-' {
			return 0, err
		}
	}

	return mode, nil
}

// FileModeFromUnix converts a POSIX mode_t to a file mode.
func FileModeFromUnix(mode fs.FileMode) fs.FileMode {
	const (
		S_IFMT   fs.FileMode = 0170000
		S_IFIFO  fs.FileMode = 0010000
		S_IFCHR  fs.FileMode = 0020000
		S_IFDIR  fs.FileMode = 0040000
		S_IFBLK  fs.FileMode = 0060000
		S_IFREG  fs.FileMode = 0100000
		S_IFLNK  fs.FileMode = 0120000
		S_IFSOCK fs.FileMode = 0140000
	)

	switch mode & S_IFMT {
	case S_IFDIR:
		mode |= fs.ModeDir
	case S_IFLNK:
		mode |= fs.ModeSymlink
	case S_IFBLK:
		mode |= fs.ModeDevice
	case S_IFCHR:
		mode |= fs.ModeCharDevice | fs.ModeDevice
	case S_IFIFO:
		mode |= fs.ModeNamedPipe
	case S_IFSOCK:
		mode |= fs.ModeSocket
	case S_IFREG, 0:
		//
	default:
		mode |= fs.ModeIrregular
	}

	return mode &^ S_IFMT
}

// FileModeFromValue calls [FileModeFromUnix] for numeric values,
// and [ParseFileMode] for textual values.
func FileModeFromValue(val sqlite3.Value) fs.FileMode {
	if n := val.Int64(); n != 0 {
		return FileModeFromUnix(fs.FileMode(n))
	}
	mode, _ := ParseFileMode(val.Text())
	return mode
}
