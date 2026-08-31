package xfs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AeonDigital/Go-Core-xerrors/pkg/xerrors"
	"github.com/AeonDigital/Go-Core-xfs/pkg/xstrings"
)

const (
	XERR_NONE   xerrors.ErrorCode = ""
	XERR_PKGCTX xerrors.ErrorCode = "ERR_XFS"
)

// RetrieveFullPath returns the fully resolved absolute path, or an error if
// system paths cannot be determined.
//
// Replaces the tilde prefix ("~") with the user's home directory and
// strictly resolves the final result into a clean, absolute filesystem path.
// If the path is relative, it anchors it to the current working directory, removing
// any redundancies like "." or "..".
func RetrieveFullPath(path string) (string, error) {
	if path == "" {
		errInfo := xstrings.BuildErrorInfo(
			"path", "''",
		)

		return "", xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_EMPTY_NOT_ALLOWED,
			nil,
			"",
			errInfo,
		)
	}

	// Corrigido para garantir compatibilidade cross-platform de barras
	path = strings.ReplaceAll(path, "\\", "/")

	var resolvedPath string

	if strings.HasPrefix(path, "~") {
		home, err := pkgBridgeXFS.GetUserHomeDir()
		if err != nil {
			errInfo := xstrings.BuildErrorInfo(
				"path", "~",
			)

			return "", xerrors.NewError500(
				XERR_PKGCTX,
				xerrors.XERR_RESOURCE_UNAVAILABLE,
				err,
				"failed to resolve user home directory",
				errInfo,
			)
		}

		if path == "~" {
			resolvedPath = home
		} else if path[1] == '/' {
			resolvedPath = filepath.Join(home, path[2:])
		} else {
			resolvedPath = path
		}
	} else {
		resolvedPath = path
	}

	// Usando a variável mockável aqui
	absolutePath, err := pkgBridgeXFS.FilepathAbs(resolvedPath)
	if err != nil {
		errInfo := xstrings.BuildErrorInfo(
			"path", path,
		)

		return "", xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_INVALID_VALUE,
			err,
			"failed to resolve absolute path",
			errInfo,
		)
	}

	return absolutePath, nil
}

// ============================================================================
// GROUP: SPECIAL DIRECTORIES
// ============================================================================

// GetUserHomeDir returns the current user's home directory.
//
// On Unix, including macOS, it returns the $HOME environment variable. On Windows, it returns %USERPROFILE%. On Plan 9, it returns the $home environment variable.
//
// If the expected variable is not set in the environment, UserHomeDir returns either a platform-specific default value or a non-nil error.
func GetUserHomeDir() (string, error) {
	return pkgBridgeXFS.GetUserHomeDir()
}

// GetUserConfigDir returns the default root directory to use for user-specific configuration data. Users should create their own application-specific subdirectory within this one and use that.
//
// On Unix systems, it returns $XDG_CONFIG_HOME as specified by https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html if non-empty, else $HOME/.config. On Darwin, it returns $HOME/Library/Application Support. On Windows, it returns %AppData%. On Plan 9, it returns $home/lib.
//
// If the location cannot be determined (for example, $HOME is not defined) or the path in $XDG_CONFIG_HOME is relative, then it will return an error.
func GetUserConfigDir() (string, error) {
	return pkgBridgeXFS.GetUserConfigDir()
}

// GetUserDataDir resolves the standard, platform-specific directory for persisting
// application data, strictly adhering to OS ecosystem conventions and specifications.
//
// Behavior by platform:
//   - Windows: Prioritizes the "%LOCALAPPDATA%\Data" environment variable path.
//     Falls back to "AppData\Local\Data" relative to the user's home directory if unset.
//   - Darwin (macOS): Returns the standard "Library/Application Support" directory
//     as mandated by Apple's File System guidelines.
//   - Linux/Other: Strictly follows the XDG Base Directory Specification. It queries
//     "$XDG_DATA_HOME" and falls back to the default "~/.local/share" directory.
//
// It returns the absolute path without creating the directory on the file system.
// An error is returned if the user's home directory cannot be resolved.
func GetUserDataDir() (string, error) {
	home, err := pkgBridgeXFS.GetUserHomeDir()
	if err != nil {
		return "", err
	}

	switch pkgBridgeXFS.GetRuntimeGOOS() {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return filepath.Join(home, "AppData", "Local", "Data"), nil
		}
		return filepath.Join(localAppData, "Data"), nil

	case "darwin":
		return filepath.Join(home, "Library", "Application Support"), nil
	default:
		xdgData := os.Getenv("XDG_DATA_HOME")
		if xdgData == "" {
			return filepath.Join(home, ".local", "share"), nil
		}
		return filepath.Join(xdgData), nil
	}
}

