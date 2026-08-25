//go:build unix

package source

import (
	"fmt"
	"os"
	"syscall"
)

func fileIdentity(f *os.File) (uint64, error) {
	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return fileInode(info)
}

func pathIdentity(path string) (uint64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return fileInode(info)
}

func fileInode(info os.FileInfo) (uint64, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("unsupported stat type")
	}
	return stat.Ino, nil
}
