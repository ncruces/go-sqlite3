package util

import (
	"io/fs"
	"os"
)

type OSFS struct{}

func (OSFS) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (OSFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (OSFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

func (OSFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}
