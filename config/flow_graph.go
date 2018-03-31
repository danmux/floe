package config

import (
	"fmt"
	"strings"
)

type levelNode struct {
	level int
	node  *node
	kids  []*levelNode
}

// Graph returns an array representing levels in the flow 0 being the trigger events
// and the end events having the highest number
func (f *Flow) Graph() (lvs [][]string, problems []string) {

	lvs = append(lvs, []string{}, []string{})

	l0 := map[string]*levelNode{}
	l1 := map[string]*levelNode{}

	nodes := map[string]*levelNode{}

	// level 0 is triggers
	for _, t := range f.Triggers {
		if _, ok := nodes[t.ID]; ok {
			problems = append(problems, fmt.Sprintf("duplicate trigger id: %s", t.ID))
			continue
		}
		ln := &levelNode{
			node:  t,
			level: 0,
		}
		nodes[t.ID] = ln
		l0[ln.node.ID] = ln
		lvs[0] = append(lvs[0], t.ID)
	}
	for _, t := range f.Tasks {
		if _, ok := nodes[t.ID]; ok {
			problems = append(problems, fmt.Sprintf("duplicate node id: %s", t.ID))
			continue
		}
		ln := &levelNode{
			node:  t,
			level: 1,
		}
		nodes[t.ID] = ln
		if t.Listen == "trigger.good" { // trigger listeners are always level 1
			l1[ln.node.ID] = ln
			lvs[1] = append(lvs[1], t.ID)
		}
	}

	// find all events listened to - then all potential emitters of those events
	tagsAndListener := map[string][]*levelNode{}
	for _, ln := range nodes {
		switch ln.node.Class {
		case NcMerge:
			for _, t := range ln.node.Wait {
				a := tagsAndListener[t]
				a = append(a, ln)
				tagsAndListener[t] = a
			}
		case NcTask:
			t := ln.node.Listen
			a := tagsAndListener[t]
			a = append(a, ln)
			tagsAndListener[t] = a
		}
	}

	// for all events being listened to find the parents
	for t, ns := range tagsAndListener {
		if t == "trigger.good" { // trigger listeners are always level 1
			for _, n := range ns {
				n.level = 1
			}
			continue
		}
		parts := strings.Split(t, ".")
		if len(parts) != 3 {
			problems = append(problems, fmt.Sprintf("nodes are listening to invalid event: %s", t))
			continue
		}
		id := parts[1]
		parent, ok := nodes[id]
		if !ok {
			problems = append(problems, fmt.Sprintf("nodes are listening to invalid event: %s, that does not match a node", t))
			continue
		}
		parent.kids = append(parent.kids, ns...)
	}

	// starting from level 1 traverse its tree adding nodes to the correct level
	for _, n := range l1 {
		fillLevels(n)
	}

	// group them by levels
	lm := map[int]map[string]bool{}
	for _, n := range l1 {
		addToLevels(n, lm)
	}

	// convert to slice of slice
	for l := 2; l < len(lm)+2; l++ {
		lv := []string{}
		for k := range lm[l] {
			lv = append(lv, k)
		}
		lvs = append(lvs, lv)
	}

	return lvs, problems
}

func fillLevels(n *levelNode) {
	for _, kn := range n.kids {
		// if this node is now at a greater level than it was
		// boos its level and all its kids
		if kn.level < n.level+1 {
			kn.level = n.level + 1
		}
		fillLevels(kn)
	}
}

func addToLevels(n *levelNode, levs map[int]map[string]bool) {
	for _, kn := range n.kids {
		l, ok := levs[kn.level]
		if !ok {
			l = map[string]bool{}
		}
		l[kn.node.ID] = true
		levs[kn.level] = l
		addToLevels(kn, levs)
	}
}
