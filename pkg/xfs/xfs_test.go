package xfs_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/AeonDigital/Go-Core-xfs/pkg/xfs"
	"github.com/AeonDigital/Go-Core-xfs/pkg/xfs/xmock"
)

// SetupTestMock orchestrates the initialization, configuration, and injection
// of the strict filesystem package bridge mock for a specific test execution scenario.
// It applies the optional configuration function and returns a cleanup function to safely
// restore the original package state upon completion.
func SetupTestMock(t *testing.T, tt xmock.TestCaseXFS) func() {
	t.Helper()

	if tt.Env != nil {
		for k, v := range tt.Env {
			t.Setenv(k, v)
		}
	}

	mockBridge := xmock.NewMockXFS()

	if tt.MockFn != nil {
		tt.MockFn(mockBridge)
	}

	restore := xfs.SetBridgeXFSForTest(mockBridge)
	return restore
}

// AssertResult handles all standard assertions for a xmock.TestCaseXFS execution.
// It verifies error presence, specific error message content, and final value equality
// using deep reflection comparison against the expected criteria.
func AssertResult(t *testing.T, tt xmock.TestCaseXFS, testFunction string, got any, err error) {
	t.Helper()

	// Intercepts *os.File types to ensure valid streams and handle resource cleanup
	if err == nil && !tt.WantErr && got != nil {
		if filePtr, ok := got.(*os.File); ok {
			if filePtr == nil {
				t.Error("Expected a valid *os.File pointer on success, got nil")
			} else {
				filePtr.Close()
			}
			// Sets got to nil to match a clean tt.Want expectation for stream returns
			got = nil
		}
	}

	if (err != nil) != tt.WantErr {
		t.Fatalf("%s() error = %v, wantErr %v", testFunction, err, tt.WantErr)
	}

	if tt.WantErr {
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		if !strings.Contains(errStr, tt.ErrContains) {
			t.Errorf("%s() Expected error to contain %q, got %q", testFunction, tt.ErrContains, errStr)
		}
		return
	}

	if !reflect.DeepEqual(got, tt.Want) {
		t.Errorf("%s() = %v (%T), want %v (%T)", testFunction, got, got, tt.Want, tt.Want)
	}
}

// mockFileInfo implements os.FileInfo needed for tests
type mockFileInfo struct {
	os.FileInfo
	isRegular bool
	isDir     bool
	mode      os.FileMode
	size      int64
}

func (m mockFileInfo) Mode() os.FileMode {
	// Technical Note: Preserves permission bits if explicitly assigned in m.mode
	if m.mode != 0 {
		return m.mode
	}
	if m.isRegular {
		return 0 // A FileMode 0 represents a regular file in Go
	}
	return os.ModeDir // Returns a directory mode if not a regular file
}

func (m mockFileInfo) IsDir() bool {
	return m.isDir
}

func (m mockFileInfo) Size() int64 {
	return m.size
}

// Função auxiliar para resolver caminhos relativos ao PWD do teste com segurança
func mustAbs(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("failed to setup absolute path for test: %v", err)
	}
	return abs
}

func TestWrappersCoverage(t *testing.T) {
	bridge := xfs.PkgBridgeXFS{}

	// Setup realistic dynamic paths to guarantee execution safety without panics
	tmpDir := t.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "coverage_stub")
	if err != nil {
		t.Fatalf("failed to setup coverage file requirement: %v", err)
	}
	defer tmpFile.Close()

	// 1. Core and Path Resolution Wrappers
	_, _ = bridge.FilepathAbs(tmpDir)
	_, _ = bridge.RetrieveFullPath(".")
	_, _ = bridge.GetParentPath(tmpFile.Name())

	// 2. Special Directories Wrappers
	_, _ = bridge.GetUserHomeDir()
	_, _ = bridge.GetUserConfigDir()
	_, _ = bridge.GetUserDataDir()
	_, _ = bridge.GetUserCacheDir()
	_, _ = bridge.GetUserStateDir()
	_, _ = bridge.GetUserRuntimeDir()
	_, _ = bridge.GetUserLogDir()

	// 3. File and Directory Boundary Interaction Wrappers
	_, _ = bridge.OsStat(tmpFile.Name())
	if f, err := bridge.OsOpen(tmpFile.Name()); err == nil {
		_, _ = bridge.ReadDir(f, -1)
		_ = f.Close()
	}
	if f, err := bridge.OsOpenFile(tmpFile.Name(), os.O_RDONLY, 0); err == nil {
		_ = f.Close()
	}
	if fTemp, err := bridge.OsCreateTemp(tmpDir, "ephemeral_*"); err == nil {
		_ = fTemp.Close()
	}

	// 4. State, Checking and Inspection Logic Wrappers
	_ = bridge.IsDir(tmpDir)
	_ = bridge.Exists(tmpFile.Name())
	_, _ = bridge.GetVolumeFreeSpace(tmpDir, 1024)

	// 5. Destructive and Mutation Wrappers (Executed safely over t.TempDir)
	targetMkdir := filepath.Join(tmpDir, "sub_dir")
	targetMkdirAll := filepath.Join(tmpDir, "nested", "tree")

	_ = bridge.OsMkdir(targetMkdir, 0o755)
	_ = bridge.OsMkdirAll(targetMkdirAll, 0o755)
	_ = bridge.OsChmod(tmpFile.Name(), 0o644)
	_ = bridge.OsRemove(targetMkdir)
	_ = bridge.OsRemoveAll(targetMkdirAll)

	var buf bytes.Buffer
	r := strings.NewReader("dados")
	_, _ = bridge.IOCopy(&buf, r)

	_, _ = xfs.GetUserHomeDir()
	_, _ = xfs.GetUserConfigDir()
	_, _ = xfs.GetUserCacheDir()
}

func TestRetrieveFullPath(t *testing.T) {
	testFunction := "RetrieveFullPath"
	mockHome := filepath.FromSlash("/user/mockhome")

	type inputArgs struct {
		path string
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Empty path string should fail immediately",
			Input: inputArgs{
				path: "",
			},
			WantErr:     true,
			MockFn:      func(m *xmock.MockXFS) {},
			ErrContains: "[CTX: ERR_XFS][ERR: E1003][MSG: empty string value not allowed]",
		},
		{
			Name: "Standard path without tilde should turn into absolute",
			Input: inputArgs{
				path: "documents/project",
			},
			Want:    mustAbs(t, "documents/project"),
			WantErr: false,
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.FilepathAbs(filepath.Abs)
			},
		},
		{
			Name: "Tilde alone should resolve exactly to user home directory",
			Input: inputArgs{
				path: "~",
			},
			Want:    filepath.ToSlash(mockHome),
			WantErr: false,
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetUserHomeDir(mockHome, nil)
				m.SetReturn.FilepathAbs(filepath.ToSlash(mockHome), nil)
			},
		},
		{
			Name: "Tilde with forward slash should append structure to home directory",
			Input: inputArgs{
				path: "~/downloads/file.txt",
			},
			Want:    filepath.ToSlash(filepath.Join(mockHome, "downloads", "file.txt")),
			WantErr: false,
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetUserHomeDir(mockHome, nil)
				m.SetReturn.FilepathAbs(filepath.ToSlash(filepath.Join(mockHome, "downloads", "file.txt")), nil)
			},
		},
		{
			Name: "Tilde with backward slash should append structure to home directory",
			Input: inputArgs{
				path: "~\\documents\\report.pdf",
			},
			Want:    filepath.ToSlash(filepath.Join(mockHome, "documents", "report.pdf")),
			WantErr: false,
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetUserHomeDir(mockHome, nil)
				m.SetReturn.FilepathAbs(filepath.ToSlash(filepath.Join(mockHome, "documents", "report.pdf")), nil)
			},
		},
		{
			Name: "Tilde stuck to letters without slashes should be treated as relative name",
			Input: inputArgs{
				path: "~aeon-folder",
			},
			Want:    mustAbs(t, "~aeon-folder"),
			WantErr: false,
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetUserHomeDir(mockHome, nil)
				m.OnCall.FilepathAbs(filepath.Abs)
			},
		},
		{
			Name: "Should fail and wrap error when user home resolution fails",
			Input: inputArgs{
				path: "~/downloads",
			},
			WantErr:     true,
			ErrContains: "[CTX: ERR_XFS][ERR: E1008][MSG: failed to resolve user home directory]",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetUserHomeDir("", errors.New("permission denied"))
			},
		},
		{
			Name: "Should fail when absolute path resolution returns an error",
			Input: inputArgs{
				path: "documents/project",
			},
			WantErr:     true,
			ErrContains: "[CTX: ERR_XFS][ERR: E2001][MSG: failed to resolve absolute path]",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.FilepathAbs("", errors.New("mocked filesystem breakdown error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			got, err := xfs.RetrieveFullPath(args.path)

			// Validation
			AssertResult(t, tt, testFunction, got, err)
		})
	}
}

