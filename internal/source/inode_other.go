//go:build !unix && !windows

package source

import (
	"hash/fnv"
	"os"
)

func fileIdentity(f *os.File) (uint64, error) {
	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return nameIdentity(info), nil
}

func pathIdentity(path string) (uint64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return nameIdentity(info), nil
}

func nameIdentity(info os.FileInfo) uint64 {
	// Name only: size and modtime change as a live log grows, and must not
	// look like rotation.
	h := fnv.New64a()
	_, _ = h.Write([]byte(info.Name()))
	return h.Sum64()
}
