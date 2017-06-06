package coap

import (
	"strings"
	"testing"
)

type TestSet struct {
	Op  string
	Val string
	Pri int
	Exp string
}

func ck(a, b interface{}) int {
	return strings.Compare(a.(TestSet).Val, b.(TestSet).Val)
}

func cp(a, b interface{}) int {
	return a.(TestSet).Pri - b.(TestSet).Pri
}

func TestTreap(t *testing.T) {
	x := NewTreap(ck, cp)
	if x == nil {
		t.Errorf("expected NewTreap to work")
	}

	tests := []TestSet{
		{"get", "not-there", -1, "NIL"},
		{"ups", "a", 100, ""},
		{"get", "a", -1, "a"},
		{"ups", "b", 200, ""},
		{"get", "a", -1, "a"},
		{"get", "b", -1, "b"},
		{"ups", "c", 300, ""},
		{"get", "a", -1, "a"},
		{"get", "b", -1, "b"},
		{"get", "c", -1, "c"},
		{"get", "not-there", -1, "NIL"},
		{"ups", "a", 400, ""},
		{"get", "a", -1, "a"},
		{"get", "b", -1, "b"},
		{"get", "c", -1, "c"},
		{"get", "not-there", -1, "NIL"},
		{"del", "a", -1, ""},
		{"get", "a", -1, "NIL"},
		{"get", "b", -1, "b"},
		{"get", "c", -1, "c"},
		{"get", "not-there", -1, "NIL"},
		{"ups", "a", 10, ""},
		{"get", "a", -1, "a"},
		{"get", "b", -1, "b"},
		{"get", "c", -1, "c"},
		{"get", "not-there", -1, "NIL"},
		{"del", "a", -1, ""},
		{"del", "b", -1, ""},
		{"del", "c", -1, ""},
		{"get", "a", -1, "NIL"},
		{"get", "b", -1, "NIL"},
		{"get", "c", -1, "NIL"},
		{"get", "not-there", -1, "NIL"},
		{"del", "a", -1, ""},
		{"del", "b", -1, ""},
		{"del", "c", -1, ""},
		{"get", "a", -1, "NIL"},
		{"get", "b", -1, "NIL"},
		{"get", "c", -1, "NIL"},
		{"get", "not-there", -1, "NIL"},
		{"ups", "a", 10, ""},
		{"get", "a", -1, "a"},
		{"get", "b", -1, "NIL"},
		{"get", "c", -1, "NIL"},
		{"get", "not-there", -1, "NIL"},
		{"ups", "b", 1000, "b"},
		{"del", "b", -1, ""}, // cover join that is nil
		{"ups", "b", 20, "b"},
		{"ups", "c", 12, "c"},
		{"del", "b", -1, ""}, // cover join second return
		{"ups", "a", 5, "a"}, // cover upsert existing with lower priority
	}

	for testIdx, test := range tests {
		t.Log("id", testIdx, test)
		switch test.Op {
		case "get":
			i := x.Get(test)
			if !(i == nil && test.Exp == "NIL") && i.(TestSet).Val != test.Exp {
				t.Errorf("test: %v, on Get, expected: %v, got: %v", testIdx, test.Exp, i)
			}
		case "ups":
			x = x.Upsert(test)
		case "del":
			x = x.Delete(test)
		}
	}
}
