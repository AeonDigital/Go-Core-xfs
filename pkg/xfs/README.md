xfs
================================

> Cross-platform filesystem utilities with a testable package bridge boundary.

&nbsp;

This package provides a set of helper functions to resolve paths, inspect filesystem
entities, manage files and directories, and obtain standard user directories.
It is designed for safe, cross-platform use and isolates external system calls via
an internal package bridge for easier unit testing.


________________________________________________________________________________

## Purpose

`xfs` centralizes common filesystem operations behind a consistent API.
It simplifies tasks such as:

* resolving absolute paths and expanding `~` home directory shortcuts,
* checking file and directory existence, type and permissions,
* reading and writing filesystem entries,
* creating and deleting files and directories,
* locating standard user directories for config, data, and logs.

The package is built to avoid direct global system calls in business code,
favoring a single bridge contract that can be mocked during tests.


________________________________________________________________________________

## Installation

Use `go get` to add the package to your module:

```shell
  go get github.com/AeonDigital/Go-Core/xfs@latest
```

Import it in your code:

```go
import "github.com/AeonDigital/Go-Core-xfs/pkg/xfs"
```

If you also need structured error handling from the repository, import:

```go
import "github.com/AeonDigital/Go-Core-xerrors/pkg/xerrors"
```


________________________________________________________________________________

## Basic usage

Resolve paths and verify filesystem objects:

```go
path, err := xfs.RetrieveFullPath("~/projects")
if err != nil {
    return err
}

exists := xfs.Exists(path)
isDir := xfs.IsDir(path)
```

Work with files and directories:

```go
file, err := xfs.CreateFile("~/tmp/example.txt")
if err != nil {
    return err
}
file.Close()

err = xfs.CreateDirPath("~/tmp/nested/config")
if err != nil {
    return err
}

writable := xfs.IsWritable("~/tmp")
```

Use standard user directories for app-specific storage:

```go
configDir, err := xfs.GetUserConfigDir()
logDir, err := xfs.GetUserLogDir("myapp")
```


________________________________________________________________________________

## Supported APIs

The package exposes a broad set of filesystem helpers, including:

* `RetrieveFullPath`
* `GetUserHomeDir`
* `GetUserConfigDir`
* `GetUserDataDir`
* `GetUserCacheDir`
* `GetUserStateDir`
* `GetUserRuntimeDir`
* `GetUserLogDir`
* `GetParentPath`
* `Exists`
* `IsFile`
* `IsDir`
* `IsEmptyDir`
* `IsReadable`
* `IsWritable`
* `GetPermission`
* `SetPermission`
* `CreateFile`
* `CreateDir`
* `CreateDirPath`
* `OpenFileWrite`
* `OpenFileRead`
* `DeleteFile`
* `DeleteDir`
* `HasSpaceAvailable`
* `GetFileSize`
* `IsSameFile`

This package uses an internal `IPkgBridgeXFS` boundary to decouple real OS calls from
unit tests and platform-specific behavior.


________________________________________________________________________________

## External dependencies

`xfs` depends only on the Go standard library and the repository's internal
error package:

* `github.com/AeonDigital/Go-Core/xerrors`

No third-party external packages are required by this package.
