package chezmoi

import "io"

// A StatusSystem wraps a system and logs all of the actions executed as a status.
type StatusSystem struct {
	system System
	w      io.Writer
	dir    string
}

// NewStatusSystem returns a new StatusSystem.
func NewStatusSystem(system System, w io.Writer, dir string, color bool) *StatusSystem {
	return &StatusSystem{
		system: system,
		w:      w,
		dir:    dir,
	}
}