// GetUserCacheDir returns the default root directory to use for user-specific cached data. Users should create their own application-specific subdirectory within this one and use that.
//
// On Unix systems, it returns $XDG_CACHE_HOME as specified by https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html if non-empty, else $HOME/.cache. On Darwin, it returns $HOME/Library/Caches. On Windows, it returns %LocalAppData%. On Plan 9, it returns $home/lib/cache.
//
// If the location cannot be determined (for example, $HOME is not defined) or the path in $XDG_CACHE_HOME is relative, then it will return an error.
func GetUserCacheDir() (string, error) {
	return pkgBridgeXFS.GetUserCacheDir()
}

// GetUserStateDir resolves the standard, platform-specific directory for storing
// application state files, such as history, current session data, and non-critical logs.
//
// Behavior by platform:
//   - Windows: Prioritizes the "%LOCALAPPDATA%\State" environment variable path.
//     Falls back to "AppData\Local\State" relative to the user's home directory if unset.
//   - Darwin (macOS): Returns the standard "Library/Application Support" directory,
//     aligning state persistence with Apple's bundle state and data conventions.
//   - Linux/Other: Strictly follows the XDG Base Directory Specification. It queries
//     "$XDG_STATE_HOME" and falls back to the default "~/.local/state" directory.
//
// It returns the absolute path without creating the directory on the file system.
// An error is returned if the user's home directory cannot be resolved.
func GetUserStateDir() (string, error) {
	home, err := pkgBridgeXFS.GetUserHomeDir()
	if err != nil {
		return "", err
	}

	switch pkgBridgeXFS.GetRuntimeGOOS() {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return filepath.Join(home, "AppData", "Local", "State"), nil
		}
		return filepath.Join(localAppData, "State"), nil

	case "darwin":
		return filepath.Join(home, "Library", "Application Support"), nil

	default:
		xdgState := os.Getenv("XDG_STATE_HOME")
		if xdgState == "" {
			return filepath.Join(home, ".local", "state"), nil
		}
		return filepath.Join(xdgState), nil
	}
}

// GetUserRuntimeDir resolves the standard, platform-specific directory for storing
// ephemeral runtime files, such as sockets, named pipes, process locks, or volatile session configs.
//
// Behavior by platform:
//   - Windows: Appends a "Runtime" subdirectory to the "%LOCALAPPDATA%" environment path.
//     Falls back to "AppData\Local\Runtime" relative to the user's home directory if unset.
//   - Darwin (macOS): Defaults to the OS-managed transient directory returned by os.TempDir(),
//     suffixed with the application namespace, to match macOS ephemeral data policies.
//   - Linux/Other: Strictly follows the XDG Base Directory Specification. It queries
//     "$XDG_RUNTIME_DIR" and falls back to the system's global temporary path via os.TempDir() if unset.
//
// It returns the absolute path without creating the directory on the file system.
// An error is returned if the user's home directory cannot be resolved.
func GetUserRuntimeDir() (string, error) {
	home, err := pkgBridgeXFS.GetUserHomeDir()
	if err != nil {
		return "", err
	}

	switch pkgBridgeXFS.GetRuntimeGOOS() {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return filepath.Join(home, "AppData", "Local", "Runtime"), nil
		}
		return filepath.Join(localAppData, "Runtime"), nil

	case "darwin":
		// macOS native guidelines dictate using a unique subdirectory inside the OS transient storage
		return filepath.Join(os.TempDir()), nil

	default:
		xdgRuntime := os.Getenv("XDG_RUNTIME_DIR")
		if xdgRuntime == "" {
			return filepath.Join(os.TempDir()), nil
		}
		// XDG specification fallback for missing XDG_RUNTIME_DIR is the system's transient directory
		return filepath.Join(xdgRuntime), nil
	}
}