func TestGetUserDataDir(t *testing.T) {
	testFunction := "GetUserDataDir"
	mockHome := filepath.FromSlash("/user/mockhome")
	mockDataDir := filepath.Join(mockHome, "mockdatadir")

	tests := []xmock.TestCaseXFS{
		{
			Name:    "Should return an error when the user home directory cannot be resolved",
			WantErr: true,
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetUserHomeDir("", errors.New("failed on retrieve user home path"))
			},
		},
		{
			Name: "Windows: Should fall back to standard AppData path when LOCALAPPDATA environment variable is empty",
			Env: map[string]string{
				"LOCALAPPDATA": "",
			},
			Want: filepath.Join(mockHome, "AppData", "Local", "Data"),
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("windows")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
		{
			Name: "Windows: Should append Data suffix when LOCALAPPDATA environment variable is explicitly defined",
			Env: map[string]string{
				"LOCALAPPDATA": mockDataDir,
			},
			Want: filepath.Join(mockDataDir, "Data"),
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("windows")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
		{
			Name: "Darwin: Should return Apple's standard Application Support path",
			Want: filepath.Join(mockHome, "Library", "Application Support"),
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("darwin")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
		{
			Name: "Linux: Should fall back to standard XDG path when XDG_DATA_HOME environment variable is empty",
			Env: map[string]string{
				"XDG_DATA_HOME": "",
			},
			Want: filepath.Join(mockHome, ".local", "share"),
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("linux")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
		{
			Name: "Linux: Should honor the custom path when XDG_DATA_HOME environment variable is explicitly defined",
			Env: map[string]string{
				"XDG_DATA_HOME": "/my/xdg/data/home/directory", //<<< @TODO No ambiente do githubactions não consigo definir uma variável de ambiente
			},
			Want: "/my/xdg/data/home/directory",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("linux")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Trigger the business implementation
			got, err := xfs.GetUserDataDir()

			// Validation
			AssertResult(t, tt, testFunction, got, err)
		})
	}
}

func TestGetUserStateDir(t *testing.T) {
	testFunction := "GetUserStateDir"
	mockHome := filepath.FromSlash("/user/mockhome")
	mockStateDir := filepath.Join(mockHome, "mockstatedir")

	tests := []xmock.TestCaseXFS{
		{
			Name:    "Should return an error when the user home directory cannot be resolved",
			WantErr: true,
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetUserHomeDir("", errors.New("failed on retrieve user home path"))
			},
		},
		{
			Name: "Windows: Should fall back to standard AppData path when LOCALAPPDATA environment variable is empty",
			Env: map[string]string{
				"LOCALAPPDATA": "",
			},
			Want: filepath.Join(mockHome, "AppData", "Local", "State"),
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("windows")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
		{
			Name: "Windows: Should append State suffix when LOCALAPPDATA environment variable is explicitly defined",
			Env: map[string]string{
				"LOCALAPPDATA": mockStateDir,
			},
			Want: filepath.Join(mockStateDir, "State"),
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("windows")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
		{
			Name: "Darwin: Should return Apple's standard Application Support path as the state container",
			Want: filepath.Join(mockHome, "Library", "Application Support"),
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("darwin")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
		{
			Name: "Linux: Should fall back to standard XDG path when XDG_STATE_HOME environment variable is empty",
			Env: map[string]string{
				"XDG_STATE_HOME": "",
			},
			Want: filepath.Join(mockHome, ".local", "state"),
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("linux")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
		{
			Name: "Linux: Should honor the custom path when XDG_STATE_HOME environment variable is explicitly defined",
			Env: map[string]string{
				"XDG_STATE_HOME": "/my/xdg/state/home/directory", //<<< @TODO No ambiente do githubactions não consigo definir uma variável de ambiente
			},
			Want: "/my/xdg/state/home/directory",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("linux")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			defer SetupTestMock(t, tt)()
			got, err := xfs.GetUserStateDir()
			AssertResult(t, tt, testFunction, got, err)
		})
	}
}

func TestGetUserRuntimeDir(t *testing.T) {
	testFunction := "GetUserRuntimeDir"
	mockHome := filepath.FromSlash("/user/mockhome")
	mockRuntimeDir := filepath.Join(mockHome, "mockruntimedir")

	tests := []xmock.TestCaseXFS{
		{
			Name:    "Should return an error when the user home directory cannot be resolved",
			WantErr: true,
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetUserHomeDir("", errors.New("failed on retrieve user home path"))
			},
		},
		{
			Name: "Windows: Should fall back to standard AppData path when LOCALAPPDATA environment variable is empty",
			Env: map[string]string{
				"LOCALAPPDATA": "",
			},
			Want: filepath.Join(mockHome, "AppData", "Local", "Runtime"),
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("windows")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
		{
			Name: "Windows: Should append Runtime suffix when LOCALAPPDATA environment variable is explicitly defined",
			Env: map[string]string{
				"LOCALAPPDATA": mockRuntimeDir,
			},
			Want: filepath.Join(mockRuntimeDir, "Runtime"),
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("windows")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
		{
			Name: "Darwin: Should fall back to the system's dynamic temporary directory",
			Want: filepath.Join(os.TempDir()),
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("darwin")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
		{
			Name: "Linux: Should fall back to the system's temporary directory when XDG_RUNTIME_DIR environment variable is empty",
			Env: map[string]string{
				"XDG_RUNTIME_DIR": "",
			},
			Want: filepath.Join(os.TempDir()),
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("linux")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
		{
			Name: "Linux: Should honor the custom path when XDG_RUNTIME_DIR environment variable is explicitly defined",
			Env: map[string]string{
				"XDG_RUNTIME_DIR": "/my/xdg/runtime/directory", //<<< @TODO No ambiente do githubactions não consigo definir uma variável de ambiente
			},
			Want: "/my/xdg/runtime/directory",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("linux")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			defer SetupTestMock(t, tt)()
			got, err := xfs.GetUserRuntimeDir()
			AssertResult(t, tt, testFunction, got, err)
		})
	}
}

func TestGetUserLogDir(t *testing.T) {
	testFunction := "GetUserLogDir"
	mockHome := filepath.FromSlash("/user/mockhome")
	mockLogDir := filepath.Join(mockHome, "mocklogdir")

	tests := []xmock.TestCaseXFS{
		{
			Name:    "Should return an error when the user home directory cannot be resolved",
			WantErr: true,
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetUserHomeDir("", errors.New("failed on retrieve user home path"))
			},
		},
		{
			Name: "Windows: Should fall back to standard AppData path when LOCALAPPDATA environment variable is empty",
			Env: map[string]string{
				"LOCALAPPDATA": "",
			},
			Want: filepath.Join(mockHome, "AppData", "Local", "Logs"),
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("windows")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
		{
			Name: "Windows: Should append Logs suffix when LOCALAPPDATA environment variable is explicitly defined",
			Env: map[string]string{
				"LOCALAPPDATA": mockLogDir,
			},
			Want: filepath.Join(mockLogDir, "Logs"),
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("windows")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
		{
			Name: "Darwin: Should return Apple's standard Logs path",
			Want: filepath.Join(mockHome, "Library", "Logs"),
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("darwin")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
		{
			Name: "Linux: Should fall back to standard XDG state directory when XDG_STATE_HOME environment variable is empty",
			Env: map[string]string{
				"XDG_STATE_HOME": "",
			},
			Want: filepath.Join(mockHome, ".local", "state"),
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("linux")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
		{
			Name: "Linux: Should honor the custom path when XDG_STATE_HOME environment variable is explicitly defined",
			Env: map[string]string{
				"XDG_STATE_HOME": "/my/xdg/state/logs", //<<< @TODO No ambiente do githubactions não consigo definir uma variável de ambiente
			},
			Want: "/my/xdg/state/logs",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.GetRuntimeGOOS("linux")
				m.SetReturn.GetUserHomeDir(mockHome, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			defer SetupTestMock(t, tt)()
			got, err := xfs.GetUserLogDir()
			AssertResult(t, tt, testFunction, got, err)
		})
	}
}

func TestGetParentPath(t *testing.T) {
	testFunction := "GetUserLogDir"

	type inputArgs struct {
		path string
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Empty file path should return empty string immediately",
			Input: inputArgs{
				path: "",
			},
			Want: "",
		},
		{
			Name: "Should return error if RetrieveFullPath returns an error",
			Input: inputArgs{
				path: "invalid/path",
			},
			Want:    "",
			WantErr: true,
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("", errors.New("failed on retrieve full path"))
			},
		},
		{
			Name: "Should return the correct parent directory on success",
			Input: inputArgs{
				path: "~/documents/project/file.txt",
			},
			Want: filepath.FromSlash("/user/mockhome/documents/project"),
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/user/mockhome/documents/project/file.txt", nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			got, err := xfs.GetParentPath(args.path)

			// Validation
			AssertResult(t, tt, testFunction, got, err)
		})
	}
}

func TestExists(t *testing.T) {
	testFunction := "Exists"

	type inputArgs struct {
		path string
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return false if RetrieveFullPath returns an error",
			Input: inputArgs{
				path: "invalid/path",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(path string) (string, error) {
					return "", errors.New("mocked retrieval failure")
				})
			},
		},
		{
			Name: "Should return false if file or directory does not exist",
			Input: inputArgs{
				path: "~/missing-file.txt",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/user/mockhome/missing-file.txt", nil)
				m.SetReturn.OsStat(nil, os.ErrNotExist)
			},
		},
		{
			Name: "Should return true if file or directory exists successfully",
			Input: inputArgs{
				path: "~/existing-file.txt",
			},
			Want:        true,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/user/mockhome/existing-file.txt", nil)
				m.SetReturn.OsStat(nil, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			got := xfs.Exists(args.path)

			// Validation
			AssertResult(t, tt, testFunction, got, nil)
		})
	}
}

func TestIsFile(t *testing.T) {
	testFunction := "IsFile"

	type inputArgs struct {
		path string
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return false if RetrieveFullPath returns an error",
			Input: inputArgs{
				path: "invalid/path",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(path string) (string, error) {
					return "", errors.New("mocked retrieval failure")
				})
			},
		},
		{
			Name: "Should return false if os.Stat returns an error",
			Input: inputArgs{
				path: "~/missing-file.txt",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/user/mockhome/missing-file.txt", nil)
				m.SetReturn.OsStat(nil, os.ErrNotExist)
			},
		},
		{
			Name: "Should return false if path exists but it is a directory",
			Input: inputArgs{
				path: "~/documents",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/user/mockhome/documents", nil)
				m.SetReturn.OsStat(mockFileInfo{isRegular: false}, nil)
			},
		},
		{
			Name: "Should return true if path exists and it is a regular file",
			Input: inputArgs{
				path: "~/documents/notes.txt",
			},
			Want:        true,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/user/mockhome/documents/notes.txt", nil)
				m.SetReturn.OsStat(mockFileInfo{isRegular: true}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			got := xfs.IsFile(args.path)

			// Validation
			AssertResult(t, tt, testFunction, got, nil)
		})
	}
}

func TestIsDir(t *testing.T) {
	testFunction := "IsDir"

	type inputArgs struct {
		path string
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return false if RetrieveFullPath returns an error",
			Input: inputArgs{
				path: "invalid/path",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return "", errors.New("err")
				})
			},
		},
		{
			Name: "Should return false if os.Stat returns an error",
			Input: inputArgs{
				path: "~/missing-dir",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/dir", nil)
				m.SetReturn.OsStat(nil, os.ErrNotExist)
			},
		},
		{
			Name: "Should return false if path exists but it is a regular file",
			Input: inputArgs{
				path: "~/notes.txt",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/file", nil)
				fi, _ := os.Stat("xfs_test.go")
				m.SetReturn.OsStat(fi, nil)
			},
		},
		{
			Name: "Should return true if path exists and it is a directory",
			Input: inputArgs{
				path: "~/documents",
			},
			Want:        true,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/dir", nil)
				fi, _ := os.Stat(".")
				m.SetReturn.OsStat(fi, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			got := xfs.IsDir(args.path)

			// Validation
			AssertResult(t, tt, testFunction, got, nil)
		})
	}
}

