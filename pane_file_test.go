package main

import (
	"strings"
	"testing"
)

// --- Block navigation tests (TDD: red first, then green) ---

func TestInitBlock_HalfScreen(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 100),
		lines:    make([]string, 100),
	}
	p.initBlock(40)
	if p.blockEnd-p.blockStart != 20 {
		t.Errorf("initBlock should set half screen size: got %d lines, want 20", p.blockEnd-p.blockStart)
	}
	if p.blockStart != 0 {
		t.Errorf("initBlock should start at 0, got %d", p.blockStart)
	}
	if p.presetIdx != 1 {
		t.Errorf("initBlock should set preset to 1 (half screen), got %d", p.presetIdx)
	}
}

func TestInitBlock_SmallFile(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 5),
		lines:    make([]string, 5),
	}
	p.initBlock(40)
	// half screen = 20 but file only has 5 lines, so block should clamp
	if p.blockEnd != 5 {
		t.Errorf("initBlock should clamp to file length: blockEnd got %d, want 5", p.blockEnd)
	}
}

func TestMoveBlockDown(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 20),
		lines:    make([]string, 20),
	}
	p.blockStart = 0
	p.blockEnd = 5
	p.moveBlockDown(20)
	if p.blockStart != 1 || p.blockEnd != 6 {
		t.Errorf("moveBlockDown: got [%d,%d), want [1,6)", p.blockStart, p.blockEnd)
	}
}

func TestMoveBlockDown_ClampsAtEnd(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 10),
		lines:    make([]string, 10),
	}
	p.blockStart = 7
	p.blockEnd = 10
	p.moveBlockDown(10)
	if p.blockStart != 7 || p.blockEnd != 10 {
		t.Errorf("moveBlockDown at end: got [%d,%d), want [7,10)", p.blockStart, p.blockEnd)
	}
}

func TestMoveBlockUp(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 20),
		lines:    make([]string, 20),
	}
	p.blockStart = 5
	p.blockEnd = 10
	p.moveBlockUp(20)
	if p.blockStart != 4 || p.blockEnd != 9 {
		t.Errorf("moveBlockUp: got [%d,%d), want [4,9)", p.blockStart, p.blockEnd)
	}
}

func TestMoveBlockUp_ClampsAtStart(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 10),
		lines:    make([]string, 10),
	}
	p.blockStart = 0
	p.blockEnd = 3
	p.moveBlockUp(10)
	if p.blockStart != 0 || p.blockEnd != 3 {
		t.Errorf("moveBlockUp at start: got [%d,%d), want [0,3)", p.blockStart, p.blockEnd)
	}
}

func TestGrowBlock(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 20),
		lines:    make([]string, 20),
	}
	p.blockStart = 5
	p.blockEnd = 8
	p.growBlock(20)
	if p.blockEnd != 9 {
		t.Errorf("growBlock: blockEnd got %d, want 9", p.blockEnd)
	}
	if p.blockStart != 5 {
		t.Errorf("growBlock: blockStart should not change, got %d", p.blockStart)
	}
}

func TestGrowBlock_ClampsAtFileEnd(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 10),
		lines:    make([]string, 10),
	}
	p.blockStart = 5
	p.blockEnd = 10
	p.growBlock(10)
	if p.blockEnd != 10 {
		t.Errorf("growBlock at end: blockEnd got %d, want 10", p.blockEnd)
	}
}

func TestShrinkBlock(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 20),
		lines:    make([]string, 20),
	}
	p.blockStart = 5
	p.blockEnd = 10
	p.shrinkBlock(20)
	if p.blockEnd != 9 {
		t.Errorf("shrinkBlock: blockEnd got %d, want 9", p.blockEnd)
	}
}

func TestShrinkBlock_MinimumOneeLine(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 20),
		lines:    make([]string, 20),
	}
	p.blockStart = 5
	p.blockEnd = 6 // 1 line
	p.shrinkBlock(20)
	if p.blockEnd != 6 {
		t.Errorf("shrinkBlock min 1 line: blockEnd got %d, want 6", p.blockEnd)
	}
}

func TestNextPreset(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 100),
		lines:    make([]string, 100),
	}
	p.blockStart = 0
	p.blockEnd = 50
	p.presetIdx = 1 // half screen
	p.nextPreset(100)
	if p.presetIdx != 2 {
		t.Errorf("nextPreset: presetIdx got %d, want 2", p.presetIdx)
	}
	if p.blockEnd-p.blockStart != 25 {
		t.Errorf("nextPreset: block size got %d, want 25 (quarter screen)", p.blockEnd-p.blockStart)
	}
}

func TestPrevPreset(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 100),
		lines:    make([]string, 100),
	}
	p.blockStart = 0
	p.blockEnd = 50
	p.presetIdx = 1 // half screen
	p.prevPreset(100)
	if p.presetIdx != 0 {
		t.Errorf("prevPreset: presetIdx got %d, want 0", p.presetIdx)
	}
	if p.blockEnd-p.blockStart != 100 {
		t.Errorf("prevPreset: block size got %d, want 100 (full file)", p.blockEnd-p.blockStart)
	}
}

