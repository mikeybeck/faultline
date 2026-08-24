//go:build unix

package source

import (
	"fmt"
	"os"
	"syscall"
)

func fileInode(info os.FileInfo) (uint64, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("unsupported stat type")
	}
	return stat.Ino, nil
}
