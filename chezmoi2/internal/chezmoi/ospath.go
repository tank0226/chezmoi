package chezmoi

import (
	"path/filepath"
)

// An OSPath is a native OS path.
type OSPath struct {
	s string
}

// NewOSPath returns a new OSPath.
func NewOSPath(s string) *OSPath {
	return &OSPath{
		s: filepath.FromSlash(s),
	}
}

// Dir returns p's directory.
func (p *OSPath) Dir() *OSPath {
	return &OSPath{
		s: filepath.Dir(p.s),
	}
}

// Empty returns if p is empty.
func (p *OSPath) Empty() bool {
	return p.s != ""
}

// Join joins elems on to p.
func (p *OSPath) Join(elems ...string) *OSPath {
	return &OSPath{
		s: filepath.Join(append([]string{p.s}, elems...)...),
	}
}

// MarshalText implements encoding.TextMarshaler.MarshalText.
func (p *OSPath) MarshalText() ([]byte, error) {
	return []byte(p.s), nil
}

// Normalize performs tilde expansion on p and returns the normalized result.
func (p *OSPath) Normalize(homeDirAbsPath AbsPath) (AbsPath, error) {
	return NormalizePath(ExpandTilde(p.s, homeDirAbsPath))
}

func (p *OSPath) String() string {
	return p.s
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText.
func (p *OSPath) UnmarshalText(data []byte) error {
	p.s = string(data)
	return nil
}
