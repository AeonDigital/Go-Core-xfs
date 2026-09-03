//go:build !linux && !freebsd && !dragonfly && !darwin && !netbsd && !openbsd && !solaris && !illumos && !windows

package xfs

import "errors"

func GetVolumeFreeSpace(currentPath string, requiredBytes uint64) (bool, error) {
	return false, errors.New("get volume free space is not supported on this operating system")
}

// ExportSetStatfsFunc is a no-op on unsupported platforms and provided so tests
// that import this package can compile on any OS.
func ExportSetStatfsFunc(fn func(string, interface{}) error) func() {
	return func() {}
}
