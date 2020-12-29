package chezmoitest

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/twpayne/go-vfs"
	"github.com/twpayne/go-vfs/vfst"
)

// JoinLines joins lines with newlines.
func JoinLines(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

// NormalizedHomeDir returns the normalized home directory.
func NormalizedHomeDir() string {
	switch runtime.GOOS {
	case "windows":
		return "C:/home/user"
	default:
		return "/home/user"
	}
}

// SkipUnlessGOOS calls t.Skip() if name does not match runtime.GOOS.
func SkipUnlessGOOS(t *testing.T, name string) {
	t.Helper()
	switch {
	case strings.HasSuffix(name, "_windows") && runtime.GOOS != "windows":
		t.Skip("skipping Windows-only test")
	case strings.HasSuffix(name, "_unix") && runtime.GOOS == "windows":
		t.Skip("skipping non-Windows test")
	}
}

// WithTestFS calls f with a test filesystem populated with root.
func WithTestFS(t *testing.T, root interface{}, f func(fs vfs.FS)) {
	t.Helper()
	fs, cleanup, err := vfst.NewTestFS(root)
	require.NoError(t, err)
	t.Cleanup(cleanup)
	f(fs)
}
