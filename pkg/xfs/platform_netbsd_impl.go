//go:build netbsd

package xfs

import "golang.org/x/sys/unix"

// package-level wrapper around unix.Statvfs so tests can replace it.
var statvfsFunc = unix.Statvfs

// ExportSetStatfsFunc allows tests to replace the underlying statvfs implementation.
// The function accepts a callback with signature func(path string, buf interface{}) error
// where buf will be a *unix.Statvfs_t. It returns a restore function.
func ExportSetStatfsFunc(fn func(string, interface{}) error) func() {
	if fn == nil {
		statvfsFunc = unix.Statvfs
		return func() { statvfsFunc = unix.Statvfs }
	}

	old := statvfsFunc
	statvfsFunc = func(path string, buf *unix.Statvfs_t) error {
		return fn(path, buf)
	}
	return func() { statvfsFunc = old }
}

// GetVolumeFreeSpace calculates the available storage space on NetBSD systems.
// It executes a native unix.Statvfs system call to resolve active blocks and volume geometry.
func GetVolumeFreeSpace(currentPath string, requiredBytes uint64) (bool, error) {
	var stat unix.Statvfs_t

	// All native syscall logic is contained here. The testing infrastructure
	// will mock this entire method execution via the IPkgBridgeXFS contract boundary.
	if err := statvfsFunc(currentPath, &stat); err != nil {
		return false, err
	}

	freeSpace := uint64(stat.Bavail) * uint64(stat.Bsize)
	return freeSpace >= requiredBytes, nil
}
