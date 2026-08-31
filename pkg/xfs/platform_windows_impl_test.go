//go:build windows

package xfs_test

import (
	"testing"

	"github.com/AeonDigital/Go-Core-xfs/pkg/xfs"
	"golang.org/x/sys/windows"
)

func TestGetVolumeFreeSpace_Sufficient(t *testing.T) {
	restore := xfs.ExportSetStatfsFunc(func(path string, buf interface{}) error {
		b := buf.(*xfs.DiskFreeSpaceBuf)
		*b.FreeBytesAvailable = 100
		*b.TotalNumberOfBytes = 1024
		*b.TotalNumberOfFreeBytes = 100
		return nil
	})
	defer restore()

	ok, err := xfs.PkgBridgeXFS{}.GetVolumeFreeSpace("C:\\", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected sufficient space, got false")
	}
}

func TestGetVolumeFreeSpace_Insufficient(t *testing.T) {
	restore := xfs.ExportSetStatfsFunc(func(path string, buf interface{}) error {
		b := buf.(*xfs.DiskFreeSpaceBuf)
		*b.FreeBytesAvailable = 10
		*b.TotalNumberOfBytes = 1024
		*b.TotalNumberOfFreeBytes = 10
		return nil
	})
	defer restore()

	ok, err := xfs.PkgBridgeXFS{}.GetVolumeFreeSpace("C:\\", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected insufficient space, got true")
	}
}

func TestGetVolumeFreeSpace_Error(t *testing.T) {
	restore := xfs.ExportSetStatfsFunc(func(path string, buf interface{}) error {
		return windows.ERROR_ACCESS_DENIED
	})
	defer restore()

	_, err := xfs.PkgBridgeXFS{}.GetVolumeFreeSpace("C:\\", 1)
	if err == nil {
		t.Fatalf("expected error from disk free space call, got nil")
	}
}

func TestExportSetStatfsFunc_NilBranch(t *testing.T) {
	restoreCustom := xfs.ExportSetStatfsFunc(func(path string, buf interface{}) error {
		b := buf.(*xfs.DiskFreeSpaceBuf)
		*b.FreeBytesAvailable = 0
		*b.TotalNumberOfBytes = 0
		*b.TotalNumberOfFreeBytes = 0
		return nil
	})
	if restoreCustom == nil {
		t.Fatal("expected non-nil restore from custom setter")
	}
	defer restoreCustom()

	restoreNil := xfs.ExportSetStatfsFunc(nil)
	if restoreNil == nil {
		t.Fatal("expected non-nil restore from nil setter")
	}

	restoreNil()
}