// GetUserLogDir resolves the standard, platform-specific directory for storing
// application log files, ensuring compliance with each operating system's logging conventions.
//
// Behavior by platform:
//   - Windows: Appends a "Logs" subdirectory to the "%LOCALAPPDATA%" environment path.
//     Falls back to "AppData\Local\Logs" relative to the user's home directory if unset.
//   - Darwin (macOS): Returns the standard user-specific logging directory "~/Library/Logs"
//     as defined by Apple's diagnostics and logging architecture.
//   - Linux/Other: Follows modern XDG extensions. It queries "$XDG_STATE_HOME" (falling back
//     to "~/.local/state") as the primary logging path, and checks "$XDG_CACHE_HOME" ("~/.cache")
//     as a secondary fallback.
//
// It returns the absolute path without creating the directory on the file system.
// An error is returned if the user's home directory cannot be resolved.
func GetUserLogDir() (string, error) {
	home, err := pkgBridgeXFS.GetUserHomeDir()
	if err != nil {
		return "", err
	}

	switch pkgBridgeXFS.GetRuntimeGOOS() {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return filepath.Join(home, "AppData", "Local", "Logs"), nil
		}
		return filepath.Join(localAppData, "Logs"), nil

	case "darwin":
		// Native macOS path for user-space application logs
		return filepath.Join(home, "Library", "Logs"), nil

	default:
		// XDG_STATE_HOME (~/.local/state) - Recommended for persistent history/logs
		xdgState := os.Getenv("XDG_STATE_HOME")
		if xdgState == "" {
			return filepath.Join(home, ".local", "state"), nil
		}

		// Final fallback conforming to standard paths
		return filepath.Join(xdgState), nil
	}
}

// GetParentPath extracts and returns the parent directory path from a given file path.
// It automatically expands the tilde prefix ("~") using the RetrieveFullPath function before isolating the layout.
// Returns the parent directory string, or an empty string if the input path is empty or if path expansion fails.
func GetParentPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(path)
	if err != nil {
		return "", err
	}

	return filepath.Dir(expandedPath), nil
}

// ============================================================================
// GROUP: EXISTENCE AND TYPES
// ============================================================================

// Exists checks if a file or directory exists at the given path.
// It automatically expands the tilde prefix ("~") using the RetrieveFullPath function before executing the check.
// Returns true if the target path is found on the system, or false if it does not exist or if path expansion fails.
func Exists(path string) bool {
	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(path)
	if err != nil {
		return false
	}

	_, err = pkgBridgeXFS.OsStat(expandedPath)
	return err == nil
}

// IsFile checks if the specified path exists and points to a regular file.
// It automatically expands the tilde prefix ("~") using the RetrieveFullPath function before executing the check.
// Returns true if the path is a regular file, or false if it does not exist, is a directory/symlink, or if system calls fail.
func IsFile(path string) bool {
	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(path)
	if err != nil {
		return false
	}

	info, err := pkgBridgeXFS.OsStat(expandedPath)
	if err != nil {
		return false
	}

	return info.Mode().IsRegular()
}

