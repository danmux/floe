package config

import (
	"testing"
)

func TestGraph(t *testing.T) {
	c, err := ParseYAML(in)
	if err != nil {
		t.Fatal(err)
	}
	graph, p := c.Flows[0].Graph()
	if len(p) != 0 {
		t.Error("something went wrong with the graph")
	}
	if len(graph) != 6 {
		t.Error("graph is the wrong length", len(graph))
	}
}
