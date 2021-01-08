package chezmoilog

import (
	"errors"
	"os"
	"os/exec"

	"github.com/rs/zerolog"
)

// An osExecCmdLogObject wraps an *os/exec.Cmd and adds
// github.com/rs/zerolog.LogObjectMarshaler functionality.
type osExecCmdLogObject struct {
	*exec.Cmd
}

// An osExecExitErrorLogObject wraps an error and adds
// github.com/rs/zerolog.LogObjectMarshaler functionality if the wrapped error
// is an os/exec.ExitError.
type osExecExitErrorLogObject struct {
	err error
}

// An osProcessStateLogObject wraps an *os.ProcessState and adds
// github.com/rs/zerolog.LogObjectMarshaler functionality.
type osProcessStateLogObject struct {
	*os.ProcessState
}

// MarshalZerologObject implements
// github.com/rs/zerolog.LogObjectMarshaler.MarshalZerologObject.
func (cmd osExecCmdLogObject) MarshalZerologObject(event *zerolog.Event) {
	if cmd.Path != "" {
		event.Str("path", cmd.Path)
	}
	if cmd.Args != nil {
		event.Strs("args", cmd.Args)
	}
	if cmd.Dir != "" {
		event.Str("dir", cmd.Dir)
	}
	if cmd.Env != nil {
		event.Strs("env", cmd.Env)
	}
}

// MarshalZerologObject implements
// github.com/rs/zerolog.LogObjectMarshaler.MarshalZerologObject.
func (err osExecExitErrorLogObject) MarshalZerologObject(event *zerolog.Event) {
	var osExecExitError *exec.ExitError
	if !errors.As(err.err, &osExecExitError) {
		return
	}
	event.EmbedObject(osProcessStateLogObject(osProcessStateLogObject{osExecExitError.ProcessState}))
	if osExecExitError.Stderr != nil {
		event.Bytes("stderr", osExecExitError.Stderr)
	}
}

// MarshalZerologObject implements
// github.com/rs/zerolog.LogObjectMarshaler.MarshalZerologObject.
func (p osProcessStateLogObject) MarshalZerologObject(event *zerolog.Event) {
	if p.Exited() {
		if !p.Success() {
			event.Int("exitCode", p.ExitCode())
		}
	} else {
		event.Int("pid", p.Pid())
	}
	if userTime := p.UserTime(); userTime != 0 {
		event.Dur("userTime", userTime)
	}
	if systemTime := p.SystemTime(); systemTime != 0 {
		event.Dur("systemTime", systemTime)
	}
}

// LogCmdCombinedOutput calls cmd.CombinedOutput, logs the result, and returns the result.
func LogCmdCombinedOutput(logger zerolog.Logger, cmd *exec.Cmd) ([]byte, error) {
	combinedOutput, err := cmd.CombinedOutput()
	if err != nil {
		logger.Warn().
			EmbedObject(osExecCmdLogObject{Cmd: cmd}).
			Err(err).
			EmbedObject(osExecExitErrorLogObject{err: err}).
			Msg("CombinedOutput")
		return combinedOutput, err
	}
	logger.Debug().
		EmbedObject(osExecCmdLogObject{Cmd: cmd}).
		Msg("CombinedOutput")
	return combinedOutput, nil
}

// LogCmdOutput calls cmd.Output, logs the result, and returns the result.
func LogCmdOutput(logger zerolog.Logger, cmd *exec.Cmd) ([]byte, error) {
	output, err := cmd.Output()
	if err != nil {
		logger.Error().
			EmbedObject(osExecCmdLogObject{Cmd: cmd}).
			Err(err).
			EmbedObject(osExecExitErrorLogObject{err: err}).
			Msg("Output")
		return output, err
	}
	logger.Debug().
		EmbedObject(osExecCmdLogObject{Cmd: cmd}).
		Msg("Output")
	return output, nil
}

// LogCmdRun calls cmd.Run, logs the result, and returns the result.
func LogCmdRun(logger zerolog.Logger, cmd *exec.Cmd) error {
	if err := cmd.Run(); err != nil {
		logger.Error().
			EmbedObject(osExecCmdLogObject{Cmd: cmd}).
			Err(err).
			EmbedObject(osExecExitErrorLogObject{err: err}).
			Msg("Run")
		return err
	}
	logger.Debug().
		EmbedObject(osExecCmdLogObject{Cmd: cmd}).
		Msg("Run")
	return nil
}