// IsDir checks if the specified path exists and points to a directory.
// It automatically expands the tilde prefix ("~") using the RetrieveFullPath function before executing the check.
// Returns true if the path is a directory, or false if it does not exist, is a regular file/symlink, or if system calls fail.
func IsDir(path string) bool {
	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(path)
	if err != nil {
		return false
	}

	info, err := pkgBridgeXFS.OsStat(expandedPath)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// IsEmptyDir checks if the specified directory exists and contains no files or subdirectories.
// It automatically expands the tilde prefix ("~") before evaluating the target directory path.
// Returns true if the directory is completely empty, or false along with an error if the path does not exist, is not a directory, or if read permissions are denied.
func IsEmptyDir(path string) (bool, error) {
	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(path)
	if err != nil {
		return false, err
	}

	f, err := pkgBridgeXFS.OsOpen(expandedPath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = pkgBridgeXFS.ReadDir(f, 1)
	if err == nil {
		return false, nil
	}

	if errors.Is(err, io.EOF) {
		return true, nil
	}

	return false, err
}

// IsFileDirExists extracts the parent directory path from a given file path and checks if it exists as a valid directory.
// It automatically expands the tilde prefix ("~") before isolating the parent directory layout.
// Returns true if the parent directory structure exists on the system, or false if the path is empty, expansion fails, or the directory does not exist.
func IsFileDirExists(filePath string) bool {
	if filePath == "" {
		return false
	}

	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(filePath)
	if err != nil {
		return false
	}

	parentDir := filepath.Dir(expandedPath)
	return pkgBridgeXFS.IsDir(parentDir)
}

// ============================================================================
// GROUP: PERMISSIONS
// ============================================================================

// IsReadable checks if the current application process has sufficient permissions to read the file or directory at the specified path.
// It automatically expands the tilde prefix ("~") and robustly handles both regular files and directory permission boundaries across different operating systems.
// Returns true if the path can be successfully read, or false if permissions are denied or if the path does not exist.
func IsReadable(path string) bool {
	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(path)
	if err != nil {
		return false
	}

	info, err := pkgBridgeXFS.OsStat(expandedPath)
	if err != nil {
		return false
	}

	if info.IsDir() {
		f, err := pkgBridgeXFS.OsOpen(expandedPath)
		if err != nil {
			return false
		}
		defer f.Close()

		_, err = pkgBridgeXFS.ReadDir(f, 1)

		return err == nil || errors.Is(err, io.EOF)
	}

	file, err := pkgBridgeXFS.OsOpenFile(expandedPath, os.O_RDONLY, 0)
	if err != nil {
		return false
	}
	file.Close()
	return true
}

// IsWritable checks if the current application process has sufficient permissions to modify or write to the file or directory at the specified path.
// It automatically expands the tilde prefix ("~") and tests directories by attempting to create a temporary file inside them, or files by opening them in write-only append mode.
// Returns true if the path can be written to, or false if permissions are denied, if the path does not exist, or if the test operations fail.
func IsWritable(path string) bool {
	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(path)
	if err != nil {
		return false
	}

	info, err := pkgBridgeXFS.OsStat(expandedPath)
	if err != nil {
		return false
	}

	if info.IsDir() {
		tempFile, err := pkgBridgeXFS.OsCreateTemp(expandedPath, ".fsutil_test_")
		if err != nil {
			return false
		}

		defer pkgBridgeXFS.OsRemove(tempFile.Name())
		defer tempFile.Close()

		return true
	}

	file, err := pkgBridgeXFS.OsOpenFile(expandedPath, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return false
	}
	file.Close()
	return true
}

// GetPermission retrieves the system permission bits (os.FileMode) for the specified file or directory.
// It automatically expands the tilde prefix ("~") using the RetrieveFullPath function before executing the check.
// Returns the file mode permissions, or an error if path expansion fails or if the path does not exist.
func GetPermission(path string) (os.FileMode, error) {
	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(path)
	if err != nil {
		return 0, err
	}

	info, err := pkgBridgeXFS.OsStat(expandedPath)
	if err != nil {
		errInfo := xstrings.BuildErrorInfo(
			"path", path,
		)

		return 0, xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_OPERATION_FAILED,
			err,
			"failed to read path metadata",
			errInfo,
		)
	}

	return info.Mode().Perm(), nil
}

// SetPermission updates the filesystem permission bits (os.FileMode) for the specified file or directory.
// It automatically expands the tilde prefix ("~") using the RetrieveFullPath function before applying the changes.
// Returns nil on success, or an error if path expansion fails, the path does not exist, or the operation is denied.
func SetPermission(path string, perm os.FileMode) error {
	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(path)
	if err != nil {
		return err
	}

	err = pkgBridgeXFS.OsChmod(expandedPath, perm)
	if err != nil {
		errInfo := xstrings.BuildErrorInfo(
			"perm", perm.String(),
		)

		return xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_PERMISSION_DENIED,
			err,
			"failed to change target filesystem permissions",
			errInfo,
		)
	}

	return nil
}

