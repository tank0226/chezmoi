package chezmoi

import (
	"fmt"
	"path/filepath"
	"strings"
)

// NewAbsPath returns a new AbsPath.
func NewAbsPath(path string) (AbsPath, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("%s: not an absolute path")
	}
	return AbsPath(path), nil
}

// NewAbsPathFromExtPath returns a new AbsPath by converting extPath to use
// slashes, tilde expansion, making the path absolute, and converting the volume
// name to uppercase.
func NewAbsPathFromExtPath(extPath string, homeDirAbsPath AbsPath) (AbsPath, error) {
	tildeSlashPath := expandTilde(filepath.ToSlash(extPath), homeDirAbsPath)
	if filepath.IsAbs(tildeSlashPath) {
		return AbsPath(tildeSlashPath), nil
	}
	slashPathAbsPath, err := filepath.Abs(tildeSlashPath)
	if err != nil {
		return "", err
	}
	if n := volumeNameLen(slashPathAbsPath); n > 0 {
		slashPathAbsPath = strings.ToUpper(slashPathAbsPath[:n]) + slashPathAbsPath[n:]
	}
	return AbsPath(slashPathAbsPath), nil
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

// expandTilde expands a leading tilde in path.
func expandTilde(path string, homeDirAbsPath AbsPath) string {
	switch {
	case path == "~":
		return string(homeDirAbsPath)
	case len(path) >= 2 && path[0] == '~' && isSlash(path[1]):
		return string(homeDirAbsPath.Join(RelPath(path[2:])))
	default:
		return path
	}
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
