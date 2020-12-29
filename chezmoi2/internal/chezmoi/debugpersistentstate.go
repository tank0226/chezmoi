package chezmoi

import (
	"github.com/rs/zerolog"
)

// A DebugPersistentState wraps a PersistentState and logs to a log.Logger.
type DebugPersistentState struct {
	s      PersistentState
	logger zerolog.Logger
}

// NewDebugPersistentState returns a new debugPersistentState that wraps s and
// logs to logger.
func NewDebugPersistentState(s PersistentState, logger zerolog.Logger) *DebugPersistentState {
	return &DebugPersistentState{
		s:      s,
		logger: logger,
	}
}

// Close implements PersistentState.Close.
func (s *DebugPersistentState) Close() error {
	err := s.s.Close()
	s.logger.Debug().
		Err(err).
		Msg("Close")
	return err
}

// CopyTo implements PersistentState.CopyTo.
func (s *DebugPersistentState) CopyTo(p PersistentState) error {
	err := s.s.CopyTo(p)
	s.logger.Debug().
		Err(err).
		Msg("CopyTo")
	return err
}

// Delete implements PersistentState.Delete.
func (s *DebugPersistentState) Delete(bucket, key []byte) error {
	err := s.s.Delete(bucket, key)
	s.logger.Debug().
		Bytes("bucket", bucket).
		Bytes("key", key).
		Err(err).
		Msg("Delete")
	return err
}

// ForEach implements PersistentState.ForEach.
func (s *DebugPersistentState) ForEach(bucket []byte, fn func(k, v []byte) error) error {
	err := s.s.ForEach(bucket, func(k, v []byte) error {
		err := fn(k, v)
		s.logger.Debug().
			Bytes("bucket", bucket).
			Bytes("key", k).
			Bytes("value", v).
			Err(err).
			Msg("ForEach")
		return err
	})
	s.logger.Debug().
		Bytes("bucket", bucket).
		Err(err)
	return err
}

// Get implements PersistentState.Get.
func (s *DebugPersistentState) Get(bucket, key []byte) ([]byte, error) {
	value, err := s.s.Get(bucket, key)
	s.logger.Debug().
		Bytes("bucket", bucket).
		Bytes("key", key).
		Bytes("value", value).
		Err(err).
		Msg("Get")
	return value, err
}

// Set implements PersistentState.Set.
func (s *DebugPersistentState) Set(bucket, key, value []byte) error {
	err := s.s.Set(bucket, key, value)
	s.logger.Debug().
		Bytes("bucket", bucket).
		Bytes("key", key).
		Bytes("value", value).
		Err(err).
		Msg("Set")
	return err
}
