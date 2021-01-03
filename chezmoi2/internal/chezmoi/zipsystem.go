package chezmoi

import (
	"archive/zip"
	"io"
	"os"
	"os/exec"
	"time"

	vfs "github.com/twpayne/go-vfs"
)

// A ZIPSystem is a System that writes to a ZIP archive.
type ZIPSystem struct {
	nullReaderSystem
	w        *zip.Writer
	modified time.Time
}

// NewZIPSystem returns a new ZIPSystem that writes a ZIP archive to w.
func NewZIPSystem(w io.Writer, modified time.Time) *ZIPSystem {
	return &ZIPSystem{
		w:        zip.NewWriter(w),
		modified: modified,
	}
}

// Chmod implements System.Chmod.
func (s *ZIPSystem) Chmod(name AbsPath, mode os.FileMode) error {
	return os.ErrPermission
}

// Close closes m.
func (s *ZIPSystem) Close() error {
	return s.w.Close()
}

// Mkdir implements System.Mkdir.
func (s *ZIPSystem) Mkdir(name AbsPath, perm os.FileMode) error {
	fh := zip.FileHeader{
		Name:     string(name),
		Modified: s.modified,
	}
	fh.SetMode(os.ModeDir | perm)
	_, err := s.w.CreateHeader(&fh)
	return err
}

// RemoveAll implements System.RemoveAll.
func (s *ZIPSystem) RemoveAll(name AbsPath) error {
	return os.ErrPermission
}

// Rename implements System.Rename.
func (s *ZIPSystem) Rename(oldpath, newpath AbsPath) error {
	return os.ErrPermission
}

// RunCmd implements System.RunCmd.
func (s *ZIPSystem) RunCmd(cmd *exec.Cmd) error {
	return nil
}

// RunScript implements System.RunScript.
func (s *ZIPSystem) RunScript(scriptname RelPath, dir AbsPath, data []byte) error {
	return s.WriteFile(AbsPath(scriptname), data, 0o700)
}

// UnderlyingFS implements System.UnderlyingFS.
func (s *ZIPSystem) UnderlyingFS() vfs.FS {
	return nil
}

// WriteFile implements System.WriteFile.
func (s *ZIPSystem) WriteFile(filename AbsPath, data []byte, perm os.FileMode) error {
	fh := zip.FileHeader{
		Name:               string(filename),
		Method:             zip.Deflate,
		Modified:           s.modified,
		UncompressedSize64: uint64(len(data)),
	}
	fh.SetMode(perm)
	fw, err := s.w.CreateHeader(&fh)
	if err != nil {
		return err
	}
	_, err = fw.Write(data)
	return err
}

// WriteSymlink implements System.WriteSymlink.
func (s *ZIPSystem) WriteSymlink(oldname string, newname AbsPath) error {
	data := []byte(oldname)
	fh := zip.FileHeader{
		Name:               string(newname),
		Modified:           s.modified,
		UncompressedSize64: uint64(len(data)),
	}
	fh.SetMode(os.ModeSymlink)
	fw, err := s.w.CreateHeader(&fh)
	if err != nil {
		return err
	}
	_, err = fw.Write(data)
	return err
}
