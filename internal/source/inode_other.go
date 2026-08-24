//go:build !unix

package source

import (
	"hash/fnv"
	"os"
)

func fileInode(info os.FileInfo) (uint64, error) {
	// Fallback: approximate identity with name+modtime+size.
	h := fnv.New64a()
	_, _ = h.Write([]byte(info.Name()))
	_, _ = h.Write([]byte(info.ModTime().String()))
	return h.Sum64() ^ uint64(info.Size()), nil
}
