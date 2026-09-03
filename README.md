Go-Core-xfs
================================

![Go Test Coverage](https://raw.githubusercontent.com/github.com/AeonDigital/Go-Core-xfs/badges/.badges/main/coverage.svg)

> [Aeon Digital](http://www.aeondigital.com.br)
> rianna@aeondigital.com.br

&nbsp;

> Cross-platform filesystem utilities with a testable package bridge boundary.

This package provides a robust set of helper functions to resolve paths, inspect
filesystem entities, safely manipulate files and directories, and dynamically locate
standard user storage structures. It is designed for absolute, cross-platform stability
and isolates external operating system operations via an internal package bridge
contract to guarantee painless, concurrent unit testing.




&nbsp;
________________________________________________________________________________

## 1. INSTALLATION

Install the package from its current repository location:

```shell
go get github.com/AeonDigital/Go-Core-xfs@latest
```




&nbsp;
________________________________________________________________________________

## 2. PURPOSE

`xfs` centralizes common filesystem operations behind a consistent, predictable API.
It simplifies and standardizes tasks such as:

- **Path Normalization** — resolving absolute layouts, handling cross-platform separators,
  and expanding `~` home directory shortcuts.
- **Structural Inspections** — checking the existence and integrity of files, directories,
  and tracking low-level properties like device permission bits.
- **Safe Manipulations** — reading, writing, copying, and removing targets while
  using defensive guards to prevent accidental directory loss.
- **Dynamic Volume Auditing** — calculating real-time drive space availability across
  different platform architectures while respecting storage quotas.

The package is built to avoid direct global system calls in business code, favoring
a single bridge contract (`IPkgBridgeXFS`) that can be completely intercepted and
mocked during tests.




&nbsp;
________________________________________________________________________________

## 3. MULTI-PLATFORM LAYOUT MAPPING

The package dynamically translates standard user paths according to the runtime operating
system ecosystem conventions, abstracting complex specifications into clean API endpoints:

| API Method            | Linux / BSD (XDG Spec)                | Darwin (macOS Guidelines)       | Windows (Win32 Conventions) |
| --------------------- | ------------------------------------- | ------------------------------- | --------------------------- |
| `GetUserHomeDir()`    | `$HOME`                               | `$HOME`                         | `%USERPROFILE%`             |
| `GetUserConfigDir()`  | `$XDG_CONFIG_HOME` or `~/.config`     | `~/Library/Application Support` | `%AppData%`                 |
| `GetUserDataDir()`    | `$XDG_DATA_HOME` or `~/.local/share`  | `~/Library/Application Support` | `%LOCALAPPDATA%\Data`       |
| `GetUserCacheDir()`   | `$XDG_CACHE_HOME` or `~/.cache`       | `~/Library/Caches`              | `%LocalAppData%`            |
| `GetUserStateDir()`   | `$XDG_STATE_HOME` or `~/.local/state` | `~/Library/Application Support` | `%LOCALAPPDATA%\State`      |
| `GetUserRuntimeDir()` | `$XDG_RUNTIME_DIR` or `os.TempDir()`  | `os.TempDir()`                  | `%LOCALAPPDATA%\Runtime`    |
| `GetUserLogDir()`     | `$XDG_STATE_HOME` or `~/.local/state` | `~/Library/Logs`                | `%LOCALAPPDATA%\Logs`       |




&nbsp;
________________________________________________________________________________

## 4. BASIC USAGE

### 4.1. Path Resolution & Verification

Paths are automatically cleaned, cross-platform separators are normalized, and the
tilde prefix (`~`) is safely expanded to the current user's home directory.

```go
// Fully resolves and cleans absolute paths natively
path, err := xfs.RetrieveFullPath("~/workspace/../projects")
if err != nil {
    return err
}

// Inspect structural entities safely
if xfs.Exists(path) && xfs.IsDir(path) {
    isEmpty, _ := xfs.IsEmptyDir(path)
}
```



&nbsp;
---- ---- ---- ---- ---- ---- ---- ---- ---- ---- ---- ---- ---- ---- 

### 4.2. Directory & File Manipulations

Creation and modification methods accept standard optional variadic `os.FileMode`
arguments, applying sensible environment defaults when omitted.

```go
// Creates a complete nested path tree using 0o755 permissions by default
err := xfs.CreateDirPath("~/tmp/storage/configs")
if err != nil {
    return err
}

// Explicitly overrides default permissions with custom file modes
file, err := xfs.CreateFile("~/tmp/storage/configs/app.json", 0o600)
if err != nil {
    return err
}
defer file.Close()
```



&nbsp;
---- ---- ---- ---- ---- ---- ---- ---- ---- ---- ---- ---- ---- ---- 

### 4.3. Permission Bounds & Safetiness

`xfs` verifies read and write capabilities accurately across different operating
systems without altering target descriptors. Directory writability is safely evaluated
by attempting non-destructive transient actions under the hood.

```go
path := "~/tmp/storage"

if xfs.IsReadable(path) && xfs.IsWritable(path) {
    // Current application process has sufficient execution permissions
    perm, err := xfs.GetPermission(path)
}
```



&nbsp;
---- ---- ---- ---- ---- ---- ---- ---- ---- ---- ---- ---- ---- ---- 

### 4.4. Low-Level Volume Integrity

If a target path does not exist yet on the filesystem, `HasSpaceAvailable` automatically
walks upward through the directory hierarchy to locate the closest valid storage
boundary, executing native syscall audits against user quotas.

```go
// Safely matches volume space bounds before triggering large write payloads
requiredBytes := uint64(2 * 1024 * 1024 * 1024) // 2 GB

hasSpace, err := xfs.HasSpaceAvailable("~/downloads/incoming/payload.bin", requiredBytes)
if err != nil {
    return err
}

if !hasSpace {
    return errors.New("aborted: insufficient disk space on target volume")
}
```




&nbsp;
________________________________________________________________________________

## 6. TESTING AND MOCKS INTEGRATION

The package is engineered explicitly around the `IPkgBridgeXFS` interface contract
boundary. In production execution, it routes calls natively without intermediate
overhead to the Go standard library or platform-specific syscalls (`unix.Statfs`,
`unix.Statvfs`, or Win32 `GetDiskFreeSpaceEx`).

During your unit testing phases, you can completely substitute external system dependencies
with predictable mock structures generated by tools like **`xmocks`**:

```bash
# Generate automated mocks for your independent test suite
xmocks
```


```go
// Example of overriding the default bridge behavior within tests
func TestYourBusinessLogic(t *testing.T) {
    // xmocks generated structures can be injected directly into the internal hook
    // to bypass real disk interactions completely
}
```




&nbsp;
________________________________________________________________________________

## 7. ADDITIONAL INFORMATION

This project uses the [Semantic Versioning](https://semver.org) system proposed by
**Tom Preston-Werner**.




&nbsp;
________________________________________________________________________________

## 8. LICENSE

This project is offered under the [MIT license](LICENSE.md).