func TestIsEmptyDir(t *testing.T) {
	testFunction := "IsEmptyDir"

	type inputArgs struct {
		path string
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return error if RetrieveFullPath fails",
			Input: inputArgs{
				path: "invalid/path",
			},
			Want:        false,
			WantErr:     true,
			ErrContains: "retrieval failure",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return "", errors.New("retrieval failure")
				})
			},
		},
		{
			Name: "Should return error if os.Open fails",
			Input: inputArgs{
				path: "~/missing-dir",
			},
			Want:        false,
			WantErr:     true,
			ErrContains: "permission denied",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/dir", nil)
				m.OnCall.OsOpen(func(n string) (*os.File, error) {
					return nil, os.ErrPermission
				})
			},
		},
		{
			Name: "Should return false if directory is NOT empty (err is nil)",
			Input: inputArgs{
				path: "~/full-dir",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath(".", nil)
				f, _ := os.Open(".")
				m.OnCall.OsOpen(func(n string) (*os.File, error) {
					return f, nil
				})
				m.OnCall.ReadDir(func(f *os.File, n int) ([]os.DirEntry, error) {
					return []os.DirEntry{nil}, nil
				})
			},
		},
		{
			Name: "Should return true if directory is empty (err is EOF)",
			Input: inputArgs{
				path: "~/empty-dir",
			},
			Want:        true,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath(".", nil)
				f, _ := os.Open(".")
				m.OnCall.OsOpen(func(n string) (*os.File, error) {
					return f, nil
				})
				m.OnCall.ReadDir(func(f *os.File, n int) ([]os.DirEntry, error) {
					return nil, io.EOF
				})
			},
		},
		{
			Name: "Should return generic error if ReadDir fails with unexpected error",
			Input: inputArgs{
				path: "~/broken-dir",
			},
			Want:        false,
			WantErr:     true,
			ErrContains: "hardware disk failure",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath(".", nil)
				f, _ := os.Open(".")
				m.OnCall.OsOpen(func(n string) (*os.File, error) {
					return f, nil
				})
				m.OnCall.ReadDir(func(f *os.File, n int) ([]os.DirEntry, error) {
					return nil, errors.New("hardware disk failure")
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			got, err := xfs.IsEmptyDir(args.path)

			// Validation
			AssertResult(t, tt, testFunction, got, err)
		})
	}
}

