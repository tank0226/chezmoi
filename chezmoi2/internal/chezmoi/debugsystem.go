package chezmoi

import (
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	vfs "github.com/twpayne/go-vfs"
)

// A DebugSystem wraps a System and logs all of the actions it executes.
type DebugSystem struct {
	system System
	logger zerolog.Logger
}

// NewDebugSystem returns a new DebugSystem.
func NewDebugSystem(system System, logger zerolog.Logger) *DebugSystem {
	return &DebugSystem{
		system: system,
		logger: logger,
	}
}

// Chmod implements System.Chmod.
func (s *DebugSystem) Chmod(name AbsPath, mode os.FileMode) error {
	err := s.system.Chmod(name, mode)
	s.logger.Debug().
		Str("name", string(name)).
		Int("mode", int(mode)).
		Err(err).
		Msg("Chmod")
	return err
}

// Glob implements System.Glob.
func (s *DebugSystem) Glob(name string) ([]string, error) {
	matches, err := s.system.Glob(name)
	s.logger.Debug().
		Str("name", name).
		Strs("matches", matches).
		Err(err).
		Msg("Glob")
	return matches, err
}

// IdempotentCmdOutput implements System.IdempotentCmdOutput.
func (s *DebugSystem) IdempotentCmdOutput(cmd *exec.Cmd) ([]byte, error) {
	type result struct {
		startTime time.Time
		output    []byte
		err       error
	}

	resultCh := make(chan result)
	go func(resultCh chan<- result) {
		defer close(resultCh)
		start := time.Now().UTC()
		output, err := s.system.IdempotentCmdOutput(cmd)
		resultCh <- result{
			startTime: start,
			output:    output,
			err:       err,
		}
	}(resultCh)

	var r result
	select {
	case r = <-resultCh:
	case <-time.After(1 * time.Second):
		s.logger.Debug().
			Dict("cmd", cmdDict(cmd)).
			Msg("IdempotentCmdOutput")
		r = <-resultCh
	}

	s.logger.Debug().
		Dict("cmd", cmdDict(cmd)).
		Str("output", firstFewBytes(r.output)).
		Err(r.err).
		Dur("duration", time.Since(r.startTime)).
		Msg("IdempotentCmdOutput")

	return r.output, r.err
}

// Lstat implements System.Lstat.
func (s *DebugSystem) Lstat(name AbsPath) (os.FileInfo, error) {
	info, err := s.system.Lstat(name)
	s.logger.Debug().
		Str("name", string(name)).
		Err(err).
		Msg("Lstat")
	return info, err
}

// Mkdir implements System.Mkdir.
func (s *DebugSystem) Mkdir(name AbsPath, perm os.FileMode) error {
	err := s.system.Mkdir(name, perm)
	s.logger.Debug().
		Str("name", string(name)).
		Int("perm", int(perm)).
		Err(err).
		Msg("Mkdir")
	return err
}

// RawPath implements System.RawPath.
func (s *DebugSystem) RawPath(path AbsPath) (AbsPath, error) {
	return s.system.RawPath(path)
}

// ReadDir implements System.ReadDir.
func (s *DebugSystem) ReadDir(name AbsPath) ([]os.FileInfo, error) {
	infos, err := s.system.ReadDir(name)
	s.logger.Debug().
		Str("name", string(name)).
		Err(err).
		Msg("ReadDir")
	return infos, err
}

// ReadFile implements System.ReadFile.
func (s *DebugSystem) ReadFile(filename AbsPath) ([]byte, error) {
	data, err := s.system.ReadFile(filename)
	s.logger.Debug().
		Str("filename", string(filename)).
		Str("data", firstFewBytes(data)).
		Err(err).
		Msg("ReadFile")
	return data, err
}

// Readlink implements System.Readlink.
func (s *DebugSystem) Readlink(name AbsPath) (string, error) {
	linkname, err := s.system.Readlink(name)
	s.logger.Debug().
		Str("name", string(name)).
		Str("linkname", linkname).
		Err(err).
		Msg("Readlink")
	return linkname, err
}

