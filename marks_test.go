package main

import (
	"fmt"
	"testing"
)

type seg struct {
	start, end int
	state      SegmentState
}

func toSeg(segs []Segment) []seg {
	var result []seg
	for _, s := range segs {
		result = append(result, seg{s.StartLine, s.EndLine, s.State})
	}
	return result
}

func segsEqual(a, b []seg) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func mkSeg(start, end int, state SegmentState) Segment {
	return Segment{StartLine: start, EndLine: end, State: state}
}

func mkReviewers(segs ...Segment) map[string][]Segment {
	return map[string][]Segment{"Test User": segs}
}

func TestMergeSegments(t *testing.T) {
	tests := []struct {
		name string
		in   []Segment
		want []seg
	}{
		{
			"contiguous same state",
			[]Segment{mkSeg(1, 5, StateReviewed), mkSeg(6, 10, StateReviewed)},
			[]seg{{1, 10, StateReviewed}},
		},
		{
			"contiguous diff state",
			[]Segment{mkSeg(1, 5, StateReviewed), mkSeg(6, 10, StateUnreviewed)},
			[]seg{{1, 5, StateReviewed}, {6, 10, StateUnreviewed}},
		},
		{
			"single segment",
			[]Segment{mkSeg(1, 100, StateUnreviewed)},
			[]seg{{1, 100, StateUnreviewed}},
		},
		{
			"three segments middle differs",
			[]Segment{mkSeg(1, 5, StateReviewed), mkSeg(6, 10, StateUnreviewed), mkSeg(11, 15, StateReviewed)},
			[]seg{{1, 5, StateReviewed}, {6, 10, StateUnreviewed}, {11, 15, StateReviewed}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toSeg(mergeSegments(tt.in))
			if !segsEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToggleBlockSegments(t *testing.T) {
	tests := []struct {
		name       string
		segs       []Segment
		blockStart int
		blockEnd   int
		want       []seg
	}{
		{
			"unreviewed toggle to reviewed",
			[]Segment{mkSeg(1, 100, StateUnreviewed)},
			11, 20,
			[]seg{{1, 10, StateUnreviewed}, {11, 20, StateReviewed}, {21, 100, StateUnreviewed}},
		},
		{
			"all reviewed toggle to unreviewed and merge",
			[]Segment{mkSeg(1, 10, StateUnreviewed), mkSeg(11, 20, StateReviewed), mkSeg(21, 100, StateUnreviewed)},
			11, 20,
			[]seg{{1, 100, StateUnreviewed}},
		},
		{
			"mixed boundary at 15 toggle to reviewed",
			[]Segment{mkSeg(1, 15, StateReviewed), mkSeg(16, 100, StateUnreviewed)},
			10, 25,
			[]seg{{1, 25, StateReviewed}, {26, 100, StateUnreviewed}},
		},
		{
			"block at line 1",
			[]Segment{mkSeg(1, 100, StateUnreviewed)},
			1, 10,
			[]seg{{1, 10, StateReviewed}, {11, 100, StateUnreviewed}},
		},
		{
			"block at file end",
			[]Segment{mkSeg(1, 100, StateUnreviewed)},
			91, 100,
			[]seg{{1, 90, StateUnreviewed}, {91, 100, StateReviewed}},
		},
		{
			"block exactly matching segment",
			[]Segment{mkSeg(1, 10, StateUnreviewed), mkSeg(11, 20, StateUnreviewed), mkSeg(21, 30, StateUnreviewed)},
			11, 20,
			[]seg{{1, 10, StateUnreviewed}, {11, 20, StateReviewed}, {21, 30, StateUnreviewed}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toSeg(toggleBlockSegments(tt.segs, tt.blockStart, tt.blockEnd))
			if !segsEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToggleFileSegments(t *testing.T) {
	tests := []struct {
		name string
		segs []Segment
		want []seg
	}{
		{
			"all unreviewed becomes reviewed",
			[]Segment{mkSeg(1, 100, StateUnreviewed)},
			[]seg{{1, 100, StateReviewed}},
		},
		{
			"all reviewed becomes unreviewed",
			[]Segment{mkSeg(1, 100, StateReviewed)},
			[]seg{{1, 100, StateUnreviewed}},
		},
		{
			"mixed becomes reviewed",
			[]Segment{mkSeg(1, 50, StateReviewed), mkSeg(51, 100, StateUnreviewed)},
			[]seg{{1, 100, StateReviewed}},
		},
		{
			"mixed with changed becomes reviewed",
			[]Segment{mkSeg(1, 30, StateReviewed), mkSeg(31, 60, StateChanged), mkSeg(61, 100, StateUnreviewed)},
			[]seg{{1, 100, StateReviewed}},
		},
		{
			"empty returns empty",
			nil,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toSeg(toggleFileSegments(tt.segs))
			if !segsEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyHunks(t *testing.T) {
	tests := []struct {
		name  string
		segs  []Segment
		hunks []Hunk
		want  []seg
	}{
		{
			"insert 10 lines at 51",
			[]Segment{mkSeg(1, 100, StateReviewed), mkSeg(101, 200, StateUnreviewed)},
			[]Hunk{{OldStart: 50, OldCount: 0, NewStart: 51, NewCount: 10}},
			[]seg{{1, 50, StateReviewed}, {51, 60, StateChanged}, {61, 110, StateReviewed}, {111, 210, StateUnreviewed}},
		},
		{
			"delete 5 lines at 20",
			[]Segment{mkSeg(1, 100, StateUnreviewed)},
			[]Hunk{{OldStart: 20, OldCount: 5, NewStart: 20, NewCount: 0}},
			[]seg{{1, 95, StateUnreviewed}},
		},
		{
			"replace 3 with 7 at 10",
			[]Segment{mkSeg(1, 50, StateReviewed)},
			[]Hunk{{OldStart: 10, OldCount: 3, NewStart: 10, NewCount: 7}},
			[]seg{{1, 9, StateReviewed}, {10, 16, StateChanged}, {17, 54, StateReviewed}},
		},
		{
			"two hunks bottom-to-top",
			[]Segment{mkSeg(1, 100, StateReviewed)},
			[]Hunk{
				{OldStart: 20, OldCount: 0, NewStart: 21, NewCount: 5},
				{OldStart: 80, OldCount: 0, NewStart: 81, NewCount: 3},
			},
			[]seg{
				{1, 20, StateReviewed},
				{21, 25, StateChanged},
				{26, 85, StateReviewed},
				{86, 88, StateChanged},
				{89, 108, StateReviewed},
			},
		},
		{
			"insert at segment boundary",
			[]Segment{mkSeg(1, 50, StateReviewed), mkSeg(51, 100, StateUnreviewed)},
			[]Hunk{{OldStart: 50, OldCount: 0, NewStart: 51, NewCount: 5}},
			[]seg{
				{1, 50, StateReviewed},
				{51, 55, StateChanged},
				{56, 105, StateUnreviewed},
			},
		},
		{
			"deletion spanning two segments",
			[]Segment{mkSeg(1, 50, StateReviewed), mkSeg(51, 100, StateUnreviewed)},
			[]Hunk{{OldStart: 45, OldCount: 11, NewStart: 45, NewCount: 0}},
			[]seg{{1, 44, StateReviewed}, {45, 89, StateUnreviewed}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toSeg(applyHunks(tt.segs, tt.hunks))
			if !segsEqual(got, tt.want) {
				t.Errorf("\ngot  %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestParseHunkHeader(t *testing.T) {
	tests := []struct {
		line string
		want Hunk
	}{
		{"@@ -1,5 +1,5 @@", Hunk{1, 5, 1, 5}},
		{"@@ -10 +10,3 @@", Hunk{10, 1, 10, 3}},
		{"@@ -0,0 +1,10 @@", Hunk{0, 0, 1, 10}},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got, err := parseHunkHeader(tt.line)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestReviewedPercent(t *testing.T) {
	tests := []struct {
		name string
		segs []Segment
		want int
	}{
		{"empty", nil, 0},
		{"all unreviewed", []Segment{mkSeg(1, 100, StateUnreviewed)}, 0},
		{"all reviewed", []Segment{mkSeg(1, 100, StateReviewed)}, 100},
		{"half reviewed", []Segment{mkSeg(1, 50, StateReviewed), mkSeg(51, 100, StateUnreviewed)}, 50},
		{"partial reviewed", []Segment{mkSeg(1, 25, StateReviewed), mkSeg(26, 100, StateUnreviewed)}, 25},
		{"mixed with changed", []Segment{mkSeg(1, 50, StateReviewed), mkSeg(51, 75, StateChanged), mkSeg(76, 100, StateUnreviewed)}, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reviewedPercent(tt.segs)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestOverallReviewedPercent(t *testing.T) {
	tests := []struct {
		name  string
		marks []FileMarks
		want  int
	}{
		{"empty", nil, 0},
		{"single file all reviewed", []FileMarks{{Reviewers: mkReviewers(mkSeg(1, 100, StateReviewed))}}, 100},
		{"two files half each", []FileMarks{
			{Reviewers: mkReviewers(mkSeg(1, 50, StateReviewed), mkSeg(51, 100, StateUnreviewed))},
			{Reviewers: mkReviewers(mkSeg(1, 50, StateReviewed), mkSeg(51, 100, StateUnreviewed))},
		}, 50},
		{"two files different sizes", []FileMarks{
			{Reviewers: mkReviewers(mkSeg(1, 100, StateReviewed))},
			{Reviewers: mkReviewers(mkSeg(1, 100, StateUnreviewed))},
		}, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := overallReviewedPercent(tt.marks)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

// --- Importance tests ---

type iseg struct {
	start, end int
	importance ImportanceState
}

func toISeg(segs []ImportanceSegment) []iseg {
	var result []iseg
	for _, s := range segs {
		result = append(result, iseg{s.StartLine, s.EndLine, s.Importance})
	}
	return result
}

func isegsEqual(a, b []iseg) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func mkISeg(start, end int, imp ImportanceState) ImportanceSegment {
	return ImportanceSegment{StartLine: start, EndLine: end, Importance: imp}
}

func TestNextImportance(t *testing.T) {
	tests := []struct {
		in   ImportanceState
		want ImportanceState
	}{
		{ImportanceMedium, ImportanceHigh},
		{ImportanceHigh, ImportanceIgnore},
		{ImportanceIgnore, ImportanceMedium},
	}
	for _, tt := range tests {
		got := nextImportance(tt.in)
		if got != tt.want {
			t.Errorf("nextImportance(%s) = %s, want %s", tt.in, got, tt.want)
		}
	}
}

func TestImportanceForLine(t *testing.T) {
	tests := []struct {
		name string
		segs []ImportanceSegment
		line int
		want ImportanceState
	}{
		{"empty segs", nil, 5, ImportanceMedium},
		{"within segment", []ImportanceSegment{mkISeg(1, 10, ImportanceHigh)}, 5, ImportanceHigh},
		{"outside segments", []ImportanceSegment{mkISeg(1, 10, ImportanceHigh)}, 15, ImportanceMedium},
		{"boundary start", []ImportanceSegment{mkISeg(1, 10, ImportanceIgnore)}, 1, ImportanceIgnore},
		{"boundary end", []ImportanceSegment{mkISeg(1, 10, ImportanceIgnore)}, 10, ImportanceIgnore},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := importanceForLine(tt.segs, tt.line)
			if got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestMergeImportanceSegments(t *testing.T) {
	tests := []struct {
		name string
		in   []ImportanceSegment
		want []iseg
	}{
		{
			"adjacent same",
			[]ImportanceSegment{mkISeg(1, 5, ImportanceMedium), mkISeg(6, 10, ImportanceMedium)},
			[]iseg{{1, 10, ImportanceMedium}},
		},
		{
			"adjacent different",
			[]ImportanceSegment{mkISeg(1, 5, ImportanceMedium), mkISeg(6, 10, ImportanceHigh)},
			[]iseg{{1, 5, ImportanceMedium}, {6, 10, ImportanceHigh}},
		},
		{
			"single",
			[]ImportanceSegment{mkISeg(1, 100, ImportanceHigh)},
			[]iseg{{1, 100, ImportanceHigh}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toISeg(mergeImportanceSegments(tt.in))
			if !isegsEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToggleBlockImportance(t *testing.T) {
	tests := []struct {
		name       string
		segs       []ImportanceSegment
		blockStart int
		blockEnd   int
		totalLines int
		want       []iseg
	}{
		{
			"medium block becomes high",
			[]ImportanceSegment{mkISeg(1, 100, ImportanceMedium)},
			11, 20, 100,
			[]iseg{{1, 10, ImportanceMedium}, {11, 20, ImportanceHigh}, {21, 100, ImportanceMedium}},
		},
		{
			"high block becomes ignore",
			[]ImportanceSegment{mkISeg(1, 10, ImportanceMedium), mkISeg(11, 20, ImportanceHigh), mkISeg(21, 100, ImportanceMedium)},
			11, 20, 100,
			[]iseg{{1, 10, ImportanceMedium}, {11, 20, ImportanceIgnore}, {21, 100, ImportanceMedium}},
		},
		{
			"ignore block becomes medium and merges",
			[]ImportanceSegment{mkISeg(1, 10, ImportanceMedium), mkISeg(11, 20, ImportanceIgnore), mkISeg(21, 100, ImportanceMedium)},
			11, 20, 100,
			[]iseg{{1, 100, ImportanceMedium}},
		},
		{
			"empty segs defaults to medium then toggles to high",
			nil,
			5, 15, 50,
			[]iseg{{1, 4, ImportanceMedium}, {5, 15, ImportanceHigh}, {16, 50, ImportanceMedium}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toISeg(toggleBlockImportance(tt.segs, tt.blockStart, tt.blockEnd, tt.totalLines))
			if !isegsEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToggleFileImportance(t *testing.T) {
	tests := []struct {
		name string
		segs []ImportanceSegment
		want []iseg
	}{
		{
			"medium becomes high",
			[]ImportanceSegment{mkISeg(1, 100, ImportanceMedium)},
			[]iseg{{1, 100, ImportanceHigh}},
		},
		{
			"high becomes ignore",
			[]ImportanceSegment{mkISeg(1, 100, ImportanceHigh)},
			[]iseg{{1, 100, ImportanceIgnore}},
		},
		{
			"ignore becomes medium",
			[]ImportanceSegment{mkISeg(1, 100, ImportanceIgnore)},
			[]iseg{{1, 100, ImportanceMedium}},
		},
		{
			"empty returns empty",
			nil,
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toISeg(toggleFileImportance(tt.segs))
			if !isegsEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewFileImportanceSegments(t *testing.T) {
	got := toISeg(newFileImportanceSegments(100))
	want := []iseg{{1, 100, ImportanceMedium}}
	if !isegsEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if newFileImportanceSegments(0) != nil {
		t.Errorf("expected nil for lineCount=0")
	}
}

func TestSegmentStateForLine(t *testing.T) {
	segs := []Segment{
		mkSeg(1, 10, StateReviewed),
		mkSeg(11, 20, StateUnreviewed),
		mkSeg(21, 30, StateChanged),
	}

	tests := []struct {
		line int
		want SegmentState
	}{
		{5, StateReviewed},
		{15, StateUnreviewed},
		{25, StateChanged},
		{1, StateReviewed},    // segment start boundary
		{10, StateReviewed},   // segment end boundary
		{11, StateUnreviewed}, // next segment start
		{99, StateUnreviewed}, // beyond all segments
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("line_%d", tt.line), func(t *testing.T) {
			got := segmentStateForLine(segs, tt.line)
			if got != tt.want {
				t.Errorf("line %d: got %s, want %s", tt.line, got, tt.want)
			}
		})
	}
}