func TestIsFileDirExists(t *testing.T) {
	testFunction := "IsFileDirExists"

	type inputArgs struct {
		path string
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return false immediately if filePath is empty",
			Input: inputArgs{
				path: "",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn:      nil,
		},
		{
			Name: "Should return false if RetrieveFullPath returns an error",
			Input: inputArgs{
				path: "invalid/path",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return "", errors.New("err")
				})
			},
		},
		{
			Name: "Should return false if the parent directory does not exist",
			Input: inputArgs{
				path: "~/documents/missing-folder/file.txt",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/home/documents/missing-folder/file.txt", nil)
				m.OnCall.IsDir(func(p string) bool {
					return false
				})
			},
		},
		{
			Name: "Should return true if the parent directory exists",
			Input: inputArgs{
				path: "~/documents/project/file.txt",
			},
			Want:        true,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/home/documents/project/file.txt", nil)
				m.OnCall.IsDir(func(p string) bool {
					return true
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			got := xfs.IsFileDirExists(args.path)

			// Validation
			AssertResult(t, tt, testFunction, got, nil)
		})
	}
}

func TestIsReadable(t *testing.T) {
	testFunction := "IsReadable"

	type inputArgs struct {
		path string
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return false if RetrieveFullPath fails",
			Input: inputArgs{
				path: "invalid/path",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return "", errors.New("err")
				})
			},
		},
		{
			Name: "Should return false if os.Stat fails",
			Input: inputArgs{
				path: "~/missing-path",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock", nil)
				m.SetReturn.OsStat(nil, os.ErrNotExist)
			},
		},
		{
			Name: "Directory: Should return false if os.Open fails (permission denied)",
			Input: inputArgs{
				path: "~/protected-dir",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath(".", nil)
				fi, _ := os.Stat(".")
				m.SetReturn.OsStat(fi, nil)
				m.OnCall.OsOpen(func(n string) (*os.File, error) {
					return nil, os.ErrPermission
				})
			},
		},
		{
			Name: "Directory: Should return true if ReadDir succeeds (not empty)",
			Input: inputArgs{
				path: "~/populated-dir",
			},
			Want:        true,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath(".", nil)
				fi, _ := os.Stat(".")
				m.SetReturn.OsStat(fi, nil)
				f, _ := os.Open(".")
				m.OnCall.OsOpen(func(n string) (*os.File, error) {
					return f, nil
				})
				m.OnCall.ReadDir(func(f *os.File, n int) ([]os.DirEntry, error) {
					return []os.DirEntry{nil}, nil
				})
			},
		},
		{
			Name: "Directory: Should return true if ReadDir returns EOF (empty directory)",
			Input: inputArgs{
				path: "~/empty-dir",
			},
			Want:        true,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath(".", nil)
				fi, _ := os.Stat(".")
				m.SetReturn.OsStat(fi, nil)
				f, _ := os.Open(".")
				m.OnCall.OsOpen(func(n string) (*os.File, error) {
					return f, nil
				})
				m.OnCall.ReadDir(func(f *os.File, n int) ([]os.DirEntry, error) {
					return nil, io.EOF
				})
			},
		},
		{
			Name: "Directory: Should return false if ReadDir returns an unexpected error",
			Input: inputArgs{
				path: "~/broken-dir",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath(".", nil)
				fi, _ := os.Stat(".")
				m.SetReturn.OsStat(fi, nil)
				f, _ := os.Open(".")
				m.OnCall.OsOpen(func(n string) (*os.File, error) {
					return f, nil
				})
				m.OnCall.ReadDir(func(f *os.File, n int) ([]os.DirEntry, error) {
					return nil, errors.New("disk err")
				})
			},
		},
		{
			Name: "File: Should return false if os.OpenFile fails (permission denied)",
			Input: inputArgs{
				path: "~/protected-file.txt",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/file.txt", nil)
				m.SetReturn.OsStat(mockFileInfo{isRegular: true}, nil)
				m.OnCall.OsOpenFile(func(n string, f int, mode os.FileMode) (*os.File, error) {
					return nil, os.ErrPermission
				})
			},
		},
		{
			Name: "File: Should return true if os.OpenFile succeeds",
			Input: inputArgs{
				path: "~/readable-file.txt",
			},
			Want:        true,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/file.txt", nil)
				m.SetReturn.OsStat(mockFileInfo{isRegular: true}, nil)
				m.OnCall.OsOpenFile(func(n string, f int, mode os.FileMode) (*os.File, error) {
					tmpFile, err := os.CreateTemp(t.TempDir(), "readable_stub")
					return tmpFile, err
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			got := xfs.IsReadable(args.path)

			// Validation
			AssertResult(t, tt, testFunction, got, nil)
		})
	}
}

func TestIsWritable(t *testing.T) {
	testFunction := "IsWritable"

	type inputArgs struct {
		path string
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return false if RetrieveFullPath fails",
			Input: inputArgs{
				path: "invalid/path",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return "", errors.New("err")
				})
			},
		},
		{
			Name: "Should return false if os.Stat fails",
			Input: inputArgs{
				path: "~/missing-path",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock", nil)
				m.SetReturn.OsStat(nil, os.ErrNotExist)
			},
		},
		{
			Name: "Directory: Should return false if os.CreateTemp fails (read-only filesystem)",
			Input: inputArgs{
				path: "~/readonly-dir",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/dir", nil)
				m.SetReturn.OsStat(mockFileInfo{isRegular: false, isDir: true}, nil)
				m.OnCall.OsCreateTemp(func(d, p string) (*os.File, error) {
					return nil, os.ErrPermission
				})
			},
		},
		{
			Name: "Directory: Should return true if os.CreateTemp succeeds",
			Input: inputArgs{
				path: "~/writable-dir",
			},
			Want:        true,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/dir", nil)
				m.SetReturn.OsStat(mockFileInfo{isRegular: false, isDir: true}, nil)
				m.OnCall.OsCreateTemp(func(d, p string) (*os.File, error) {
					tmpFile, err := os.CreateTemp(t.TempDir(), "writable_dir_stub")
					return tmpFile, err
				})
				m.OnCall.OsRemove(func(name string) error {
					return nil
				})
			},
		},
		{
			Name: "File: Should return false if os.OpenFile fails (no write permission)",
			Input: inputArgs{
				path: "~/protected-file.txt",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/file.txt", nil)
				m.SetReturn.OsStat(mockFileInfo{isRegular: true}, nil)
				m.OnCall.OsOpenFile(func(n string, f int, mode os.FileMode) (*os.File, error) {
					return nil, os.ErrPermission
				})
			},
		},
		{
			Name: "File: Should return true if os.OpenFile succeeds",
			Input: inputArgs{
				path: "~/writable-file.txt",
			},
			Want:        true,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/file.txt", nil)
				m.SetReturn.OsStat(mockFileInfo{isRegular: true}, nil)
				m.OnCall.OsOpenFile(func(n string, f int, mode os.FileMode) (*os.File, error) {
					tmpFile, err := os.CreateTemp(t.TempDir(), "writable_file_stub")
					return tmpFile, err
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			got := xfs.IsWritable(args.path)

			// Validation
			AssertResult(t, tt, testFunction, got, nil)
		})
	}
}