// ============================================================================
// GROUP: CREATE, EDIT and DELETE
// ============================================================================

// CreateFile creates a new file at the specified path, truncating it if it already exists.
// It automatically expands the tilde prefix ("~") before executing the operation.
// An optional os.FileMode can be passed; if omitted, it defaults to standard system permissions (0666).
// Returns a pointer to the opened file object (*os.File) ready for writing, or an error if path expansion or file creation fails.
func CreateFile(path string, perm ...os.FileMode) (*os.File, error) {
	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(path)
	if err != nil {
		return nil, err
	}

	filePerm := os.FileMode(0o666)
	if len(perm) > 0 {
		filePerm = perm[0]
	}

	file, err := os.OpenFile(expandedPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, filePerm)
	if err != nil {
		errInfo := xstrings.BuildErrorInfo(
			"path", path,
		)

		return nil, xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_RESOURCE_UNAVAILABLE,
			err,
			"failed to create targeted file",
			errInfo,
		)
	}

	return file, nil
}

// CreateDir creates a single directory at the specified path.
// It automatically expands the tilde prefix ("~") before creating the folder and fails if the parent directory structure does not exist.
// An optional os.FileMode can be passed; if omitted, it defaults to standard permissions (0755).
// Returns nil on success, or an error if path expansion fails or the directory cannot be created.
func CreateDir(path string, perm ...os.FileMode) error {
	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(path)
	if err != nil {
		return err
	}

	dirPerm := os.FileMode(0o755)
	if len(perm) > 0 {
		dirPerm = perm[0]
	}

	err = pkgBridgeXFS.OsMkdir(expandedPath, dirPerm)
	if err != nil {
		errInfo := xstrings.BuildErrorInfo(
			"path", path,
		)

		return xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_RESOURCE_UNAVAILABLE,
			err,
			"failed to create targeted directory",
			errInfo,
		)
	}

	return nil
}

// CreateDirPath creates a directory path along with any necessary parent directories.
// It automatically expands the tilde prefix ("~") before creating the folder structure.
// An optional os.FileMode can be passed; if omitted, it defaults to standard permissions (0755).
// If the target path already exists, it does nothing and returns nil.
func CreateDirPath(path string, perm ...os.FileMode) error {
	if path == "" {
		errInfo := xstrings.BuildErrorInfo(
			"path", "''",
		)

		return xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_EMPTY_NOT_ALLOWED,
			nil,
			"",
			errInfo,
		)
	}

	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(path)
	if err != nil {
		return err
	}

	dirPerm := os.FileMode(0o755)
	if len(perm) > 0 {
		dirPerm = perm[0]
	}

	err = pkgBridgeXFS.OsMkdirAll(expandedPath, dirPerm)
	if err != nil {
		errInfo := xstrings.BuildErrorInfo(
			"path", path,
		)

		return xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_RESOURCE_UNAVAILABLE,
			err,
			"failed to create targeted directory tree",
			errInfo,
		)
	}

	return nil
}

