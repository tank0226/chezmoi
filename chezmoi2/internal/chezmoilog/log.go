package chezmoilog

import (
	"errors"
	"os/exec"

	"github.com/rs/zerolog"
)

// execCmdLogObject wraps an exec.Cmd and adds
// github.com/rs/zerolog.LogObjectMarshaler functionality.
type execCmdLogObject struct {
	*exec.Cmd
}

// MarshalZerologObject implements
// github.com/rs/zerolog.LogObjectMarshaler.MarshalZerologObject.
func (o execCmdLogObject) MarshalZerologObject(e *zerolog.Event) {
	if o.Path != "" {
		e.Str("path", o.Path)
	}
	if o.Args != nil {
		e.Strs("args", o.Args)
	}
	if o.Dir != "" {
		e.Str("dir", o.Dir)
	}
	if o.Env != nil {
		e.Strs("env", o.Env)
	}
}

// LogCmdCombinedOutput calls cmd.CombinedOutput, logs the result, and returns the result.
func LogCmdCombinedOutput(logger zerolog.Logger, cmd *exec.Cmd) ([]byte, error) {
	output, err := cmd.CombinedOutput()
	if err != nil {
		event := logger.Warn().Object("cmd", execCmdLogObject{Cmd: cmd}).Err(err)
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			event = event.Bytes("stderr", exitError.Stderr)
		}
		event.Msg("CombinedOutput")
		return output, err
	}
	logger.Debug().Object("cmd", execCmdLogObject{Cmd: cmd}).Msg("CombinedOutput")
	return output, nil
}

// LogCmdOutput calls cmd.Output, logs the result, and returns the result.
func LogCmdOutput(logger zerolog.Logger, cmd *exec.Cmd) ([]byte, error) {
	output, err := cmd.Output()
	if err != nil {
		event := logger.Warn().Object("cmd", execCmdLogObject{Cmd: cmd}).Err(err)
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			event = event.Bytes("stderr", exitError.Stderr)
		}
		event.Msg("Output")
		return output, err
	}
	logger.Debug().Object("cmd", execCmdLogObject{Cmd: cmd}).Msg("Output")
	return output, nil
}

// LogCmdRun calls cmd.Run, logs the result, and returns the result.
func LogCmdRun(logger zerolog.Logger, cmd *exec.Cmd) error {
	if err := cmd.Run(); err != nil {
		logger.Warn().Object("cmd", execCmdLogObject{Cmd: cmd}).Err(err).Msg("Run")
		return err
	}
	logger.Debug().Object("cmd", execCmdLogObject{Cmd: cmd}).Msg("Run")
	return nil
}
