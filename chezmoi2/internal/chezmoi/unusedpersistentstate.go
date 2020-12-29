package chezmoi

// An UnusedPersistentState is a PersistentState that is never used. All method
// calls panic.
type UnusedPersistentState struct{}

// NewUnusedPersistentState returns a new UnusedPersistentState.
func NewUnusedPersistentState() UnusedPersistentState { return UnusedPersistentState{} }

// Close does nothing.
func (UnusedPersistentState) Close() error { return nil }

// CopyTo panics.
func (UnusedPersistentState) CopyTo(PersistentState) error { panic(nil) }

// Delete panics.
func (UnusedPersistentState) Delete([]byte, []byte) error { panic(nil) }

// ForEach panics.
func (UnusedPersistentState) ForEach([]byte, func([]byte, []byte) error) error { panic(nil) }

// Get panics.
func (UnusedPersistentState) Get([]byte, []byte) ([]byte, error) { panic(nil) }

// Set panics.
func (UnusedPersistentState) Set([]byte, []byte, []byte) error { panic(nil) }
