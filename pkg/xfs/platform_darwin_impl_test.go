//go:build darwin

package xfs_test

import (
	"testing"

	"github.com/AeonDigital/Go-Core-xfs/pkg/xfs"
	"golang.org/x/sys/unix"
)

func TestGetVolumeFreeSpace_Sufficient(t *testing.T) {
	restore := xfs.ExportSetStatfsFunc(func(path string, buf interface{}) error {
		s := buf.(*unix.Statfs_t)
		s.Bavail = 100
		s.Bsize = 1024
		return nil
	})
	defer restore()

	ok, err := xfs.PkgBridgeXFS{}.GetVolumeFreeSpace("/any", 100*1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected sufficient space, got false")
	}
}

func TestGetVolumeFreeSpace_Insufficient(t *testing.T) {
	restore := xfs.ExportSetStatfsFunc(func(path string, buf interface{}) error {
		s := buf.(*unix.Statfs_t)
		s.Bavail = 10
		s.Bsize = 1024
		return nil
	})
	defer restore()

	ok, err := xfs.PkgBridgeXFS{}.GetVolumeFreeSpace("/any", 50*1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected insufficient space, got true")
	}
}

func TestGetVolumeFreeSpace_Error(t *testing.T) {
	restore := xfs.ExportSetStatfsFunc(func(path string, buf interface{}) error {
		return unix.Errno(1)
	})
	defer restore()

	_, err := xfs.PkgBridgeXFS{}.GetVolumeFreeSpace("/any", 1)
	if err == nil {
		t.Fatalf("expected error from statfs, got nil")
	}
}

func TestExportSetStatfsFunc_NilBranch(t *testing.T) {
	restoreCustom := xfs.ExportSetStatfsFunc(func(path string, buf interface{}) error {
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
