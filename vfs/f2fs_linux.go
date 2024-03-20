//go:build amd64 || arm64

package vfs

const (
	// https://gitlab.com/cznic/sqlite/-/blob/master/lib/sqlite_linux_amd64.go#L418-424
	// https://gitlab.com/cznic/sqlite/-/blob/master/lib/sqlite_linux_arm64.go#L418-424

	_F2FS_FEATURE_ATOMIC_WRITE     = 4
	_F2FS_IOCTL_MAGIC              = 245
	_F2FS_IOC_ABORT_VOLATILE_WRITE = 62725
	_F2FS_IOC_COMMIT_ATOMIC_WRITE  = 62722
	_F2FS_IOC_GET_FEATURES         = 2147546380
	_F2FS_IOC_START_ATOMIC_WRITE   = 62721
	_F2FS_IOC_START_VOLATILE_WRITE = 62723
)
