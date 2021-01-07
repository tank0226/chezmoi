package chezmoi

import "errors"

var errClosed = errors.New("closed")

// A MockPersistentState is a mock persistent state.
type MockPersistentState struct {
	buckets map[string]map[string][]byte
}

// NewMockPersistentState returns a new PersistentState.
func NewMockPersistentState() *MockPersistentState {
	return &MockPersistentState{
		buckets: make(map[string]map[string][]byte),
	}
}

// Close closes s.
func (s *MockPersistentState) Close() error {
	if s.buckets == nil {
		return errClosed
	}
	s.buckets = nil
	return nil
}

// CopyTo implements PersistentState.CopyTo.
func (s *MockPersistentState) CopyTo(p PersistentState) error {
	if s.buckets == nil {
		return errClosed
	}
	for bucket, bucketMap := range s.buckets {
		for key, value := range bucketMap {
			if err := p.Set([]byte(bucket), []byte(key), value); err != nil {
				return err
			}
		}
	}
	return nil
}

// Delete implements PersistentState.Delete.
func (s *MockPersistentState) Delete(bucket, key []byte) error {
	if s.buckets == nil {
		return errClosed
	}
	bucketMap, ok := s.buckets[string(bucket)]
	if !ok {
		return nil
	}
	delete(bucketMap, string(key))
	return nil
}

// ForEach implements PersistentState.ForEach.
func (s *MockPersistentState) ForEach(bucket []byte, fn func(k, v []byte) error) error {
	if s.buckets == nil {
		return errClosed
	}
	for k, v := range s.buckets[string(bucket)] {
		if err := fn([]byte(k), v); err != nil {
			return err
		}
	}
	return nil
}

// Get implements PersistentState.Get.
func (s *MockPersistentState) Get(bucket, key []byte) ([]byte, error) {
	if s.buckets == nil {
		return nil, errClosed
	}
	bucketMap, ok := s.buckets[string(bucket)]
	if !ok {
		return nil, nil
	}
	return bucketMap[string(key)], nil
}

// Set implements PersistentState.Set.
func (s *MockPersistentState) Set(bucket, key, value []byte) error {
	if s.buckets == nil {
		return errClosed
	}
	bucketMap, ok := s.buckets[string(bucket)]
	if !ok {
		bucketMap = make(map[string][]byte)
		s.buckets[string(bucket)] = bucketMap
	}
	bucketMap[string(key)] = value
	return nil
}
