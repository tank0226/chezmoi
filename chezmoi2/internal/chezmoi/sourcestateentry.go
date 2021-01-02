package chezmoi

// A SourceStateEntry represents the state of an entry in the source state.
type SourceStateEntry interface {
	Evaluate() error
	Name() SourceStatePath
	Order() int
	TargetStateEntry() (TargetStateEntry, error)
}

// A SourceStateDir represents the state of a directory in the source state.
type SourceStateDir struct {
	Attr             DirAttr
	name             SourceStatePath
	targetStateEntry TargetStateEntry
}

// A SourceStateFile represents the state of a file in the source state.
type SourceStateFile struct {
	*lazyContents
	Attr                 FileAttr
	name                 SourceStatePath
	targetStateEntryFunc func() (TargetStateEntry, error)
	targetStateEntry     TargetStateEntry
	targetStateEntryErr  error
}

// A SourceStateRemove represents that an entry should be removed.
type SourceStateRemove struct {
	name SourceStatePath
}

// A SourceStateRenameDir represents the renaming of a directory in the source
// state.
type SourceStateRenameDir struct {
	oldName SourceStatePath
	newName SourceStatePath
}

// Evaluate evaluates s and returns any error.
func (s *SourceStateDir) Evaluate() error {
	return nil
}

// Name returns s's name.
func (s *SourceStateDir) Name() SourceStatePath {
	return s.name
}

// Order returns s's order.
func (s *SourceStateDir) Order() int {
	return 0
}

// TargetStateEntry returns s's target state entry.
func (s *SourceStateDir) TargetStateEntry() (TargetStateEntry, error) {
	return s.targetStateEntry, nil
}

// Evaluate evaluates s and returns any error.
func (s *SourceStateFile) Evaluate() error {
	_, err := s.ContentsSHA256()
	return err
}

// Name returns s's name.
func (s *SourceStateFile) Name() SourceStatePath {
	return s.name
}

// Order returns s's order.
func (s *SourceStateFile) Order() int {
	return s.Attr.Order
}

// TargetStateEntry returns s's target state entry.
func (s *SourceStateFile) TargetStateEntry() (TargetStateEntry, error) {
	if s.targetStateEntryFunc != nil {
		s.targetStateEntry, s.targetStateEntryErr = s.targetStateEntryFunc()
		s.targetStateEntryFunc = nil
	}
	return s.targetStateEntry, s.targetStateEntryErr
}

// Evaluate evaluates s and returns any error.
func (s *SourceStateRemove) Evaluate() error {
	return nil
}

// Name returns s's name.
func (s *SourceStateRemove) Name() SourceStatePath {
	return s.name
}

// Order returns s's order.
func (s *SourceStateRemove) Order() int {
	return 0
}

// TargetStateEntry returns s's target state entry.
func (s *SourceStateRemove) TargetStateEntry() (TargetStateEntry, error) {
	return &TargetStateAbsent{}, nil
}

// Evaluate evaluates s and returns any error.
func (s *SourceStateRenameDir) Evaluate() error {
	return nil
}

// Order returns s's order.
func (s *SourceStateRenameDir) Order() int {
	return -1
}

// Name returns s's name.
func (s *SourceStateRenameDir) Name() SourceStatePath {
	return s.newName
}

// TargetStateEntry returns s's target state entry.
func (s *SourceStateRenameDir) TargetStateEntry() (TargetStateEntry, error) {
	return &TargetStateRenameDir{
		oldName: s.oldName,
		newName: s.newName,
	}, nil
}
