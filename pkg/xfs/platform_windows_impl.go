//go:build windows

package xfs

import "golang.org/x/sys/windows"

// DiskFreeSpaceBuf is the buffer that ExportSetStatfsFunc uses for Windows tests.
type DiskFreeSpaceBuf struct {
	FreeBytesAvailable     *uint64
	TotalNumberOfBytes     *uint64
	TotalNumberOfFreeBytes *uint64
}

// package-level wrapper around windows.GetDiskFreeSpaceEx so tests can replace it.
var statfsFunc = func(path string, buf *DiskFreeSpaceBuf) error {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return windows.GetDiskFreeSpaceEx(pathPtr, buf.FreeBytesAvailable, buf.TotalNumberOfBytes, buf.TotalNumberOfFreeBytes)
}

// ExportSetStatfsFunc allows tests to replace the underlying disk free space implementation.
// The function accepts a callback with signature func(path string, buf interface{}) error
// where buf will be a *DiskFreeSpaceBuf. It returns a restore function.
func ExportSetStatfsFunc(fn func(string, interface{}) error) func() {
	if fn == nil {
		statfsFunc = func(path string, buf *DiskFreeSpaceBuf) error {
			pathPtr, err := windows.UTF16PtrFromString(path)
			if err != nil {
				return err
			}
			return windows.GetDiskFreeSpaceEx(pathPtr, buf.FreeBytesAvailable, buf.TotalNumberOfBytes, buf.TotalNumberOfFreeBytes)
		}
		return func() { statfsFunc = windowsGetDiskFreeSpaceExWrapper }
	}

	old := statfsFunc
	statfsFunc = func(path string, buf *DiskFreeSpaceBuf) error {
		return fn(path, buf)
	}
	return func() { statfsFunc = old }
}

func windowsGetDiskFreeSpaceExWrapper(path string, buf *DiskFreeSpaceBuf) error {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return windows.GetDiskFreeSpaceEx(pathPtr, buf.FreeBytesAvailable, buf.TotalNumberOfBytes, buf.TotalNumberOfFreeBytes)
}

// GetVolumeFreeSpace calculates the available storage space on Windows systems.
// It executes a native Win32 GetDiskFreeSpaceEx system call to resolve active blocks and volume geometry.
func GetVolumeFreeSpace(currentPath string, requiredBytes uint64) (bool, error) {
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64
	buf := &DiskFreeSpaceBuf{
		FreeBytesAvailable:     &freeBytesAvailable,
		TotalNumberOfBytes:     &totalNumberOfBytes,
		TotalNumberOfFreeBytes: &totalNumberOfFreeBytes,
	}

	// All native syscall logic is contained here. The testing infrastructure
	// will mock this entire method execution via the IPkgBridgeXFS contract boundary.
	if err := statfsFunc(currentPath, buf); err != nil {
		return false, err
	}

	return freeBytesAvailable >= requiredBytes, nil
}
