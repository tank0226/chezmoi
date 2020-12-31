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

// Glob implements System.Glob.
func (s *RealSystem) Glob(pattern string) ([]string, error) {
	return doublestar.GlobOS(doubleStarOS{FS: s}, pattern)
}

// IdempotentCmdOutput implements System.IdempotentCmdOutput.
func (s *RealSystem) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	return cmd.Output()
}

// PathSeparator implements doublestar.OS.PathSeparator.
func (s *RealSystem) PathSeparator() rune {
	return '/'
}

// RunCmd implements System.RunCmd.
func (s *RealSystem) RunCmd(cmd *exec.Cmd) error {
	return cmd.Run()
}

// RunScript implements System.RunScript.
func (s *RealSystem) RunScript(scriptname, dir string, data []byte) (err error) {
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
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = s.RunCmd(cmd)
	return
}

// UnderlyingFS implements System.UnderlyingFS.
func (s *RealSystem) UnderlyingFS() vfs.FS {
	return s.FS
}