func TestGetPermission(t *testing.T) {
	testFunction := "GetPermission"

	type inputArgs struct {
		path string
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return error if RetrieveFullPath fails",
			Input: inputArgs{
				path: "invalid/path",
			},
			Want:        os.FileMode(0),
			WantErr:     true,
			ErrContains: "[CTX: XFS][MSG: failed to resolve user home directory]",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return "", errors.New("[CTX: XFS][MSG: failed to resolve user home directory]")
				})
			},
		},
		{
			Name: "Should return error if os.Stat fails",
			Input: inputArgs{
				path: "~/missing-path",
			},
			Want:        os.FileMode(0),
			WantErr:     true,
			ErrContains: "[CTX: ERR_XFS][ERR: E4002][MSG: failed to read path metadata]",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock", nil)
				m.SetReturn.OsStat(nil, os.ErrNotExist)
			},
		},
		{
			Name: "Should return correct file permissions on success",
			Input: inputArgs{
				path: "~/notes.txt",
			},
			Want:        os.FileMode(0o644),
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/notes.txt", nil)
				m.SetReturn.OsStat(mockFileInfo{mode: os.FileMode(0o644)}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			got, err := xfs.GetPermission(args.path)

			// Validation
			AssertResult(t, tt, testFunction, got, err)
		})
	}
}

func TestSetPermission(t *testing.T) {
	testFunction := "SetPermission"

	type inputArgs struct {
		path string
		perm os.FileMode
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return error if RetrieveFullPath fails",
			Input: inputArgs{
				path: "invalid/path",
				perm: 0o644,
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: XFS][MSG: failed to resolve user home directory]",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return "", errors.New("[CTX: XFS][MSG: failed to resolve user home directory]")
				})
			},
		},
		{
			Name: "Should return error if os.Chmod fails (permission denied)",
			Input: inputArgs{
				path: "~/protected-file.txt",
				perm: 0o777,
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: ERR_XFS][ERR: E1007][MSG: failed to change target filesystem permissions]",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/protected-file.txt", nil)
				m.OnCall.OsChmod(func(path string, mode os.FileMode) error {
					return os.ErrPermission
				})
			},
		},
		{
			Name: "Should return nil on success",
			Input: inputArgs{
				path: "~/regular-file.txt",
				perm: 0o600,
			},
			Want:        nil,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/regular-file.txt", nil)
				m.OnCall.OsChmod(func(path string, mode os.FileMode) error {
					return nil
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			err := xfs.SetPermission(args.path, args.perm)

			// Validation
			AssertResult(t, tt, testFunction, nil, err)
		})
	}
}

func TestCreateFile(t *testing.T) {
	testFunction := "CreateFile"

	type inputArgs struct {
		path string
		perm []os.FileMode
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return error if RetrieveFullPath fails",
			Input: inputArgs{
				path: "invalid/path",
				perm: nil,
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: XFS][MSG: failed to resolve user home directory]",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return "", errors.New("[CTX: XFS][MSG: failed to resolve user home directory]")
				})
			},
		},
		{
			Name: "Should return error if os.OpenFile fails (permission denied)",
			Input: inputArgs{
				path: "~/protected-file.txt",
				perm: nil,
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: ERR_XFS][ERR: E1008][MSG: failed to create targeted file]",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/protected-file.txt", nil)
				m.OnCall.OsOpenFile(func(n string, f int, mode os.FileMode) (*os.File, error) {
					return nil, os.ErrPermission
				})
			},
		},
		{
			Name: "Should succeed using default permissions (0666) when no perm is provided",
			Input: inputArgs{
				path: "~/default-perm.txt",
				perm: nil,
			},
			Want:        nil,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					tmpFile := filepath.Join(t.TempDir(), "stub_default.txt")
					return tmpFile, nil
				})
				m.OnCall.OsOpenFile(func(n string, f int, mode os.FileMode) (*os.File, error) {
					if mode != 0o666 {
						return nil, fmt.Errorf("expected default permission 0666, got %o", mode)
					}
					return os.OpenFile(n, os.O_CREATE|os.O_WRONLY, mode)
				})
			},
		},
		{
			Name: "Should succeed using custom permission when provided via variadic parameter",
			Input: inputArgs{
				path: "~/custom-perm.txt",
				perm: []os.FileMode{0o600},
			},
			Want:        nil,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					tmpFile := filepath.Join(t.TempDir(), "stub_custom.txt")
					return tmpFile, nil
				})
				m.OnCall.OsOpenFile(func(n string, f int, mode os.FileMode) (*os.File, error) {
					if mode != 0o600 {
						return nil, fmt.Errorf("expected custom permission 0600, got %o", mode)
					}
					return os.OpenFile(n, os.O_CREATE|os.O_WRONLY, mode)
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			got, err := xfs.CreateFile(args.path, args.perm...)

			// Validation
			AssertResult(t, tt, testFunction, got, err)
		})
	}
}

func TestCreateDir(t *testing.T) {
	testFunction := "CreateDir"

	type inputArgs struct {
		path string
		perm []os.FileMode
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return error if RetrieveFullPath fails",
			Input: inputArgs{
				path: "invalid/path",
				perm: nil,
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: XFS][MSG: failed to resolve user home directory]",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return "", errors.New("[CTX: XFS][MSG: failed to resolve user home directory]")
				})
			},
		},
		{
			Name: "Should return error if os.Mkdir fails (permission denied)",
			Input: inputArgs{
				path: "~/protected-dir",
				perm: nil,
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: ERR_XFS][ERR: E1008][MSG: failed to create targeted directory]",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/protected-dir", nil)
				m.OnCall.OsMkdir(func(path string, perm os.FileMode) error {
					return os.ErrPermission
				})
			},
		},
		{
			Name: "Should succeed using default permissions (0755) when no perm is provided",
			Input: inputArgs{
				path: "~/default-dir",
				perm: nil,
			},
			Want:        nil,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return filepath.Join(t.TempDir(), "sub_default"), nil
				})
				m.OnCall.OsMkdir(func(path string, perm os.FileMode) error {
					if perm != 0o755 {
						return fmt.Errorf("expected default permission 0755, got %o", perm)
					}
					return os.Mkdir(path, perm)
				})
			},
		},
		{
			Name: "Should succeed using custom permission when provided via variadic parameter",
			Input: inputArgs{
				path: "~/custom-dir",
				perm: []os.FileMode{0o700},
			},
			Want:        nil,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return filepath.Join(t.TempDir(), "sub_custom"), nil
				})
				m.OnCall.OsMkdir(func(path string, perm os.FileMode) error {
					if perm != 0o700 {
						return fmt.Errorf("expected custom permission 0700, got %o", perm)
					}
					return os.Mkdir(path, perm)
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			err := xfs.CreateDir(args.path, args.perm...)

			// Validation
			AssertResult(t, tt, testFunction, nil, err)
		})
	}
}

