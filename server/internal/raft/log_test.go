package raft

import (
	"reflect"
	"testing"

	"github.com/ARCoder181105/kvstore/internal/protocol"
)

//
// GetLastIndex Tests
//

func TestGetLastIndex_FreshNode(t *testing.T) {
	node := New("test", nil, nil)
	if idx := node.getLastIndex(); idx != 0 {
		t.Errorf("Expected 0 on fresh node, got %d", idx)
	}
}

func TestGetLastIndex_WithEntries(t *testing.T) {
	node := New("test", nil, nil)
	node.log = []LogEntry{
		{Index: 0, Term: 0},
		{Index: 1, Term: 1},
		{Index: 2, Term: 1},
	}
	if idx := node.getLastIndex(); idx != 2 {
		t.Errorf("Expected last index to be 2, got %d", idx)
	}
}

func TestGetLastIndex_AfterTruncate(t *testing.T) {
	node := New("test", nil, nil)
	node.log = []LogEntry{
		{Index: 0, Term: 0},
		{Index: 1, Term: 1},
		{Index: 2, Term: 1},
		{Index: 3, Term: 2},
	}
	node.truncateFrom(2) // Should leave [0, 1]
	if idx := node.getLastIndex(); idx != 1 {
		t.Errorf("Expected last index after truncate to be 1, got %d", idx)
	}
}

//
// GetLastTerm Tests
//

func TestGetLastTerm_FreshNode(t *testing.T) {
	node := New("test", nil, nil)
	if term := node.getLastTerm(); term != 0 {
		t.Errorf("Expected term 0 on fresh node, got %d", term)
	}
}

func TestGetLastTerm_WithEntries(t *testing.T) {
	node := New("test", nil, nil)
	node.log = []LogEntry{{Index: 0, Term: 0}, {Index: 1, Term: 5}}
	if term := node.getLastTerm(); term != 5 {
		t.Errorf("Expected term 5, got %d", term)
	}
}

func TestGetLastTerm_AfterTruncate(t *testing.T) {
	node := New("test", nil, nil)
	node.log = []LogEntry{{Index: 0, Term: 0}, {Index: 1, Term: 2}, {Index: 2, Term: 3}}
	node.truncateFrom(2) // Log becomes [ {Index:0, Term:0}, {Index:1,Term:2} ]
	if term := node.getLastTerm(); term != 2 {
		t.Errorf("Expected term 2 after truncate, got %d", term)
	}
}

//
// GetEntry Tests
//

func TestGetEntry_ValidIndex(t *testing.T) {
	node := New("test", nil, nil)
	expected := LogEntry{Index: 1, Term: 2, Command: protocol.Command{ID: protocol.CmdSet, Key: "a", Value: []byte("b")}}
	node.log = append(node.log, expected)
	
	actual := node.getEntry(1)
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %+v, got %+v", expected, actual)
	}
}

func TestGetEntry_OutOfBounds(t *testing.T) {
	node := New("test", nil, nil)
	actual := node.getEntry(5)
	if actual.Index != 0 || actual.Term != 0 {
		t.Errorf("Expected empty entry for out of bounds, got %+v", actual)
	}
}

func TestGetEntry_FreshNode(t *testing.T) {
	node := New("test", nil, nil)
	actual := node.getEntry(0)
	if actual.Index != 0 || actual.Term != 0 {
		t.Errorf("Expected empty entry for fresh node, got %+v", actual)
	}
}

//
// AppendEntry Tests
//

func TestAppendEntry_FreshNode(t *testing.T) {
	node := New("test", nil, nil)
	node.appendEntry(LogEntry{Index: 1, Term: 1})
	if len(node.log) != 2 || node.log[1].Term != 1 {
		t.Errorf("Failed to append to fresh node")
	}
}

func TestAppendEntry_ExistingEntries(t *testing.T) {
	node := New("test", nil, nil)
	node.appendEntry(LogEntry{Index: 1, Term: 2})
	if len(node.log) != 2 || node.log[1].Index != 1 {
		t.Errorf("Failed to append to existing log")
	}
}

//
// TruncateFrom Tests
//

func TestTruncateFrom_Middle(t *testing.T) {
	node := New("test", nil, nil)
	node.log = append(node.log, LogEntry{Index: 1, Term: 1}, LogEntry{Index: 2, Term: 2})
	node.truncateFrom(1) // Should keep only index 0
	if len(node.log) != 1 {
		t.Errorf("Expected length 1, got %d", len(node.log))
	}
}

func TestTruncateFrom_Full(t *testing.T) {
	node := New("test", nil, nil)
	node.truncateFrom(1) // Truncating from index 1 should preserve sentinel
	if len(node.log) != 1 {
		t.Errorf("Expected length 1, got %d", len(node.log))
	}
}

func TestTruncateFrom_OutOfBounds(t *testing.T) {
	node := New("test", nil, nil)
	node.truncateFrom(5) // Should have no effect
	if len(node.log) != 1 {
		t.Errorf("Expected length to stay 1, got %d", len(node.log))
	}
}

func TestTruncateFrom_FreshNode(t *testing.T) {
	node := New("test", nil, nil)
	node.truncateFrom(1) // Should keep the sentinel
	if len(node.log) != 1 {
		t.Errorf("Expected length 1, got %d", len(node.log))
	}
}
