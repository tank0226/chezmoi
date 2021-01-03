package chezmoi

import "encoding/json"

// A PersistentState is a persistent state.
type PersistentState interface {
	Close() error
	CopyTo(s PersistentState) error
	Delete(bucket, key []byte) error
	ForEach(bucket []byte, fn func(k, v []byte) error) error
	Get(bucket, key []byte) ([]byte, error)
	Set(bucket, key, value []byte) error
}

// PersistentStateData returns the structured data in s.
func PersistentStateData(s PersistentState) (interface{}, error) {
	entryStateData, err := persistentStateBucketData(s, entryStateBucket)
	if err != nil {
		return nil, err
	}
	scriptStateData, err := persistentStateBucketData(s, scriptStateBucket)
	if err != nil {
		return nil, err
	}
	return struct {
		EntryState  interface{} `json:"entryState" toml:"entryState" yaml:"entryState"`
		ScriptState interface{} `json:"scriptState" toml:"scriptState" yaml:"scriptState"`
	}{
		EntryState:  entryStateData,
		ScriptState: scriptStateData,
	}, nil
}

// persistentStateBucketData returns the state data in bucket in s.
func persistentStateBucketData(s PersistentState, bucket []byte) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	if err := s.ForEach(bucket, func(k, v []byte) error {
		var value map[string]interface{}
		if err := json.Unmarshal(v, &value); err != nil {
			return err
		}
		result[string(k)] = value
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}
