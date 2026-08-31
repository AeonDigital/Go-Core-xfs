package xfs

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// pkgBridgeXFS holds the active package bridge instance used to handle
// external resources and system APIs. By default, it points to the concrete
// production implementation (PkgBridgeXFS), but it can be overridden
// via internal test utilities during unit testing execution.
var pkgBridgeXFS IPkgBridgeXFS = PkgBridgeXFS{}

// IPkgBridgeXFS defines the unified system call and external resource boundary
// for the xfs package. It consolidates all direct interactions with the operating
// system, standard library, and package-level utility functions into a singular,
// interceptable contract. This architectural decoupling eliminates global state,
// isolates platform-specific logic, and guarantees thread-safe mocking environments
// during concurrent black-box execution.
type IPkgBridgeXFS interface {
	FilepathAbs(path string) (string, error)
	RetrieveFullPath(path string) (string, error)

	// GROUP: SPECIAL DIRECTORIES

	GetUserHomeDir() (string, error)
	GetUserConfigDir() (string, error)
	GetUserDataDir() (string, error)
	GetUserCacheDir() (string, error)
	GetUserStateDir() (string, error)
	GetUserRuntimeDir() (string, error)
	GetUserLogDir() (string, error)

	// GROUP: EXISTENCE AND TYPES
	ReadDir(f *os.File, n int) ([]fs.DirEntry, error)

	IsDir(path string) bool
	Exists(path string) bool

	// GROUP : NATIVE OS
	OsStat(name string) (os.FileInfo, error)
	OsOpen(name string) (*os.File, error)
	// OSUserDataDir(home string, appName string) string
	// OSUserLogDir(home string, appName string) string
	OsOpenFile(name string, flag int, perm os.FileMode) (*os.File, error)
	OsCreateTemp(dir, pattern string) (*os.File, error)
	OsRemove(name string) error
	OsRemoveAll(path string) error
	OsChmod(name string, mode os.FileMode) error
	OsMkdir(name string, perm os.FileMode) error
	OsMkdirAll(path string, perm os.FileMode) error

	GetVolumeFreeSpace(currentPath string, requiredBytes uint64) (bool, error)

	GetRuntimeGOOS() string

	IOCopy(dst io.Writer, src io.Reader) (written int64, err error)
}

// PkgBridgeXFS serves as the standard, production-ready implementation of IPkgBridgeXFS.
// It directly maps abstract bridge requests to the Go standard library packages ('os',
// 'path/filepath') and local package infrastructure, routing calls natively without
// intermediate overhead or stateful transformations.
type PkgBridgeXFS struct{}

func (PkgBridgeXFS) FilepathAbs(path string) (string, error) {
	return filepath.Abs(path)
}

func (PkgBridgeXFS) RetrieveFullPath(path string) (string, error) {
	return RetrieveFullPath(path)
}

func (PkgBridgeXFS) GetUserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func (PkgBridgeXFS) GetUserConfigDir() (string, error) {
	return os.UserConfigDir()
}

func (PkgBridgeXFS) GetUserDataDir() (string, error) {
	return GetUserDataDir()
}

func (PkgBridgeXFS) GetUserCacheDir() (string, error) {
	return os.UserCacheDir()
}

func (PkgBridgeXFS) GetUserStateDir() (string, error) {
	return GetUserStateDir()
}

func (PkgBridgeXFS) GetUserRuntimeDir() (string, error) {
	return GetUserRuntimeDir()
}

func (PkgBridgeXFS) GetUserLogDir() (string, error) {
	return GetUserLogDir()
}

func (PkgBridgeXFS) GetParentPath(path string) (string, error) {
	return GetParentPath(path)
}

func (PkgBridgeXFS) OsStat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (PkgBridgeXFS) OsOpen(name string) (*os.File, error) {
	return os.Open(name)
}

func (PkgBridgeXFS) ReadDir(f *os.File, n int) ([]fs.DirEntry, error) {
	return f.ReadDir(n)
}

func (PkgBridgeXFS) IsDir(path string) bool {
	return IsDir(path)
}

func (PkgBridgeXFS) OsOpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

func (PkgBridgeXFS) OsCreateTemp(dir, pattern string) (*os.File, error) {
	return os.CreateTemp(dir, pattern)
}

func (PkgBridgeXFS) OsRemove(name string) error {
	return os.Remove(name)
}

func (PkgBridgeXFS) OsRemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (PkgBridgeXFS) OsChmod(name string, mode os.FileMode) error {
	return os.Chmod(name, mode)
}

func (PkgBridgeXFS) OsMkdir(name string, perm os.FileMode) error {
	return os.Mkdir(name, perm)
}

func (PkgBridgeXFS) OsMkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (PkgBridgeXFS) Exists(path string) bool {
	return Exists(path)
}

func (PkgBridgeXFS) GetVolumeFreeSpace(currentPath string, requiredBytes uint64) (bool, error) {
	return GetVolumeFreeSpace(currentPath, requiredBytes)
}

func (PkgBridgeXFS) GetRuntimeGOOS() string {
	return runtime.GOOS
}

func (PkgBridgeXFS) IOCopy(dst io.Writer, src io.Reader) (written int64, err error) {
	return io.Copy(dst, src)
}
