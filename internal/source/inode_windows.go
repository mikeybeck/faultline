//go:build windows

package source

import (
	"os"
	"syscall"
	"unsafe"
)

func fileIdentity(f *os.File) (uint64, error) {
	return fileIndex(syscall.Handle(f.Fd()))
}

func pathIdentity(path string) (uint64, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	h, err := syscall.CreateFile(
		p,
		syscall.GENERIC_READ,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return 0, err
	}
	defer syscall.CloseHandle(h)
	return fileIndex(h)
}

type byHandleFileInformation struct {
	FileAttributes     uint32
	CreationTime       syscall.Filetime
	LastAccessTime     syscall.Filetime
	LastWriteTime      syscall.Filetime
	VolumeSerialNumber uint32
	FileSizeHigh       uint32
	FileSizeLow        uint32
	NumberOfLinks      uint32
	FileIndexHigh      uint32
	FileIndexLow       uint32
}

var procGetFileInformationByHandle = syscall.NewLazyDLL("kernel32.dll").NewProc("GetFileInformationByHandle")

func fileIndex(h syscall.Handle) (uint64, error) {
	var info byHandleFileInformation
	r1, _, err := procGetFileInformationByHandle.Call(uintptr(h), uintptr(unsafe.Pointer(&info)))
	if r1 == 0 {
		return 0, err
	}
	idx := uint64(info.FileIndexHigh)<<32 | uint64(info.FileIndexLow)
	return uint64(info.VolumeSerialNumber)<<32 ^ idx, nil
}
