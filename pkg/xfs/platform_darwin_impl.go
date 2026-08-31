//go:build darwin

package xfs

import "golang.org/x/sys/unix"

// package-level wrapper around unix.Statfs so tests can replace it.
var statfsFunc = unix.Statfs

// ExportSetStatfsFunc allows tests to replace the underlying statfs implementation.
// The function accepts a callback with signature func(path string, buf interface{}) error
// where buf will be a *unix.Statfs_t. It returns a restore function.
func ExportSetStatfsFunc(fn func(string, interface{}) error) func() {
	if fn == nil {
		statfsFunc = unix.Statfs
		return func() { statfsFunc = unix.Statfs }
	}

	old := statfsFunc
	statfsFunc = func(path string, buf *unix.Statfs_t) error {
		return fn(path, buf)
	}
	return func() { statfsFunc = old }
}

// GetVolumeFreeSpace calculates the available storage space on a Darwin-based system.
// It executes a native unix.Statfs system call to resolve active blocks and volume geometry.
func GetVolumeFreeSpace(currentPath string, requiredBytes uint64) (bool, error) {
	var stat unix.Statfs_t

	// All native syscall logic is contained here. The testing infrastructure
	// will mock this entire method execution via the IPkgBridgeXFS contract boundary.
	if err := statfsFunc(currentPath, &stat); err != nil {
		return false, err
	}

	freeSpace := uint64(stat.Bavail) * uint64(stat.Bsize)
	return freeSpace >= requiredBytes, nil
}