// OpenFileWrite opens a file for writing, creating it with the specified or default permissions (0644) if it does not exist.
// It automatically expands the tilde prefix ("~") before opening the path.
// If truncate is true, the file's existing content is cleared upon opening; otherwise, new data is appended to the end.
// An optional os.FileMode can be passed to override the default creation permissions.
// Returns a pointer to the opened file object (*os.File) ready for writing, or an error if path expansion or file opening fails.
func OpenFileWrite(path string, truncate bool, perm ...os.FileMode) (*os.File, error) {
	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(path)
	if err != nil {
		return nil, err
	}

	// 0644: Standard read/write permissions for the owner, read-only for others
	filePerm := os.FileMode(0o644)
	if len(perm) > 0 {
		filePerm = perm[0]
	}

	// Flag Explanations:
	// os.O_WRONLY: Open exclusively for writing
	// os.O_CREATE: Create the file if it does not exist
	// os.O_APPEND: Write data at the end of the file (log style)
	// os.O_TRUNC: Truncate entire file content on open file
	useFlag := os.O_APPEND
	if truncate {
		useFlag = os.O_TRUNC
	}

	file, err := os.OpenFile(expandedPath, os.O_WRONLY|os.O_CREATE|useFlag, filePerm)
	if err != nil {
		errInfo := xstrings.BuildErrorInfo(
			"path", path,
		)

		return nil, xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_RESOURCE_UNAVAILABLE,
			err,
			"failed to open targeted file for writing operations",
			errInfo,
		)
	}

	return file, nil
}

// OpenFileRead opens an existing file exclusively in read-only mode.
// It automatically expands the tilde prefix ("~") using the RetrieveFullPath function before attempting to open the path.
// Returns a pointer to the opened file object (*os.File) ready for reading, or an error if path expansion fails or if the file does not exist.
func OpenFileRead(path string) (*os.File, error) {
	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(path)
	if err != nil {
		return nil, err
	}

	// os.O_RDONLY: Open the file read-only.
	file, err := os.OpenFile(expandedPath, os.O_RDONLY, 0)
	if err != nil {
		errInfo := xstrings.BuildErrorInfo(
			"path", path,
		)

		return nil, xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_OPERATION_FAILED,
			err,
			"failed to open targeted file for reading operations",
			errInfo,
		)
	}

	return file, nil
}

// CopyFile duplicates the contents of a source file to a destination file.
// It automatically expands the tilde prefix ("~") on both file paths before execution.
// It creates the destination file if it does not exist, or opens it in append-mode if it does.
// Returns nil on success, or an error if path expansion fails, file opening operations are denied,
// the byte stream transfer fails, or the structural metadata cannot be flushed to disk.
func CopyFile(pathOrigin string, pathDestiny string) error {
	originFile, err := OpenFileRead(pathOrigin)
	if err != nil {
		return err
	}
	defer originFile.Close()

	destinyFile, err := OpenFileWrite(pathDestiny, false)
	if err != nil {
		return err
	}
	defer destinyFile.Close()

	_, err = pkgBridgeXFS.IOCopy(destinyFile, originFile)
	if err != nil {
		return err
	}

	return destinyFile.Sync()
}

// DeleteFile removes a regular file at the specified path.
// It automatically expands the tilde prefix ("~") before performing the check and deletion.
// It explicitly fails if the targeted path is a directory to prevent accidental directory structural loss.
// Returns nil on success, or an error if path expansion fails, the path points to a directory, or the system removal operation is denied.
func DeleteFile(path string) error {
	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(path)
	if err != nil {
		return err
	}

	if pkgBridgeXFS.IsDir(expandedPath) {
		errInfo := xstrings.BuildErrorInfo(
			"path", path,
		)

		return xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_INVALID_TYPE,
			nil,
			"cannot use DeleteFile on a directory",
			errInfo,
		)
	}

	err = pkgBridgeXFS.OsRemove(expandedPath)
	if err != nil {
		errInfo := xstrings.BuildErrorInfo(
			"path", path,
		)

		return xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_RESOURCE_UNAVAILABLE,
			err,
			"failed to delete file at targeted path",
			errInfo,
		)
	}

	return nil
}

