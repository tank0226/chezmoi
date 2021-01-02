package chezmoi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	vfs "github.com/twpayne/go-vfs"
)

// NewAbsPath returns a new AbsPath.
func NewAbsPath(path string) (AbsPath, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("%s: not an absolute path")
	}
	return AbsPath(path), nil
}

// ExpandTilde expands a leading tilde in path.
func ExpandTilde(path string, homeDirAbsPath AbsPath) string {
	switch {
	case path == "~":
		return homeDirAbsPath
	case len(path) >= 2 && path[0] == '~' && isSlash(path[1]):
		return filepath.ToSlash(homeDirAbsPath.Join(path[2:]).String())
	default:
		return path
	}
}

// FQDNHostname does nothing on Windows.
func FQDNHostname(fs vfs.FS) (string, error) {
	// LATER find out how to determine the FQDN hostname on Windows
	return "", nil
}

// GetUmask returns the umask.
func GetUmask() os.FileMode {
	return os.ModePerm
}

// NormalizePath returns path normalized. On Windows, normalized paths are
// absolute paths with a uppercase volume name and forward slashes.
func NormalizePath(path string) (AbsPath, error) {
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if n := volumeNameLen(path); n > 0 {
		path = strings.ToUpper(path[:n]) + path[n:]
	}
	return AbsPath(filepath.ToSlash(path)), nil
}

// SetUmask sets the umask.
func SetUmask(umask os.FileMode) {}

// TrimDirPrefix returns path with the directory prefix dir stripped. path must
// be an absolute path with forward slashes.
func TrimDirPrefix(path, dir string) (string, error) {
	prefix := strings.ToLower(dir + "/")
	if !strings.HasPrefix(strings.ToLower(path), prefix) {
		return "", fmt.Errorf("%q does not have dir prefix %q", path, dir)
	}
	return path[len(prefix):], nil
}

// isExecutable returns false on Windows.
func isExecutable(info os.FileInfo) bool {
	return false
}

// isPrivate returns false on Windows.
func isPrivate(info os.FileInfo) bool {
	return false
}

func isSlash(c uint8) bool {
	return c == '\\' || c == '/'
}

// umaskPermEqual returns true on Windows.
func umaskPermEqual(perm1 os.FileMode, perm2 os.FileMode, umask os.FileMode) bool {
	return true
}

// volumeNameLen returns length of the leading volume name on Windows. It
// returns 0 elsewhere.
func volumeNameLen(path string) int {
	if len(path) < 2 {
		return 0
	}
	// with drive letter
	c := path[0]
	if path[1] == ':' && ('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z') {
		return 2
	}
	// is it UNC? https://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
	if l := len(path); l >= 5 && isSlash(path[0]) && isSlash(path[1]) &&
		!isSlash(path[2]) && path[2] != '.' {
		// first, leading `\\` and next shouldn't be `\`. its server name.
		for n := 3; n < l-1; n++ {
			// second, next '\' shouldn't be repeated.
			if isSlash(path[n]) {
				n++
				// third, following something characters. its share name.
				if !isSlash(path[n]) {
					if path[n] == '.' {
						break
					}
					for ; n < l; n++ {
						if isSlash(path[n]) {
							break
						}
					}
					return n
				}
				break
			}
		}
	}
	return 0
}
