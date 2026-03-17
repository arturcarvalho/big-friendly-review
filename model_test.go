package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestSpaceKeyTogglesSegments(t *testing.T) {
	// Use a temp dir so saveMarks doesn't pollute the repo
	t.Chdir(t.TempDir())

	m := model{
		width:  80,
		height: 40,
		ready:  true,
		file: filePane{
			filePath:   "test.go",
			rawLines:   make([]string, 20),
			lines:      make([]string, 20),
			blockStart: 0,
			blockEnd:   10,
		},
		marks: []FileMarks{{
			Path:      "test.go",
			FileName:  "test.go",
			Reviewers: map[string][]Segment{"Test User": {mkSeg(1, 20, StateUnreviewed)}},
		}},
		userName: "Test User",
	}

	// Send space key press
	msg := tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	result, _ := m.Update(msg)
	updated := result.(model)

	// Block 0 covers lines 1-10 (1-based: start+1=1, end=10)
	// Should now be reviewed
	segs := updated.marks[0].Reviewers["Test User"]
	if len(segs) < 2 {
		t.Fatalf("expected segments to be split, got %d segment(s)", len(segs))
	}
	if segs[0].State != StateReviewed {
		t.Errorf("block lines should be reviewed, got %s", segs[0].State)
	}
	if segs[0].StartLine != 1 || segs[0].EndLine != 10 {
		t.Errorf("reviewed segment range: got %d-%d, want 1-10", segs[0].StartLine, segs[0].EndLine)
	}
	if segs[1].State != StateUnreviewed {
		t.Errorf("remaining lines should be unreviewed, got %s", segs[1].State)
	}
}

func TestIKeyTogglesBlockImportance(t *testing.T) {
	t.Chdir(t.TempDir())

	m := model{
		width:  80,
		height: 40,
		ready:  true,
		file: filePane{
			filePath:   "test.go",
			rawLines:   make([]string, 20),
			lines:      make([]string, 20),
			blockStart: 0,
			blockEnd:   10,
		},
		marks: []FileMarks{{
			Path:               "test.go",
			FileName:           "test.go",
			Reviewers:          map[string][]Segment{"Test User": {mkSeg(1, 20, StateUnreviewed)}},
			ImportanceSegments: []ImportanceSegment{mkISeg(1, 20, ImportanceMedium)},
		}},
	}

	msg := tea.KeyPressMsg{Code: -1, Text: "i"}

	// First press: medium → high
	result, _ := m.Update(msg)
	updated := result.(model)

	isegs := updated.marks[0].ImportanceSegments
	if len(isegs) < 2 {
		t.Fatalf("expected split, got %d segment(s)", len(isegs))
	}
	if isegs[0].Importance != ImportanceHigh {
		t.Errorf("block should be high, got %s", isegs[0].Importance)
	}
	if isegs[0].StartLine != 1 || isegs[0].EndLine != 10 {
		t.Errorf("range: got %d-%d, want 1-10", isegs[0].StartLine, isegs[0].EndLine)
	}

	// Second press: high → ignore
	result2, _ := updated.Update(msg)
	updated2 := result2.(model)
	isegs2 := updated2.marks[0].ImportanceSegments
	found := false
	for _, s := range isegs2 {
		if s.StartLine == 1 && s.EndLine == 10 {
			if s.Importance != ImportanceIgnore {
				t.Errorf("block should be ignore, got %s", s.Importance)
			}
			found = true
			break
		}
	}
	if !found {
		t.Errorf("did not find block 1-10 in %v", isegs2)
	}
}
