package client

import "testing"

func TestTagsMatch(t *testing.T) {
	fix := []struct {
		ht    []string
		it    []string
		match bool
	}{
		{
			ht:    []string{"tag1", "tag2"},
			it:    []string{"tag1", "tag2"},
			match: true,
		},
		{
			ht:    []string{"tag1", "tag2"},
			it:    []string{"tag1"},
			match: true,
		},
		{
			ht:    []string{"tag1", "tag2"},
			it:    []string{"tag1", "tag3"},
			match: false,
		},
		{
			ht:    []string{"tag1", "tag2"},
			it:    []string{"tag1", "tag2", "tag3"},
			match: false,
		},
		{
			ht:    []string{"tag1", "tag2"},
			it:    []string{},
			match: true,
		},
		{
			ht:    []string{},
			it:    []string{},
			match: true,
		},
		{
			ht:    []string{},
			it:    []string{"tag1", "tag2"},
			match: false,
		},
	}

	for i, f := range fix {
		hc := HostConfig{
			Tags: f.ht,
		}
		if hc.TagsMatch(f.it) != f.match {
			t.Errorf("%d expected %v, got the opposite", i, f.match)
		}
	}
}