func TestCreateDirPath(t *testing.T) {
	testFunction := "CreateDirPath"

	type inputArgs struct {
		path string
		perm []os.FileMode
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return error immediately if path is empty",
			Input: inputArgs{
				path: "",
				perm: nil,
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: ERR_XFS][ERR: E1003][MSG: empty string value not allowed]",
			MockFn:      nil,
		},
		{
			Name: "Should return error if RetrieveFullPath fails",
			Input: inputArgs{
				path: "invalid/path",
				perm: nil,
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: XFS][MSG: failed to resolve user home directory]",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return "", errors.New("[CTX: XFS][MSG: failed to resolve user home directory]")
				})
			},
		},
		{
			Name: "Should return error if os.MkdirAll fails (permission denied)",
			Input: inputArgs{
				path: "~/protected-tree",
				perm: nil,
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: ERR_XFS][ERR: E1008][MSG: failed to create targeted directory tree]",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/protected-tree", nil)
				m.OnCall.OsMkdirAll(func(path string, perm os.FileMode) error {
					return os.ErrPermission
				})
			},
		},
		{
			Name: "Should succeed creating a nested tree using default permissions (0755)",
			Input: inputArgs{
				path: "~/nested/tree/default",
				perm: nil,
			},
			Want:        nil,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return filepath.Join(t.TempDir(), "nested", "tree", "default"), nil
				})
				m.OnCall.OsMkdirAll(func(path string, perm os.FileMode) error {
					if perm != 0o755 {
						return fmt.Errorf("expected default permission 0755, got %o", perm)
					}
					return os.MkdirAll(path, perm)
				})
			},
		},
		{
			Name: "Should succeed creating a nested tree using custom permissions",
			Input: inputArgs{
				path: "~/nested/tree/custom",
				perm: []os.FileMode{0o700},
			},
			Want:        nil,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return filepath.Join(t.TempDir(), "nested", "tree", "custom"), nil
				})
				m.OnCall.OsMkdirAll(func(path string, perm os.FileMode) error {
					if perm != 0o700 {
						return fmt.Errorf("expected custom permission 0700, got %o", perm)
					}
					return os.MkdirAll(path, perm)
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			err := xfs.CreateDirPath(args.path, args.perm...)

			// Validation
			AssertResult(t, tt, testFunction, nil, err)
		})
	}
}

func TestOpenFileWrite(t *testing.T) {
	testFunction := "OpenFileWrite"

	type inputArgs struct {
		path     string
		truncate bool
		perm     []os.FileMode
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return error if RetrieveFullPath fails",
			Input: inputArgs{
				path:     "invalid/path",
				truncate: false,
				perm:     nil,
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: XFS][MSG: failed to resolve user home directory]",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return "", errors.New("[CTX: XFS][MSG: failed to resolve user home directory]")
				})
			},
		},
		{
			Name: "Should return error if os.OpenFile fails (permission denied)",
			Input: inputArgs{
				path:     "~/protected-file.txt",
				truncate: false,
				perm:     nil,
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: ERR_XFS][ERR: E1008][MSG: failed to open targeted file for writing operations]",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/protected-file.txt", nil)
				m.OnCall.OsOpenFile(func(n string, f int, mode os.FileMode) (*os.File, error) {
					return nil, os.ErrPermission
				})
			},
		},
		{
			Name: "Should succeed with APPEND flag and default permission (0644) when truncate is false",
			Input: inputArgs{
				path:     "~/append-file.txt",
				truncate: false,
				perm:     nil,
			},
			Want:        nil,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return filepath.Join(t.TempDir(), "append.txt"), nil
				})
				m.OnCall.OsOpenFile(func(n string, f int, mode os.FileMode) (*os.File, error) {
					if (f & os.O_APPEND) == 0 {
						return nil, fmt.Errorf("expected flag to contain os.O_APPEND")
					}
					if mode != 0o644 {
						return nil, fmt.Errorf("expected default permission 0644, got %o", mode)
					}
					return os.OpenFile(n, f, mode)
				})
			},
		},
		{
			Name: "Should succeed with TRUNC flag and custom permission when truncate is true",
			Input: inputArgs{
				path:     "~/trunc-file.txt",
				truncate: true,
				perm:     []os.FileMode{0o600},
			},
			Want:        nil,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return filepath.Join(t.TempDir(), "trunc.txt"), nil
				})
				m.OnCall.OsOpenFile(func(n string, f int, mode os.FileMode) (*os.File, error) {
					if (f & os.O_TRUNC) == 0 {
						return nil, fmt.Errorf("expected flag to contain os.O_TRUNC")
					}
					if mode != 0o600 {
						return nil, fmt.Errorf("expected custom permission 0600, got %o", mode)
					}
					return os.OpenFile(n, f, mode)
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			got, err := xfs.OpenFileWrite(args.path, args.truncate, args.perm...)

			// Validation
			AssertResult(t, tt, testFunction, got, err)
		})
	}
}

func TestOpenFileRead(t *testing.T) {
	testFunction := "OpenFileRead"

	type inputArgs struct {
		path string
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return error if RetrieveFullPath fails",
			Input: inputArgs{
				path: "invalid/path",
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: XFS][MSG: failed to resolve user home directory]",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return "", errors.New("[CTX: XFS][MSG: failed to resolve user home directory]")
				})
			},
		},
		{
			Name: "Should return error if os.OpenFile fails (file not found)",
			Input: inputArgs{
				path: "~/missing-file.txt",
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: ERR_XFS][ERR: E4002][MSG: failed to open targeted file for reading operations]",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/missing-file.txt", nil)
				m.OnCall.OsOpenFile(func(n string, f int, mode os.FileMode) (*os.File, error) {
					return nil, os.ErrNotExist
				})
			},
		},
		{
			Name: "Should succeed opening a file with RDONLY flag",
			Input: inputArgs{
				path: "~/readable-file.txt",
			},
			Want:        nil,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					tmpFile := filepath.Join(t.TempDir(), "read_stub.txt")
					if err := os.WriteFile(tmpFile, []byte("data"), 0o644); err != nil {
						t.Fatalf("failed to setup test file: %v", err)
					}
					return tmpFile, nil
				})
				m.OnCall.OsOpenFile(func(n string, f int, mode os.FileMode) (*os.File, error) {
					if f != os.O_RDONLY {
						return nil, fmt.Errorf("expected flag os.O_RDONLY, got %d", f)
					}
					return os.OpenFile(n, f, mode)
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			got, err := xfs.OpenFileRead(args.path)

			// Validation
			AssertResult(t, tt, testFunction, got, err)
		})
	}
}