// DeleteDir removes a directory at the specified path, with optional recursive deletion of all its contents.
// It automatically expands the tilde prefix ("~") before executing any filesystem validation or removal steps.
// If recursive is true, it uses os.RemoveAll to forcefully delete the folder and everything inside it; otherwise, it uses os.Remove and fails if the directory is not completely empty.
// Returns nil on success, or an error if path expansion fails, the path is not a directory, or the system removal operation is denied.
func DeleteDir(path string, recursive bool) error {
	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(path)
	if err != nil {
		return err
	}

	if !pkgBridgeXFS.IsDir(expandedPath) {
		errInfo := xstrings.BuildErrorInfo(
			"path", path,
		)

		return xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_INVALID_TYPE,
			nil,
			"path is not a directory or does not exist",
			errInfo,
		)
	}

	if recursive {
		err = pkgBridgeXFS.OsRemoveAll(expandedPath)
	} else {
		err = pkgBridgeXFS.OsRemove(expandedPath)
	}

	if err != nil {
		errInfo := xstrings.BuildErrorInfo(
			"path", path,
		)

		msgWithData := "failed to delete targeted directory hierarchy; ('recursive'=" + strconv.FormatBool(recursive) + ")"
		return xerrors.NewError500(
			XERR_PKGCTX,
			xerrors.XERR_RESOURCE_UNAVAILABLE,
			err,
			msgWithData,
			errInfo,
		)
	}

	return nil
}

// ============================================================================
// GROUP: METADATA AND INTEGRITY
// ============================================================================

// HasSpaceAvailable checks if the storage volume containing the specified path has at least the required number of bytes free.
// It accepts a filesystem path (path) and the minimum required space in bytes (requiredBytes).
// If the target path does not exist, it traverses upward to find the closest existing parent directory to accurately target the underlying volume.
// Returns true if the volume has sufficient space available for the current user, or false along with an error if the path expansion, UTF-16 conversion, or Win32 GetDiskFreeSpaceEx API call fails.
func HasSpaceAvailable(path string, requiredBytes uint64) (bool, error) {
	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(path)
	if err != nil {
		return false, err
	}

	currentPath := expandedPath
	for !pkgBridgeXFS.Exists(currentPath) {
		parent := filepath.Dir(currentPath)
		if parent == currentPath {
			errInfo := xstrings.BuildErrorInfo(
				"path", path,
			)

			return false, xerrors.NewError500(
				XERR_PKGCTX,
				xerrors.XERR_OPERATION_FAILED,
				nil,
				"could not find a valid base volume path",
				errInfo,
			)
		}

		currentPath = parent
	}

	return pkgBridgeXFS.GetVolumeFreeSpace(currentPath, requiredBytes)
}

// GetFileSize returns the size of the file in bytes at the specified path.
// It automatically expands the tilde prefix ("~") before checking the path.
// It explicitly returns an error if the path points to a directory or if the file does not exist.
// Returns the file size in bytes as an int64, or an error if path expansion, system calls, or validations fail.
func GetFileSize(path string) (int64, error) {
	expandedPath, err := pkgBridgeXFS.RetrieveFullPath(path)
	if err != nil {
		return 0, err
	}

	info, err := pkgBridgeXFS.OsStat(expandedPath)
	if err != nil {
		return 0, err
	}

	if info.IsDir() {
		return 0, os.ErrInvalid // Directories do not have a standard "file size"
	}

	return info.Size(), nil
}

// IsSameFile checks if two different paths point to the exact same physical file or directory on the storage device.
// It automatically expands the tilde prefix ("~") on both inputs and resolves any symbolic links encountered.
// It relies on Go's native os.SameFile mechanism to compare low-level system identities like inodes or file IDs.
// Returns true if both paths target the identical filesystem resource, or false along with an error if metadata retrieval fails.
func IsSameFile(pathA, pathB string) (bool, error) {
	expandedA, err := pkgBridgeXFS.RetrieveFullPath(pathA)
	if err != nil {
		return false, err
	}

	expandedB, err := pkgBridgeXFS.RetrieveFullPath(pathB)
	if err != nil {
		return false, err
	}

	// os.Stat automatically follows symbolic links
	infoA, err := pkgBridgeXFS.OsStat(expandedA)
	if err != nil {
		return false, err
	}

	infoB, err := pkgBridgeXFS.OsStat(expandedB)
	if err != nil {
		return false, err
	}

	// os.SameFile compares the underlying system-specific file identities (inodes/file IDs)
	return os.SameFile(infoA, infoB), nil
}
