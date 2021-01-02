package chezmoi

import (
	"os"
	"os/exec"

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