func TestCopyFile(t *testing.T) {
	testFunction := "CopyFile"

	type inputArgs struct {
		pathOrigin  string
		pathDestiny string
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return error if origin file cannot be opened",
			Input: inputArgs{
				pathOrigin:  "non_existent_origin.txt",
				pathDestiny: "destiny.txt",
			},
			WantErr: true,
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return p, nil
				})
			},
		},
		{
			Name: "Should return error if destiny file cannot be opened",
			Input: inputArgs{
				pathOrigin:  "origin.txt",
				pathDestiny: "protected_destiny/destiny.txt",
			},
			WantErr: true,
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					if p == "origin.txt" {
						tmpOrigin := filepath.Join(t.TempDir(), "origin.txt")
						_ = os.WriteFile(tmpOrigin, []byte("data"), 0o644)
						return tmpOrigin, nil
					}
					return "/invalid/protected/path/destiny.txt", nil
				})
			},
		},
		{
			Name: "Should return error if io.Copy operation fails",
			Input: inputArgs{
				pathOrigin:  "origin.txt",
				pathDestiny: "destiny.txt",
			},
			WantErr: true,
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					tmpDir := t.TempDir()
					if p == "origin.txt" {
						tmpOrigin := filepath.Join(tmpDir, "origin.txt")
						_ = os.WriteFile(tmpOrigin, []byte("data"), 0o644)
						return tmpOrigin, nil
					}
					return filepath.Join(tmpDir, "destiny.txt"), nil
				})
				m.OnCall.IOCopy(func(dst io.Writer, src io.Reader) (int64, error) {
					return 0, errors.New("mocked io.Copy failure")
				})
			},
		},
		{
			Name: "Should succeed, mirror exact content into destination file and call sync",
			Input: inputArgs{
				pathOrigin:  "origin.txt",
				pathDestiny: "destiny.txt",
			},
			WantErr: false,
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					tmpDir := t.TempDir()
					if p == "origin.txt" {
						tmpOrigin := filepath.Join(tmpDir, "origin.txt")
						_ = os.WriteFile(tmpOrigin, []byte("hello go world"), 0o644)
						return tmpOrigin, nil
					}
					return filepath.Join(tmpDir, "destiny.txt"), nil
				})
				// Garante que o IOCopy mockado passe sem erros para atingir o Sync com sucesso
				m.OnCall.IOCopy(func(dst io.Writer, src io.Reader) (int64, error) {
					return io.Copy(dst, src)
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			defer SetupTestMock(t, tt)()
			args := tt.Input.(inputArgs)
			err := xfs.CopyFile(args.pathOrigin, args.pathDestiny)
			AssertResult(t, tt, testFunction, nil, err)
		})
	}
}

func TestDeleteFile(t *testing.T) {
	testFunction := "DeleteFile"

	type inputArgs struct {
		path string
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return error if RetrieveFullPath fails",
			Input: inputArgs{
				path: "invalid/path",
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: XFS][MSG: failed to resolve user home directory]",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return "", errors.New("[CTX: XFS][MSG: failed to resolve user home directory]")
				})
			},
		},
		{
			Name: "Should return error if target path is a directory",
			Input: inputArgs{
				path: "~/documents",
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: ERR_XFS][ERR: E3005][MSG: cannot use DeleteFile on a directory]",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/documents", nil)
				m.OnCall.IsDir(func(p string) bool {
					return true
				})
			},
		},
		{
			Name: "Should return error if os.Remove fails (file protected or missing)",
			Input: inputArgs{
				path: "~/locked.txt",
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: ERR_XFS][ERR: E1008][MSG: failed to delete file at targeted path]",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/locked.txt", nil)
				m.OnCall.IsDir(func(p string) bool {
					return false
				})
				m.OnCall.OsRemove(func(name string) error {
					return os.ErrPermission
				})
			},
		},
		{
			Name: "Should succeed and return nil when file is deleted",
			Input: inputArgs{
				path: "~/temporary.txt",
			},
			Want:        nil,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/temporary.txt", nil)
				m.OnCall.IsDir(func(p string) bool {
					return false
				})
				m.OnCall.OsRemove(func(name string) error {
					return nil
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			err := xfs.DeleteFile(args.path)

			// Validation
			AssertResult(t, tt, testFunction, nil, err)
		})
	}
}

func TestDeleteDir(t *testing.T) {
	testFunction := "DeleteDir"

	type inputArgs struct {
		path      string
		recursive bool
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return error if RetrieveFullPath fails",
			Input: inputArgs{
				path:      "invalid/path",
				recursive: false,
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: XFS][MSG: failed to resolve user home directory]",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return "", errors.New("[CTX: XFS][MSG: failed to resolve user home directory]")
				})
			},
		},
		{
			Name: "Should return error if target path is not a directory",
			Input: inputArgs{
				path:      "~/notes.txt",
				recursive: false,
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: ERR_XFS][ERR: E3005][MSG: path is not a directory or does not exist]",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/notes.txt", nil)
				m.OnCall.IsDir(func(p string) bool {
					return false
				})
			},
		},
		{
			Name: "Non-Recursive: Should return error if os.Remove fails (directory not empty)",
			Input: inputArgs{
				path:      "~/full-dir",
				recursive: false,
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: ERR_XFS][ERR: E1008][MSG: failed to delete targeted directory hierarchy; ('recursive'=false)]",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/full-dir", nil)
				m.OnCall.IsDir(func(p string) bool {
					return true
				})
				m.OnCall.OsRemove(func(name string) error {
					return errors.New("directory not empty")
				})
			},
		},
		{
			Name: "Non-Recursive: Should succeed if directory is empty",
			Input: inputArgs{
				path:      "~/empty-dir",
				recursive: false,
			},
			Want:        nil,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/empty-dir", nil)
				m.OnCall.IsDir(func(p string) bool {
					return true
				})
				m.OnCall.OsRemove(func(name string) error {
					return nil
				})
			},
		},
		{
			Name: "Recursive: Should return error if os.RemoveAll fails (permission denied)",
			Input: inputArgs{
				path:      "~/protected-tree",
				recursive: true,
			},
			Want:        nil,
			WantErr:     true,
			ErrContains: "[CTX: ERR_XFS][ERR: E1008][MSG: failed to delete targeted directory hierarchy; ('recursive'=true)]",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/protected-tree", nil)
				m.OnCall.IsDir(func(p string) bool {
					return true
				})
				m.OnCall.OsRemoveAll(func(path string) error {
					return os.ErrPermission
				})
			},
		},
		{
			Name: "Recursive: Should succeed deleting the entire directory tree",
			Input: inputArgs{
				path:      "~/nested-tree",
				recursive: true,
			},
			Want:        nil,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/nested-tree", nil)
				m.OnCall.IsDir(func(p string) bool {
					return true
				})
				m.OnCall.OsRemoveAll(func(path string) error {
					return nil
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			err := xfs.DeleteDir(args.path, args.recursive)

			// Validation
			AssertResult(t, tt, testFunction, nil, err)
		})
	}
}

