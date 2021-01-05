package chezmoitest

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/twpayne/go-vfs"
	"github.com/twpayne/go-vfs/vfst"
)

var gpgKeyMarkedAsUltimatelyTrustedRx = regexp.MustCompile(`gpg: key ([0-9A-F]+) marked as ultimately trusted`)

// JoinLines joins lines with newlines.
func JoinLines(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

// GPGQuickGenerateKey generates and returns a GPG key in homeDir.
func GPGQuickGenerateKey(homeDir string) (string, error) {
	output, err := exec.Command(
		"gpg",
		"--batch",
		"--homedir", homeDir,
		"--no-tty",
		"--passphrase", "chezmoi-test-passphrase",
		"--pinentry-mode", "loopback",
		"--quick-generate-key", "chezmoi-test-key",
	).CombinedOutput()
	if err != nil {
		return "", err
	}
	submatch := gpgKeyMarkedAsUltimatelyTrustedRx.FindSubmatch(output)
	if submatch == nil {
		return "", fmt.Errorf("key not found in %q", output)
	}
	return string(submatch[1]), nil
}

// HomeDir returns the home directory.
func HomeDir() string {
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
		t.Skip("skipping Windows test on UNIX")
	case strings.HasSuffix(name, "_unix") && runtime.GOOS == "windows":
		t.Skip("skipping UNIX test on Windows")
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
