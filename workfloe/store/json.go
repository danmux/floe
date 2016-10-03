package store

import (
	"encoding/json"
)

type JSONMarshaler struct{}

func (m JSONMarshaler) Marshal(v interface{}) (data []byte, err error) {
	return json.MarshalIndent(v, "", "  ")
}

func (m JSONMarshaler) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (m JSONMarshaler) Type() string {
	return "json"
}
