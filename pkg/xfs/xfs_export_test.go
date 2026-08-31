package xfs

// SetBridgeXFSForTest overrides the internal package bridge during testing.
func SetBridgeXFSForTest(b IPkgBridgeXFS) func() {
	old := pkgBridgeXFS
	pkgBridgeXFS = b
	return func() {
		pkgBridgeXFS = old
	}
}
