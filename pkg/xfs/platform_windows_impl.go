//go:build windows

package xfs

import "golang.org/x/sys/windows"

// DiskFreeSpaceBuf is the buffer that ExportSetStatfsFunc uses for Windows tests.
type DiskFreeSpaceBuf struct {
	FreeBytesAvailable     *uint64
	TotalNumberOfBytes     *uint64
	TotalNumberOfFreeBytes *uint64
}

// windowsGetDiskFreeSpaceExWrapper acts as a concrete implementation wrapper for the Win32 API.
// It handles the mandatory system conversion of the path string into a UTF-16 pointer
// before executing the native windows.GetDiskFreeSpaceEx system call.
// Returns an error if the path conversion fails or if the underlying OS API encounters a failure.
func windowsGetDiskFreeSpaceExWrapper(path string, buf *DiskFreeSpaceBuf) error {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return windows.GetDiskFreeSpaceEx(pathPtr, buf.FreeBytesAvailable, buf.TotalNumberOfBytes, buf.TotalNumberOfFreeBytes)
}

// statfsFunc holds the active function reference used to fetch disk storage metrics.
// By default, it points directly to windowsGetDiskFreeSpaceExWrapper for production runtime,
// but it can be safely hijacked and overridden by ExportSetStatfsFunc during testing phases.
var statfsFunc = windowsGetDiskFreeSpaceExWrapper

// ExportSetStatfsFunc allows tests to replace the underlying disk free space implementation.
// The function accepts a callback with signature func(path string, buf any) error
// where buf will be a *DiskFreeSpaceBuf. It returns a restore function.
func ExportSetStatfsFunc(fn func(string, any) error) func() {
	if fn == nil {
		if fn == nil {
			statfsFunc = windowsGetDiskFreeSpaceExWrapper
			return func() { statfsFunc = windowsGetDiskFreeSpaceExWrapper }
		}
	}

	old := statfsFunc
	statfsFunc = func(path string, buf *DiskFreeSpaceBuf) error {
		return fn(path, buf)
	}
	return func() { statfsFunc = old }
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
