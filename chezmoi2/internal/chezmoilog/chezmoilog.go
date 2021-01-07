package chezmoilog

import (
	"errors"
	"os/exec"

	"github.com/rs/zerolog"
)

// osExecCmdLogObject wraps an os/exec.Cmd and adds
// github.com/rs/zerolog.LogObjectMarshaler functionality.
type osExecCmdLogObject struct {
	*exec.Cmd
}

// MarshalZerologObject implements
// github.com/rs/zerolog.LogObjectMarshaler.MarshalZerologObject.
func (o osExecCmdLogObject) MarshalZerologObject(event *zerolog.Event) {
	if o.Path != "" {
		event.Str("path", o.Path)
	}
	if o.Args != nil {
		event.Strs("args", o.Args)
	}
	if o.Dir != "" {
		event.Str("dir", o.Dir)
	}
	if o.Env != nil {
		event.Strs("env", o.Env)
	}
}

// osExecExitErrorLogObject wraps an error and adds
// github.com/rs/zerolog.LogObjectMarshaler functionality if the wrapped error
// is an os/exec.ExitError.
type osExecExitErrorLogObject struct {
	err error
}

// MarshalZerologObject implements
// github.com/rs/zerolog.LogObjectMarshaler.MarshalZerologObject.
func (err osExecExitErrorLogObject) MarshalZerologObject(event *zerolog.Event) {
	var osExecExitError *exec.ExitError
	if !errors.As(err.err, &osExecExitError) {
		return
	}
	if processState := osExecExitError.ProcessState; processState != nil {
		if exitCode := processState.ExitCode(); exitCode != 0 {
			event.Int("exitCode", processState.ExitCode())
		}
		if userTime := processState.UserTime(); userTime != 0 {
			event.Dur("userTime", processState.UserTime())
		}
		if systemTime := processState.SystemTime(); systemTime != 0 {
			event.Dur("systemTime", processState.SystemTime())
		}
	}
	if osExecExitError.Stderr != nil {
		event.Bytes("stderr", osExecExitError.Stderr)
	}
}

// LogCmdCombinedOutput calls cmd.CombinedOutput, logs the result, and returns the result.
func LogCmdCombinedOutput(logger zerolog.Logger, cmd *exec.Cmd) ([]byte, error) {
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Warn().
			EmbedObject(osExecCmdLogObject{Cmd: cmd}).
			Err(err).
			EmbedObject(osExecExitErrorLogObject{err: err}).
			Msg("CombinedOutput")
		return output, err
	}
	logger.Debug().
		EmbedObject(osExecCmdLogObject{Cmd: cmd}).
		Msg("CombinedOutput")
	return output, nil
}

// LogCmdOutput calls cmd.Output, logs the result, and returns the result.
func LogCmdOutput(logger zerolog.Logger, cmd *exec.Cmd) ([]byte, error) {
	output, err := cmd.Output()
	if err != nil {
		logger.Error().
			EmbedObject(osExecCmdLogObject{Cmd: cmd}).
			Err(err).
			EmbedObject(osExecExitErrorLogObject{err: err}).
			Msg("Run")
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
