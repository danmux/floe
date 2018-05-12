package nodetype

import (
	"testing"
)

func TestMergeOpts(t *testing.T) {
	l := Opts{
		"foo": "bar",
		"env": []interface{}{
			"lar0",
			"lar1",
			"cmn",
		},
	}
	r := Opts{
		"baz": "boo",
		"env": []interface{}{
			"rar0",
			"rar1",
			"cmn",
		},
	}
	o := MergeOpts(l, r)
	envs, ok := o["env"]
	if !ok {
		t.Fatal("no env")
	}
	e, ok := envs.([]interface{})
	if !ok {
		t.Fatal("env not slice")
	}
	if len(e) != 6 {
		t.Fatal("opts env merge failed", len(e))
	}
	if e[4].(string) != "rar1" {
		t.Error("right env not appended")
	}

	l = Opts{
		"foo": "bar",
	}
	r = Opts{
		"baz": "boo",
		"env": []interface{}{
			"rar0",
		},
	}
	o = MergeOpts(l, r)
	_, ok = o["env"]
	if !ok {
		t.Fatal("no env when it did not exist")
	}
}
