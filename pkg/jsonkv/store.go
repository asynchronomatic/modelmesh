package jsonkv

import (
	"encoding/json"

	"github.com/dgraph-io/badger/v4"
)

// ErrNotFound is returned by Get when the key does not exist.
var ErrNotFound = badger.ErrKeyNotFound

// Store is a Badger-backed key/value store that JSON-encodes values.
type Store struct {
	db *badger.DB
}

// Open opens (or creates) a JSON key/value database at path.
// Path is a directory; Badger stores LSM files under it.
func Open(path string) (*Store, error) {
	opts := badger.DefaultOptions(path).WithLoggingLevel(badger.ERROR)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// Get unmarshals the JSON value at key into v.
func (s *Store) Get(key string, v any) error {
	return s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, v)
		})
	})
}

// Put marshals v as JSON and stores it at key.
func (s *Store) Put(key string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

// Delete removes key. It is not an error if the key is already absent.
func (s *Store) Delete(key string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// ForEach calls fn for every key that starts with prefix. fn must not retain
// the key or data slices after it returns.
func (s *Store) ForEach(prefix string, fn func(key string, data []byte) error) error {
	return s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		p := []byte(prefix)
		for it.Seek(p); it.ValidForPrefix(p); it.Next() {
			item := it.Item()
			key := string(item.KeyCopy(nil))
			err := item.Value(func(val []byte) error {
				cp := make([]byte, len(val))
				copy(cp, val)
				return fn(key, cp)
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// Close closes the underlying database. It is safe to call more than once.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}