func TestNextPreset_ClampsAtSmallest(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 100),
		lines:    make([]string, 100),
	}
	p.blockStart = 0
	p.blockEnd = 1
	p.presetIdx = 4 // already at 1 line
	p.nextPreset(100)
	if p.presetIdx != 4 {
		t.Errorf("nextPreset at min: presetIdx got %d, want 4", p.presetIdx)
	}
}

func TestPrevPreset_ClampsAtLargest(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 100),
		lines:    make([]string, 100),
	}
	p.blockStart = 0
	p.blockEnd = 100
	p.presetIdx = 0 // already at full file
	p.prevPreset(100)
	if p.presetIdx != 0 {
		t.Errorf("prevPreset at max: presetIdx got %d, want 0", p.presetIdx)
	}
}

func TestCenterBlock(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 100),
		lines:    make([]string, 100),
	}
	p.blockStart = 50
	p.blockEnd = 60
	p.centerBlock(20)
	// block mid = 55, should be at screen mid = offset + 10
	// so offset = 55 - 10 = 45
	if p.offset != 45 {
		t.Errorf("centerBlock: offset got %d, want 45", p.offset)
	}
}

func TestCenterBlock_ClampsAtStart(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 100),
		lines:    make([]string, 100),
	}
	p.blockStart = 0
	p.blockEnd = 4
	p.centerBlock(20)
	if p.offset != 0 {
		t.Errorf("centerBlock near start: offset got %d, want 0", p.offset)
	}
}

func TestCenterBlock_ClampsAtEnd(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 100),
		lines:    make([]string, 100),
	}
	p.blockStart = 95
	p.blockEnd = 100
	p.centerBlock(20)
	// max offset = 100 - 20 = 80
	if p.offset != 80 {
		t.Errorf("centerBlock near end: offset got %d, want 80", p.offset)
	}
}

// --- Jump navigation tests ---

func TestJumpToNext_Unreviewed(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 20),
		lines:    make([]string, 20),
	}
	p.blockStart = 0
	p.blockEnd = 3
	segs := []Segment{mkSeg(1, 5, StateReviewed), mkSeg(6, 20, StateUnreviewed)}
	p.jumpToNext(segs, nil, nil, jumpUnreviewed, 20)
	// Should jump to line 6 (0-indexed: 5)
	if p.blockStart != 5 {
		t.Errorf("jumpToNext unreviewed: blockStart got %d, want 5", p.blockStart)
	}
}

func TestJumpToNext_Changed(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 20),
		lines:    make([]string, 20),
	}
	p.blockStart = 0
	p.blockEnd = 3
	segs := []Segment{mkSeg(1, 10, StateReviewed), mkSeg(11, 15, StateChanged), mkSeg(16, 20, StateUnreviewed)}
	p.jumpToNext(segs, nil, nil, jumpChanged, 20)
	// Should jump to line 11 (0-indexed: 10)
	if p.blockStart != 10 {
		t.Errorf("jumpToNext changed: blockStart got %d, want 10", p.blockStart)
	}
}

func TestJumpToNext_Important(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 20),
		lines:    make([]string, 20),
	}
	p.blockStart = 0
	p.blockEnd = 3
	isegs := []ImportanceSegment{mkISeg(1, 10, ImportanceMedium), mkISeg(11, 15, ImportanceHigh)}
	p.jumpToNext(nil, isegs, nil, jumpImportant, 20)
	if p.blockStart != 10 {
		t.Errorf("jumpToNext important: blockStart got %d, want 10", p.blockStart)
	}
}

func TestJumpToNext_StopsAtEnd(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 10),
		lines:    make([]string, 10),
	}
	p.blockStart = 7
	p.blockEnd = 10
	segs := []Segment{mkSeg(1, 10, StateReviewed)}
	p.jumpToNext(segs, nil, nil, jumpUnreviewed, 10)
	// No unreviewed lines — should stay put
	if p.blockStart != 7 {
		t.Errorf("jumpToNext at end: blockStart got %d, want 7", p.blockStart)
	}
}

func TestJumpToPrev_Unreviewed(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 20),
		lines:    make([]string, 20),
	}
	p.blockStart = 15
	p.blockEnd = 18
	segs := []Segment{mkSeg(1, 5, StateUnreviewed), mkSeg(6, 20, StateReviewed)}
	p.jumpToPrev(segs, nil, nil, jumpUnreviewed, 20)
	// Should jump to start of unreviewed region (line 1, 0-indexed: 0)
	if p.blockStart != 0 {
		t.Errorf("jumpToPrev unreviewed: blockStart got %d, want 0", p.blockStart)
	}
}

func TestJumpToPrev_ClampsAtStart(t *testing.T) {
	p := filePane{
		rawLines: make([]string, 10),
		lines:    make([]string, 10),
	}
	p.blockStart = 0
	p.blockEnd = 3
	segs := []Segment{mkSeg(1, 10, StateReviewed)}
	p.jumpToPrev(segs, nil, nil, jumpUnreviewed, 10)
	// No unreviewed lines before — should stay put
	if p.blockStart != 0 {
		t.Errorf("jumpToPrev at start: blockStart got %d, want 0", p.blockStart)
	}
}

