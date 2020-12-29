// +build !windows

package chezmoi

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
)

var umask os.FileMode

func init() {
	umask = os.FileMode(syscall.Umask(0))
	syscall.Umask(int(umask))
}

// ExpandTilde expands a leading tilde in path.
func ExpandTilde(path, homeDir string) string {
	switch {
	case path == "~":
		return homeDir
	case strings.HasPrefix(path, "~/"):
		return filepath.Clean(filepath.Join(homeDir, path[2:]))
	default:
		return path
	}
}

// GetUmask returns the umask.
func GetUmask() os.FileMode {
	return umask
}

// NormalizePath returns path normalized. On non-Windows systems, normalized
// paths are absolute paths.
func NormalizePath(path string) (string, error) {
	return filepath.Abs(path)
}

// SetUmask sets the umask.
func SetUmask(newUmask os.FileMode) {
	umask = newUmask
	syscall.Umask(int(umask))
}

// TrimDirPrefix returns path p with the directory prefix dir stripped. path must
// be an absolute path with forward slashes.
func TrimDirPrefix(p, dir string) (string, error) {
	switch {
	case !path.IsAbs(p):
		return "", fmt.Errorf("%s: not an absolute path", p)
	case !strings.HasPrefix(p, dir+"/"):
		return "", &notInDirError{
			path: p,
			dir:  dir,
		}
	default:
		return p[len(dir)+1:], nil
	}
}

// isExecutable returns if info is executable.
func isExecutable(info os.FileInfo) bool {
	return info.Mode().Perm()&0o111 != 0
}

// isPrivate returns if info is private.
func isPrivate(info os.FileInfo) bool {
	return info.Mode().Perm()&0o77 == 0
}

// umaskPermEqual returns if two permissions are equal after applying umask.
func umaskPermEqual(perm1, perm2, umask os.FileMode) bool {
	return perm1&^umask == perm2&^umask
}
