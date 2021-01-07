package chezmoi

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"

	"github.com/bmatcuk/doublestar/v3"
	vfs "github.com/twpayne/go-vfs"
	"go.uber.org/multierr"
)

// Chmod implements System.Chmod.
func (s *RealSystem) Chmod(name AbsPath, mode os.FileMode) error {
	return s.FS.Chmod(string(name), mode)
}

// Glob implements System.Glob.
func (s *RealSystem) Glob(pattern string) ([]string, error) {
	return doublestar.GlobOS(doubleStarOS{FS: s.UnderlyingFS()}, pattern)
}

// IdempotentCmdOutput implements System.IdempotentCmdOutput.
func (s *RealSystem) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	return cmd.Output()
}

// Lstat implements System.Lstat.
func (s *RealSystem) Lstat(filename AbsPath) (os.FileInfo, error) {
	return s.FS.Lstat(string(filename))
}

// Mkdir implements System.Mkdir.
func (s *RealSystem) Mkdir(name AbsPath, perm os.FileMode) error {
	return s.FS.Mkdir(string(name), perm)
}

// PathSeparator implements doublestar.OS.PathSeparator.
func (s *RealSystem) PathSeparator() rune {
	return '/'
}

// RawPath implements System.RawPath.
func (s *RealSystem) RawPath(absPath AbsPath) (AbsPath, error) {
	rawAbsPath, err := s.FS.RawPath(string(absPath))
	if err != nil {
		return "", err
	}
	return AbsPath(rawAbsPath), nil
}

// ReadDir implements System.ReadDir.
func (s *RealSystem) ReadDir(dirname AbsPath) ([]os.FileInfo, error) {
	return s.FS.ReadDir(string(dirname))
}

// ReadFile implements System.ReadFile.
func (s *RealSystem) ReadFile(filename AbsPath) ([]byte, error) {
	return s.FS.ReadFile(string(filename))
}

// Readlink implements System.Readlink.
func (s *RealSystem) Readlink(name AbsPath) (string, error) {
	return s.FS.Readlink(string(name))
}

// RemoveAll implements System.RemoveAll.
func (s *RealSystem) RemoveAll(name AbsPath) error {
	return s.FS.RemoveAll(string(name))
}

// Rename implements System.Rename.
func (s *RealSystem) Rename(oldpath, newpath AbsPath) error {
	return s.FS.Rename(string(oldpath), string(newpath))
}

// RunCmd implements System.RunCmd.
func (s *RealSystem) RunCmd(cmd *exec.Cmd) error {
	return cmd.Run()
}

// RunScript implements System.RunScript.
func (s *RealSystem) RunScript(scriptname string, dir AbsPath, data []byte) (err error) {
	// Write the temporary script file. Put the randomness at the front of the
	// filename to preserve any file extension for Windows scripts.
	f, err := ioutil.TempFile("", "*."+path.Base(scriptname))
	if err != nil {
		return
	}
	defer func() {
		err = multierr.Append(err, os.RemoveAll(f.Name()))
	}()

	// Make the script private before writing it in case it contains any
	// secrets.
	if runtime.GOOS != "windows" {
		if err = f.Chmod(0o700); err != nil {
			return
		}
	}
	_, err = f.Write(data)
	err = multierr.Append(err, f.Close())
	if err != nil {
		return
	}

	// Run the temporary script file.
	//nolint:gosec
	cmd := exec.Command(f.Name())
	cmd.Dir = string(dir)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = s.RunCmd(cmd)
	return
}

// Stat implements System.Stat.
func (s *RealSystem) Stat(name AbsPath) (os.FileInfo, error) {
	return s.FS.Stat(string(name))
}

// UnderlyingFS implements System.UnderlyingFS.
func (s *RealSystem) UnderlyingFS() vfs.FS {
	return s.FS
}
