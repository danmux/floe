// Package store provides a general store to persist and depersist key value pairs
package store

type Store interface {
	Set(key, recType string, val interface{}) error // For a record type set the key to the value
	Get(key, recType string, val interface{}) error // For a record type and key get the value
	Del(key, recType string) error                  // For a record type and key delete the value
}

type Marshaler interface {
	Marshal(v interface{}) (data []byte, err error)
	Unmarshal(data []byte, v interface{}) error
	Type() string
}
