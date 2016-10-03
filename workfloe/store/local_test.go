package store

import (
	"os/user"
	"testing"
)

func makeRoot(t *testing.T) string {
	usr, _ := user.Current()
	return usr.HomeDir + "/floe/unit_tests/"
}

func TestStore(t *testing.T) {
	s, err := NewLocalStore(makeRoot(t), JSONMarshaler{})
	if err != nil {
		t.Fatal(err)
	}

	// confirm satisfies interface
	func(s Store) {}(s)

	key := "a12"
	rt := "age"
	val := 45
	err = s.Set(key, rt, val)
	if err != nil {
		t.Fatal(err)
	}

	val = 0

	err = s.Get(key, rt, &val)
	if err != nil {
		t.Fatal(err)
	}

	if val != 45 {
		t.Error("did not load value in")
	}
}