func TestHasSpaceAvailable(t *testing.T) {
	testFunction := "HasSpaceAvailable"

	type inputArgs struct {
		path  string
		bytes uint64
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return error if RetrieveFullPath fails",
			Input: inputArgs{
				path:  "invalid/path",
				bytes: 1024,
			},
			Want:        false,
			WantErr:     true,
			ErrContains: "[CTX: XFS][MSG: failed to resolve user home directory]",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return "", errors.New("[CTX: XFS][MSG: failed to resolve user home directory]")
				})
			},
		},
		{
			Name: "Should return error if it loops up to the root and finds no valid path",
			Input: inputArgs{
				path:  "~/some/nested/path",
				bytes: 1024,
			},
			Want:        false,
			WantErr:     true,
			ErrContains: "[CTX: ERR_XFS][ERR: E4002][MSG: could not find a valid base volume path]",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/some/nested/path", nil)
				m.OnCall.Exists(func(p string) bool {
					return false
				})
			},
		},
		{
			Name: "Should break the loop immediately if the expanded path exists",
			Input: inputArgs{
				path:  "~/existing-dir",
				bytes: 500,
			},
			Want:        true,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/existing-dir", nil)
				m.OnCall.Exists(func(p string) bool {
					return p == "/mock/existing-dir"
				})
				m.OnCall.GetVolumeFreeSpace(func(path string, bytes uint64) (bool, error) {
					if path != "/mock/existing-dir" {
						return false, errors.New("expected validation track check failure")
					}
					return true, nil
				})
			},
		},
		{
			Name: "Should loop upwards until finding an existing parent directory",
			Input: inputArgs{
				path:  "/mock/root/dir/missing1/missing2",
				bytes: 2000,
			},
			Want:        true,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/root/dir/missing1/missing2", nil)
				m.OnCall.Exists(func(p string) bool {
					return p == filepath.FromSlash("/mock/root/dir")
				})
				m.OnCall.GetVolumeFreeSpace(func(path string, bytes uint64) (bool, error) {
					if path != filepath.FromSlash("/mock/root/dir") {
						return false, errors.New("expected validation track check failure")
					}
					return true, nil
				})
			},
		},
		{
			Name: "Should propagate errors returned by getVolumeFreeSpace function",
			Input: inputArgs{
				path:  "~/existing-dir",
				bytes: 100,
			},
			Want:        false,
			WantErr:     true,
			ErrContains: "syscall disk failure",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/existing-dir", nil)
				m.OnCall.Exists(func(p string) bool {
					return true
				})
				m.OnCall.GetVolumeFreeSpace(func(path string, bytes uint64) (bool, error) {
					return false, errors.New("syscall disk failure")
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			got, err := xfs.HasSpaceAvailable(args.path, args.bytes)

			// Validation
			AssertResult(t, tt, testFunction, got, err)
		})
	}
}

func TestGetFileSize(t *testing.T) {
	testFunction := "GetFileSize"

	type inputArgs struct {
		path string
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return error if RetrieveFullPath fails",
			Input: inputArgs{
				path: "invalid/path",
			},
			Want:        int64(0),
			WantErr:     true,
			ErrContains: "mocked err",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return "", errors.New("mocked err")
				})
			},
		},
		{
			Name: "Should return error if os.Stat fails (file not found)",
			Input: inputArgs{
				path: "~/missing-file.txt",
			},
			Want:        int64(0),
			WantErr:     true,
			ErrContains: "file does not exist",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/missing-file.txt", nil)
				m.SetReturn.OsStat(nil, os.ErrNotExist)
			},
		},
		{
			Name: "Should return os.ErrInvalid if the path targets a directory",
			Input: inputArgs{
				path: "~/documents",
			},
			Want:        int64(0),
			WantErr:     true,
			ErrContains: "invalid argument",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/documents", nil)
				m.SetReturn.OsStat(mockFileInfo{isRegular: false, isDir: true}, nil)
			},
		},
		{
			Name: "Should return the correct file size on success",
			Input: inputArgs{
				path: "~/notes.txt",
			},
			Want:        int64(4096),
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("/mock/notes.txt", nil)
				m.SetReturn.OsStat(mockFileInfo{isRegular: true, size: 4096}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			got, err := xfs.GetFileSize(args.path)

			// Validation
			AssertResult(t, tt, testFunction, got, err)
		})
	}
}

func TestIsSameFile(t *testing.T) {
	testFunction := "IsSameFile"

	type inputArgs struct {
		pathA string
		pathB string
	}

	tests := []xmock.TestCaseXFS{
		{
			Name: "Should return error if RetrieveFullPath fails for pathA",
			Input: inputArgs{
				pathA: "invalid/path",
				pathB: "valid/path",
			},
			Want:        false,
			WantErr:     true,
			ErrContains: "mocked err A",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					if p == "invalid/path" {
						return "", errors.New("mocked err A")
					}
					return "/mock/valid", nil
				})
			},
		},
		{
			Name: "Should return error if RetrieveFullPath fails for pathB",
			Input: inputArgs{
				pathA: "valid/path",
				pathB: "invalid/path",
			},
			Want:        false,
			WantErr:     true,
			ErrContains: "mocked err B",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					if p == "invalid/path" {
						return "", errors.New("mocked err B")
					}
					return "/mock/valid", nil
				})
			},
		},
		{
			Name: "Should return error if os.Stat fails for pathA",
			Input: inputArgs{
				pathA: "~/missing1.txt",
				pathB: "~/notes.txt",
			},
			Want:        false,
			WantErr:     true,
			ErrContains: "file does not exist",
			MockFn: func(m *xmock.MockXFS) {
				m.SetReturn.RetrieveFullPath("", nil)
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return p, nil
				})
				m.OnCall.OsStat(func(n string) (os.FileInfo, error) {
					if n == "~/missing1.txt" {
						return nil, os.ErrNotExist
					}
					tmp, _ := os.CreateTemp(t.TempDir(), "same_file_stub")
					defer tmp.Close()
					return tmp.Stat()
				})
			},
		},
		{
			Name: "Should return error if os.Stat fails for pathB",
			Input: inputArgs{
				pathA: "~/notes.txt",
				pathB: "~/missing2.txt",
			},
			Want:        false,
			WantErr:     true,
			ErrContains: "file does not exist",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return p, nil
				})
				m.OnCall.OsStat(func(n string) (os.FileInfo, error) {
					if n == "~/missing2.txt" {
						return nil, os.ErrNotExist
					}
					tmp, _ := os.CreateTemp(t.TempDir(), "same_file_stub")
					defer tmp.Close()
					return tmp.Stat()
				})
			},
		},
		{
			Name: "Should return true if both paths point to the exact same file",
			Input: inputArgs{
				pathA: "~/notes.txt",
				pathB: "~/notes_alias.txt",
			},
			Want:        true,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return p, nil
				})
				tmpFile, _ := os.CreateTemp(t.TempDir(), "shared_inode_stub")
				info, _ := tmpFile.Stat()
				_ = tmpFile.Close()
				m.OnCall.OsStat(func(n string) (os.FileInfo, error) {
					return info, nil
				})
			},
		},
		{
			Name: "Should return false if paths point to different files",
			Input: inputArgs{
				pathA: "~/notes.txt",
				pathB: "~/documents",
			},
			Want:        false,
			WantErr:     false,
			ErrContains: "",
			MockFn: func(m *xmock.MockXFS) {
				m.OnCall.RetrieveFullPath(func(p string) (string, error) {
					return p, nil
				})
				tmpA, _ := os.CreateTemp(t.TempDir(), "file_a")
				tmpB, _ := os.CreateTemp(t.TempDir(), "file_b")
				infoA, _ := tmpA.Stat()
				infoB, _ := tmpB.Stat()
				_ = tmpA.Close()
				_ = tmpB.Close()

				m.OnCall.OsStat(func(n string) (os.FileInfo, error) {
					if n == "~/documents" {
						return infoB, nil
					}
					return infoA, nil
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Configure mocks
			defer SetupTestMock(t, tt)()

			// Unpacks the typed parameters from the generic input object safely
			args := tt.Input.(inputArgs)

			// Trigger the business implementation
			got, err := xfs.IsSameFile(args.pathA, args.pathB)

			// Validation
			AssertResult(t, tt, testFunction, got, err)
		})
	}
}
