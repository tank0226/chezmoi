package chezmoi

import (
	"os"
	"os/exec"
	"path/filepath"

	vfs "github.com/twpayne/go-vfs"
)

// A System reads from and writes to a filesystem, executes idempotent commands,
// runs scripts, and persists state.
type System interface {
	Chmod(name AbsPath, mode os.FileMode) error
	Glob(pattern string) ([]string, error)
	IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error)
	Lstat(filename AbsPath) (os.FileInfo, error)
	Mkdir(name AbsPath, perm os.FileMode) error
	RawPath(absPath AbsPath) (AbsPath, error)
	ReadDir(dirname AbsPath) ([]os.FileInfo, error)
	ReadFile(filename AbsPath) ([]byte, error)
	Readlink(name AbsPath) (string, error)
	RemoveAll(name AbsPath) error
	Rename(oldpath, newpath AbsPath) error
	RunCmd(cmd *exec.Cmd) error
	RunScript(scriptname string, dir AbsPath, data []byte) error
	Stat(name AbsPath) (os.FileInfo, error)
	UnderlyingFS() vfs.FS
	WriteFile(filename AbsPath, data []byte, perm os.FileMode) error
	WriteSymlink(oldname string, newname AbsPath) error
}

// A nullReaderSystem simulates an empty system.
type nullReaderSystem struct{}

func (nullReaderSystem) Glob(pattern string) ([]string, error)             { return nil, nil }
func (nullReaderSystem) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) { return cmd.Output() }
func (nullReaderSystem) Lstat(name AbsPath) (os.FileInfo, error)           { return nil, os.ErrNotExist }
func (nullReaderSystem) RawPath(path AbsPath) (AbsPath, error)             { return path, nil }
func (nullReaderSystem) ReadDir(dirname AbsPath) ([]os.FileInfo, error)    { return nil, os.ErrNotExist }
func (nullReaderSystem) ReadFile(filename AbsPath) ([]byte, error)         { return nil, os.ErrNotExist }
func (nullReaderSystem) Readlink(name AbsPath) (string, error)             { return "", os.ErrNotExist }
func (nullReaderSystem) Stat(name AbsPath) (os.FileInfo, error)            { return nil, os.ErrNotExist }

// MkdirAll is the equivalent of os.MkdirAll but operates on system.
func MkdirAll(s System, absPath AbsPath, perm os.FileMode) error {
	switch err := s.Mkdir(absPath, perm); {
	case err == nil:
		// Mkdir was successful.
		return nil
	case os.IsExist(err):
		// path already exists, but we don't know whether it's a directory or
		// something else. We get this error if we try to create a subdirectory
		// of a non-directory, for example if the parent directory of path is a
		// file. There's a race condition here between the call to Mkdir and the
		// call to Stat but we can't avoid it because there's not enough
		// information in the returned error from Mkdir. We need to distinguish
		// between "path already exists and is already a directory" and "path
		// already exists and is not a directory". Between the call to Mkdir and
		// the call to Stat path might have changed.
		info, statErr := s.Stat(absPath)
		if statErr != nil {
			return statErr
		}
		if !info.IsDir() {
			return err
		}
		return nil
	case os.IsNotExist(err):
		// Parent directory does not exist. Create the parent directory
		// recursively, then try again.
		parentDir := absPath.Dir()
		if parentDir == "/" || parentDir == "." {
			// We cannot create the root directory or the current directory, so
			// return the original error.
			return err
		}
		if err := MkdirAll(s, parentDir, perm); err != nil {
			return err
		}
		return s.Mkdir(absPath, perm)
	default:
		// Some other error.
		return err
	}
}

// Walk walks rootAbsPath in s.
func Walk(s System, rootAbsPath AbsPath, walkFn func(absPath AbsPath, info os.FileInfo, err error) error) error {
	// FIXME check path joining
	return vfs.Walk(s.UnderlyingFS(), string(rootAbsPath), func(absPath string, info os.FileInfo, err error) error {
		return walkFn(AbsPath(filepath.ToSlash(absPath)), info, err)
	})
}
