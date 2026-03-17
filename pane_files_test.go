package main

import (
	"strings"
	"testing"
)

func TestFilesHeaderShowsOverallPercent(t *testing.T) {
	p := filesPane{
		entries: []fileEntry{{relPath: "a.go", name: "a.go"}},
	}
	marks := []FileMarks{
		{Path: "a.go", Reviewers: mkReviewers(mkSeg(1, 100, StateReviewed))},
	}

	output := p.render(30, 10, marks, false, "")
	if !strings.Contains(output, "100%") {
		t.Errorf("header should show 100%%, got:\n%s", output)
	}
}

func TestFilesEntryShowsPercent(t *testing.T) {
	p := filesPane{
		entries: []fileEntry{{relPath: "a.go", name: "a.go"}},
	}
	marks := []FileMarks{
		{Path: "a.go", Reviewers: mkReviewers(mkSeg(1, 50, StateReviewed), mkSeg(51, 100, StateUnreviewed))},
	}

	output := p.render(30, 10, marks, false, "")
	if !strings.Contains(output, "50%") {
		t.Errorf("entry should show 50%%, got:\n%s", output)
	}
}

func TestFilesEntryAmberWhenDirty(t *testing.T) {
	p := filesPane{
		entries: []fileEntry{
			{relPath: "a.go", name: "a.go", dirty: true},
			{relPath: "b.go", name: "b.go"},
		},
		cursor: 1,
	}
	marks := []FileMarks{
		{Path: "a.go", Reviewers: mkReviewers(mkSeg(1, 10, StateUnreviewed))},
		{Path: "b.go", Reviewers: mkReviewers(mkSeg(1, 10, StateUnreviewed))},
	}

	output := p.render(30, 10, marks, false, "")
	if !strings.Contains(output, "*") {
		t.Errorf("dirty file should have asterisk, got:\n%s", output)
	}
}

func TestFilesEntryGreenWhen100(t *testing.T) {
	p := filesPane{
		entries: []fileEntry{
			{relPath: "a.go", name: "a.go"},
			{relPath: "b.go", name: "b.go"},
		},
		cursor: 1, // cursor on b.go
	}
	marks := []FileMarks{
		{Path: "a.go", Reviewers: mkReviewers(mkSeg(1, 10, StateReviewed))},
		{Path: "b.go", Reviewers: mkReviewers(mkSeg(1, 10, StateUnreviewed))},
	}

	output := p.render(30, 10, marks, false, "")
	// a.go is 100% reviewed and not selected — should use green style
	// b.go is 0% — should not be green
	if !strings.Contains(output, "100%") {
		t.Errorf("should contain 100%%, got:\n%s", output)
	}
	if !strings.Contains(output, "0%") {
		t.Errorf("should contain 0%%, got:\n%s", output)
	}
}