// RemoveAll implements System.RemoveAll.
func (s *DebugSystem) RemoveAll(name AbsPath) error {
	err := s.system.RemoveAll(name)
	s.logger.Debug().
		Str("name", string(name)).
		Err(err).
		Msg("RemoveAll")
	return err
}

// Rename implements System.Rename.
func (s *DebugSystem) Rename(oldpath, newpath AbsPath) error {
	err := s.system.Rename(oldpath, newpath)
	s.logger.Debug().
		Str("oldpath", string(oldpath)).
		Str("newpath", string(newpath)).
		Err(err).
		Msg("Rename")
	return err
}

// RunCmd implements System.RunCmd.
func (s *DebugSystem) RunCmd(cmd *exec.Cmd) error {
	type result struct {
		startTime time.Time
		err       error
	}

	resultCh := make(chan result)
	go func(resultCh chan<- result) {
		defer close(resultCh)
		start := time.Now().UTC()
		err := s.system.RunCmd(cmd)
		resultCh <- result{
			startTime: start,
			err:       err,
		}
	}(resultCh)

	var r result
	select {
	case r = <-resultCh:
	case <-time.After(1 * time.Second):
		s.logger.Debug().
			Dict("cmd", cmdDict(cmd)).
			Msg("RunCmd")
		r = <-resultCh
	}

	s.logger.Debug().
		Dict("cmd", cmdDict(cmd)).
		Err(r.err).
		Dur("duration", time.Since(r.startTime)).
		Msg("RunCmd")

	return r.err
}

// RunScript implements System.RunScript.
func (s *DebugSystem) RunScript(scriptname RelPath, dir AbsPath, data []byte) error {
	type result struct {
		startTime time.Time
		err       error
	}

	resultCh := make(chan result)
	go func(resultCh chan<- result) {
		defer close(resultCh)
		start := time.Now().UTC()
		err := s.system.RunScript(scriptname, dir, data)
		resultCh <- result{
			startTime: start,
			err:       err,
		}
	}(resultCh)

	var r result
	select {
	case r = <-resultCh:
	case <-time.After(1 * time.Second):
		s.logger.Debug().
			Str("scriptname", string(scriptname)).
			Str("dir", string(dir)).
			Str("data", firstFewBytes(data)).
			Msg("RunScript")
		r = <-resultCh
	}

	s.logger.Debug().
		Str("scriptname", string(scriptname)).
		Str("dir", string(dir)).
		Str("data", firstFewBytes(data)).
		Err(r.err).
		Dur("duration", time.Since(r.startTime)).
		Msg("RunScript")

	return r.err
}

// Stat implements System.Stat.
func (s *DebugSystem) Stat(name AbsPath) (os.FileInfo, error) {
	info, err := s.system.Stat(name)
	s.logger.Debug().
		Str("name", string(name)).
		Err(err).
		Msg("Stat")
	return info, err
}

// UnderlyingFS implements System.UnderlyingFS.
func (s *DebugSystem) UnderlyingFS() vfs.FS {
	return s.system.UnderlyingFS()
}

// WriteFile implements System.WriteFile.
func (s *DebugSystem) WriteFile(name AbsPath, data []byte, perm os.FileMode) error {
	err := s.system.WriteFile(name, data, perm)
	s.logger.Debug().
		Str("name", string(name)).
		Str("data", firstFewBytes(data)).
		Int("perm", int(perm)).
		Err(err).
		Msg("WriteFile")
	return err
}

// WriteSymlink implements System.WriteSymlink.
func (s *DebugSystem) WriteSymlink(oldname string, newname AbsPath) error {
	err := s.system.WriteSymlink(oldname, newname)
	s.logger.Debug().
		Str("oldname", oldname).
		Str("newname", string(newname)).
		Err(err).
		Msg("WriteSymlink")
	return err
}

func cmdDict(cmd *exec.Cmd) *zerolog.Event {
	return zerolog.Dict().
		Str("path", cmd.Path).
		Strs("args", cmd.Args)
}

// firstFewBytes returns the first few bytes of data in a human-readable form.
func firstFewBytes(data []byte) string {
	const few = 64
	if len(data) > few {
		data = append([]byte{}, data[:few]...)
		data = append(data, '.', '.', '.')
	}
	s := strconv.Quote(string(data))
	return s[1 : len(s)-1]
}
