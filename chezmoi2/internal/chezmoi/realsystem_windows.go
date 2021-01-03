package chezmoi

import (
	"os"
	"path/filepath"

	vfs "github.com/twpayne/go-vfs"
)

// An RealSystem is a System that writes to a filesystem and executes scripts.
type RealSystem struct {
	vfs.FS
}

// NewRealSystem returns a System that acts on fs.
func NewRealSystem(fs vfs.FS) *RealSystem {
	return &RealSystem{
		FS: fs,
	}
}

// Chmod implements System.Chmod.
func (s *RealSystem) Chmod(name AbsPath, mode os.FileMode) error {
	return nil
}

// Readlink implements System.Readlink.
func (s *RealSystem) Readlink(name AbsPath) (string, error) {
	linkname, err := s.FS.Readlink(string(name))
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(linkname), nil
}

// WriteSymlink implements System.WriteSymlink.
func (s *RealSystem) WriteSymlink(oldname string, newname AbsPath) error {
	if err := s.FS.RemoveAll(string(newname)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return s.FS.Symlink(filepath.FromSlash(oldname), string(newname))
}
