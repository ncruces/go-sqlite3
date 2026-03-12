package testfs

import (
	"bufio"
	"io/fs"
)

var (
	FS     fs.FS
	Stdout *bufio.Writer
	Stderr *bufio.Writer
)
