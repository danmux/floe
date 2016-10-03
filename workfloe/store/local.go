package store

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// LocalStore - store to disk in values marshalled by the passed in marshaller
type LocalStore struct {
	root      string // root path
	marshaler Marshaler
}

// NewLocalStore creates a local file system store
func NewLocalStore(rootPath string, marshaler Marshaler) (*LocalStore, error) {
	s := &LocalStore{
		root:      rootPath,
		marshaler: marshaler,
	}
	// enforce the existence of the root folder
	err := os.MkdirAll(s.root, 0777)
	return s, err
}

// rt and k are valid filename parts
func (s LocalStore) filename(k, r string) string {
	return filepath.Join(s.root, fmt.Sprintf("%s_%s.%s", k, r, s.marshaler.Type()))
}

// Set writes to disk the marshalled value of val
func (s *LocalStore) Set(key, recType string, val interface{}) error {
	b, err := s.marshaler.Marshal(val)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(s.filename(key, recType), b, 0640)
}

// Get reads from disk the marshalled value of val
func (s *LocalStore) Get(key, recType string, val interface{}) error {
	b, err := ioutil.ReadFile(s.filename(key, recType))
	if err != nil {
		if os.IsNotExist(err) { // not found is not an error, returning nil, nil is clearly an indication of not found
			return nil
		}
		return err
	}
	return s.marshaler.Unmarshal(b, val)
}

// Del deletes the resource from disk specified by the params
func (s *LocalStore) Del(key, recType string) error {
	err := os.Remove(s.filename(key, recType))
	if os.IsNotExist(err) { // not found is not an error, returning nil is clearly an indication of not found
		return nil
	}
	return err
}
