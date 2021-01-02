package chezmoi

import (
	"path/filepath"
)

// An OSPath is a native OS path.
type OSPath string

// NewOSPath returns a new OSPath.
func NewOSPath(s string) OSPath {
	return OSPath(filepath.FromSlash(s))
}

// Dir returns p's directory.
func (p OSPath) Dir() OSPath {
	return OSPath(filepath.Dir(string(p)))
}

// Empty returns if p is empty.
func (p OSPath) Empty() bool {
	return p != ""
}

// Join joins elems on to p.
func (p OSPath) Join(elems ...string) OSPath {
	return OSPath(filepath.Join(append([]string{string(p)}, elems...)...))
}

// Normalize performs tilde expansion on p and returns the normalized result.
func (p OSPath) Normalize(homeDirAbsPath AbsPath) (AbsPath, error) {
	return NormalizePath(ExpandTilde(string(p), homeDirAbsPath))
}

func (p OSPath) String() string {
	return string(p)
}
