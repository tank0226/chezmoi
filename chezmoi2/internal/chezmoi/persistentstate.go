package chezmoi

// A PersistentState is a persistent state.
type PersistentState interface {
	Close() error
	CopyTo(s PersistentState) error
	Delete(bucket, key []byte) error
	ForEach(bucket []byte, fn func(k, v []byte) error) error
	Get(bucket, key []byte) ([]byte, error)
	Set(bucket, key, value []byte) error
}