// --- Existing rendering tests updated for new block fields ---

func TestGutterRendersUnreviewedIndicator(t *testing.T) {
	p := filePane{
		lines:      []string{"line1", "line2", "line3"},
		rawLines:   []string{"line1", "line2", "line3"},
		blockStart: 0,
		blockEnd:   3,
		segments:   nil,
	}

	output := p.render(40, 10, true)
	if !strings.Contains(output, "⠿") {
		t.Error("unreviewed gutter should render ⠿ indicator (medium importance)")
	}
}

func TestGutterRendersReviewedIndicator(t *testing.T) {
	p := filePane{
		lines:      []string{"line1", "line2"},
		rawLines:   []string{"line1", "line2"},
		blockStart: 0,
		blockEnd:   2,
		segments:   []Segment{mkSeg(1, 2, StateReviewed)},
	}

	output := p.render(40, 10, true)
	if !strings.Contains(output, "⠿") {
		t.Error("reviewed gutter should render ⠿ indicator (medium importance)")
	}
}

func TestGutterRendersChangedIndicator(t *testing.T) {
	p := filePane{
		lines:      []string{"line1", "line2"},
		rawLines:   []string{"line1", "line2"},
		blockStart: 0,
		blockEnd:   2,
		segments:   []Segment{mkSeg(1, 2, StateChanged)},
	}

	output := p.render(40, 10, true)
	if !strings.Contains(output, "⠿") {
		t.Error("changed gutter should render ⠿ indicator (medium importance)")
	}
}

func TestFileHeaderShowsPercent(t *testing.T) {
	p := filePane{
		filePath:   "main.go",
		lines:      []string{"line1", "line2"},
		rawLines:   []string{"line1", "line2"},
		blockStart: 0,
		blockEnd:   2,
		segments:   []Segment{mkSeg(1, 1, StateReviewed), mkSeg(2, 2, StateUnreviewed)},
	}

	output := p.render(40, 10, true)
	if !strings.Contains(output, "50% reviewed") {
		t.Errorf("header should show 50%% reviewed, got:\n%s", output)
	}
}

func TestFileHeaderShowsZeroPercent(t *testing.T) {
	p := filePane{
		filePath:   "main.go",
		lines:      []string{"line1"},
		rawLines:   []string{"line1"},
		blockStart: 0,
		blockEnd:   1,
		segments:   []Segment{mkSeg(1, 1, StateUnreviewed)},
	}

	output := p.render(40, 10, true)
	if !strings.Contains(output, "0% reviewed") {
		t.Errorf("header should show 0%% reviewed, got:\n%s", output)
	}
}

func TestLineNumbersAppearInRender(t *testing.T) {
	p := filePane{
		lines:      []string{"aaa", "bbb", "ccc"},
		rawLines:   []string{"aaa", "bbb", "ccc"},
		blockStart: 0,
		blockEnd:   3,
	}

	output := p.render(40, 10, true)
	for _, num := range []string{"1 ", "2 ", "3 "} {
		if !strings.Contains(output, num) {
			t.Errorf("expected line number %q in output", num)
		}
	}
}

func TestGutterMixedSegments(t *testing.T) {
	p := filePane{
		lines:      []string{"line1", "line2", "line3", "line4"},
		rawLines:   []string{"line1", "line2", "line3", "line4"},
		blockStart: 0,
		blockEnd:   4,
		segments: []Segment{
			mkSeg(1, 2, StateReviewed),
			mkSeg(3, 4, StateUnreviewed),
		},
	}

	output := p.render(40, 10, true)
	if !strings.Contains(output, "⠿") {
		t.Error("should render ⠿ indicator for medium importance lines")
	}
}

func TestGutterHighImportance(t *testing.T) {
	p := filePane{
		lines:              []string{"line1", "line2"},
		rawLines:           []string{"line1", "line2"},
		blockStart:         0,
		blockEnd:           2,
		segments:           []Segment{mkSeg(1, 2, StateUnreviewed)},
		importanceSegments: []ImportanceSegment{mkISeg(1, 2, ImportanceHigh)},
	}

	output := p.render(40, 10, true)
	if !strings.Contains(output, "█") {
		t.Error("high importance should render █ indicator")
	}
}

func TestGutterIgnoreImportance(t *testing.T) {
	p := filePane{
		lines:              []string{"line1", "line2"},
		rawLines:           []string{"line1", "line2"},
		blockStart:         0,
		blockEnd:           2,
		segments:           []Segment{mkSeg(1, 2, StateUnreviewed)},
		importanceSegments: []ImportanceSegment{mkISeg(1, 2, ImportanceIgnore)},
	}

	output := p.render(40, 10, true)
	if !strings.Contains(output, "⠸") {
		t.Error("ignore importance should render ⠸ indicator")
	}
}
