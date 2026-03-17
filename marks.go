package main

import (
	"os/exec"
	"sort"
	"strings"
	"time"
)

var cachedUserName string

func gitUserName() string {
	if cachedUserName != "" {
		return cachedUserName
	}
	out, err := exec.Command("git", "config", "user.name").Output()
	if err != nil {
		return "Unknown"
	}
	cachedUserName = strings.TrimSpace(string(out))
	if cachedUserName == "" {
		cachedUserName = "Unknown"
	}
	return cachedUserName
}

type SegmentState string

const (
	StateReviewed   SegmentState = "reviewed"
	StateUnreviewed SegmentState = "unreviewed"
	StateChanged    SegmentState = "changed"
)

type ImportanceState string

const (
	ImportanceIgnore ImportanceState = "ignore"
	ImportanceMedium ImportanceState = "medium"
	ImportanceHigh   ImportanceState = "high"
)

type ImportanceSegment struct {
	StartLine  int             `json:"startLine"`
	EndLine    int             `json:"endLine"`
	Importance ImportanceState `json:"importance"`
}

func nextImportance(s ImportanceState) ImportanceState {
	switch s {
	case ImportanceMedium:
		return ImportanceHigh
	case ImportanceHigh:
		return ImportanceIgnore
	default:
		return ImportanceMedium
	}
}

func importanceForLine(segs []ImportanceSegment, line int) ImportanceState {
	for _, seg := range segs {
		if line >= seg.StartLine && line <= seg.EndLine {
			return seg.Importance
		}
	}
	return ImportanceMedium
}

func mergeImportanceSegments(segs []ImportanceSegment) []ImportanceSegment {
	if len(segs) <= 1 {
		return segs
	}
	result := []ImportanceSegment{segs[0]}
	for i := 1; i < len(segs); i++ {
		prev := &result[len(result)-1]
		cur := segs[i]
		if prev.Importance == cur.Importance && prev.EndLine+1 == cur.StartLine {
			prev.EndLine = cur.EndLine
		} else {
			result = append(result, cur)
		}
	}
	return result
}

func toggleBlockImportance(segs []ImportanceSegment, blockStart, blockEnd, totalLines int) []ImportanceSegment {
	if len(segs) == 0 {
		segs = []ImportanceSegment{{StartLine: 1, EndLine: totalLines, Importance: ImportanceMedium}}
	}
	currentImp := importanceForLine(segs, blockStart)
	target := nextImportance(currentImp)

	var result []ImportanceSegment
	for _, seg := range segs {
		if seg.EndLine < blockStart || seg.StartLine > blockEnd {
			result = append(result, seg)
			continue
		}
		if seg.StartLine < blockStart {
			result = append(result, ImportanceSegment{
				StartLine:  seg.StartLine,
				EndLine:    blockStart - 1,
				Importance: seg.Importance,
			})
		}
		overlapStart := blockStart
		if seg.StartLine > blockStart {
			overlapStart = seg.StartLine
		}
		overlapEnd := blockEnd
		if seg.EndLine < blockEnd {
			overlapEnd = seg.EndLine
		}
		result = append(result, ImportanceSegment{
			StartLine:  overlapStart,
			EndLine:    overlapEnd,
			Importance: target,
		})
		if seg.EndLine > blockEnd {
			result = append(result, ImportanceSegment{
				StartLine:  blockEnd + 1,
				EndLine:    seg.EndLine,
				Importance: seg.Importance,
			})
		}
	}
	return mergeImportanceSegments(result)
}

func toggleFileImportance(segs []ImportanceSegment) []ImportanceSegment {
	if len(segs) == 0 {
		return segs
	}
	target := nextImportance(segs[0].Importance)
	lastLine := segs[len(segs)-1].EndLine
	return []ImportanceSegment{{StartLine: 1, EndLine: lastLine, Importance: target}}
}

func newFileImportanceSegments(lineCount int) []ImportanceSegment {
	if lineCount <= 0 {
		return nil
	}
	return []ImportanceSegment{{StartLine: 1, EndLine: lineCount, Importance: ImportanceMedium}}
}

type Segment struct {
	StartLine int          `json:"startLine"`
	EndLine   int          `json:"endLine"`
	UpdatedAt string       `json:"updatedAt"`
	State     SegmentState `json:"state"`
}

type Comment struct {
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
	Text      string `json:"text"`
	Author    string `json:"author"`
	CreatedAt string `json:"createdAt"`
}

func lineHasComment(comments []Comment, line int) bool {
	for _, c := range comments {
		if line >= c.StartLine && line <= c.EndLine {
			return true
		}
	}
	return false
}

type FileMarks struct {
	Path               string                  `json:"path"`
	FileName           string                  `json:"fileName"`
	Commit             string                  `json:"commit"`
	Reviewers          map[string][]Segment    `json:"reviewers"`
	ImportanceSegments []ImportanceSegment     `json:"importanceSegments,omitempty"`
	Comments           []Comment              `json:"comments,omitempty"`
}

func combinedSegments(reviewers map[string][]Segment) []Segment {
	if len(reviewers) == 0 {
		return nil
	}
	// Find max line across all reviewers
	maxLine := 0
	for _, segs := range reviewers {
		for _, s := range segs {
			if s.EndLine > maxLine {
				maxLine = s.EndLine
			}
		}
	}
	if maxLine == 0 {
		return nil
	}

	// For each line, determine combined state
	result := make([]Segment, 0)
	curState := StateUnreviewed
	curStart := 1

	for line := 1; line <= maxLine; line++ {
		state := StateUnreviewed
		hasChanged := false
		for _, segs := range reviewers {
			ls := segmentStateForLine(segs, line)
			if ls == StateReviewed {
				state = StateReviewed
				break
			}
			if ls == StateChanged {
				hasChanged = true
			}
		}
		if state != StateReviewed && hasChanged {
			state = StateChanged
		}

		if line == 1 {
			curState = state
			curStart = 1
			continue
		}
		if state != curState {
			result = append(result, Segment{StartLine: curStart, EndLine: line - 1, State: curState})
			curState = state
			curStart = line
		}
	}
	result = append(result, Segment{StartLine: curStart, EndLine: maxLine, State: curState})
	return result
}

type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
}

func nowStr() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func newFileSegments(lineCount int) []Segment {
	if lineCount <= 0 {
		return nil
	}
	return []Segment{{
		StartLine: 1,
		EndLine:   lineCount,
		UpdatedAt: nowStr(),
		State:     StateUnreviewed,
	}}
}

func mergeSegments(segs []Segment) []Segment {
	if len(segs) <= 1 {
		return segs
	}
	result := []Segment{segs[0]}
	for i := 1; i < len(segs); i++ {
		prev := &result[len(result)-1]
		cur := segs[i]
		if prev.State == cur.State && prev.EndLine+1 == cur.StartLine {
			prev.EndLine = cur.EndLine
		} else {
			result = append(result, cur)
		}
	}
	return result
}

func segmentStateForLine(segs []Segment, line int) SegmentState {
	for _, seg := range segs {
		if line >= seg.StartLine && line <= seg.EndLine {
			return seg.State
		}
	}
	return StateUnreviewed
}

func toggleBlockSegments(segs []Segment, blockStart, blockEnd int) []Segment {
	allReviewed := true
	for _, seg := range segs {
		if seg.StartLine > blockEnd || seg.EndLine < blockStart {
			continue
		}
		if seg.State != StateReviewed {
			allReviewed = false
			break
		}
	}

	targetState := StateReviewed
	if allReviewed {
		targetState = StateUnreviewed
	}

	ts := nowStr()
	var result []Segment
	for _, seg := range segs {
		if seg.EndLine < blockStart || seg.StartLine > blockEnd {
			result = append(result, seg)
			continue
		}
		if seg.StartLine < blockStart {
			result = append(result, Segment{
				StartLine: seg.StartLine,
				EndLine:   blockStart - 1,
				UpdatedAt: seg.UpdatedAt,
				State:     seg.State,
			})
		}
		overlapStart := blockStart
		if seg.StartLine > blockStart {
			overlapStart = seg.StartLine
		}
		overlapEnd := blockEnd
		if seg.EndLine < blockEnd {
			overlapEnd = seg.EndLine
		}
		result = append(result, Segment{
			StartLine: overlapStart,
			EndLine:   overlapEnd,
			UpdatedAt: ts,
			State:     targetState,
		})
		if seg.EndLine > blockEnd {
			result = append(result, Segment{
				StartLine: blockEnd + 1,
				EndLine:   seg.EndLine,
				UpdatedAt: seg.UpdatedAt,
				State:     seg.State,
			})
		}
	}
	return mergeSegments(result)
}

func toggleFileSegments(segs []Segment) []Segment {
	if len(segs) == 0 {
		return segs
	}
	targetState := StateReviewed
	if reviewedPercent(segs) == 100 {
		targetState = StateUnreviewed
	}
	lastLine := segs[len(segs)-1].EndLine
	return []Segment{{
		StartLine: 1,
		EndLine:   lastLine,
		UpdatedAt: nowStr(),
		State:     targetState,
	}}
}

func applyHunk(segs []Segment, h Hunk) []Segment {
	ts := nowStr()

	if h.OldCount == 0 && h.NewCount == 0 {
		return segs
	}

	if h.OldCount == 0 {
		// Pure insertion after line h.OldStart
		insertAt := h.OldStart
		var result []Segment
		inserted := false

		for _, seg := range segs {
			if seg.EndLine <= insertAt {
				result = append(result, seg)
			} else if seg.StartLine > insertAt {
				if !inserted {
					result = append(result, Segment{
						StartLine: insertAt + 1,
						EndLine:   insertAt + h.NewCount,
						UpdatedAt: ts,
						State:     StateChanged,
					})
					inserted = true
				}
				result = append(result, Segment{
					StartLine: seg.StartLine + h.NewCount,
					EndLine:   seg.EndLine + h.NewCount,
					UpdatedAt: seg.UpdatedAt,
					State:     seg.State,
				})
			} else {
				// Spans: seg.StartLine <= insertAt < seg.EndLine
				result = append(result, Segment{
					StartLine: seg.StartLine,
					EndLine:   insertAt,
					UpdatedAt: seg.UpdatedAt,
					State:     seg.State,
				})
				result = append(result, Segment{
					StartLine: insertAt + 1,
					EndLine:   insertAt + h.NewCount,
					UpdatedAt: ts,
					State:     StateChanged,
				})
				result = append(result, Segment{
					StartLine: insertAt + h.NewCount + 1,
					EndLine:   seg.EndLine + h.NewCount,
					UpdatedAt: seg.UpdatedAt,
					State:     seg.State,
				})
				inserted = true
			}
		}

		if !inserted {
			result = append(result, Segment{
				StartLine: insertAt + 1,
				EndLine:   insertAt + h.NewCount,
				UpdatedAt: ts,
				State:     StateChanged,
			})
		}
		return result
	}

	if h.NewCount == 0 {
		// Pure deletion
		delStart := h.OldStart
		delEnd := h.OldStart + h.OldCount - 1
		delta := -h.OldCount

		var result []Segment
		for _, seg := range segs {
			if seg.EndLine < delStart {
				result = append(result, seg)
			} else if seg.StartLine > delEnd {
				result = append(result, Segment{
					StartLine: seg.StartLine + delta,
					EndLine:   seg.EndLine + delta,
					UpdatedAt: seg.UpdatedAt,
					State:     seg.State,
				})
			} else {
				if seg.StartLine < delStart {
					result = append(result, Segment{
						StartLine: seg.StartLine,
						EndLine:   delStart - 1,
						UpdatedAt: seg.UpdatedAt,
						State:     seg.State,
					})
				}
				if seg.EndLine > delEnd {
					result = append(result, Segment{
						StartLine: delStart,
						EndLine:   seg.EndLine + delta,
						UpdatedAt: seg.UpdatedAt,
						State:     seg.State,
					})
				}
			}
		}
		return mergeSegments(result)
	}

	// Replacement: remove OldCount lines at OldStart, insert NewCount changed lines
	delStart := h.OldStart
	delEnd := h.OldStart + h.OldCount - 1
	delta := h.NewCount - h.OldCount

	var result []Segment
	insertedChanged := false
	for _, seg := range segs {
		if seg.EndLine < delStart {
			result = append(result, seg)
		} else if seg.StartLine > delEnd {
			result = append(result, Segment{
				StartLine: seg.StartLine + delta,
				EndLine:   seg.EndLine + delta,
				UpdatedAt: seg.UpdatedAt,
				State:     seg.State,
			})
		} else {
			if seg.StartLine < delStart {
				result = append(result, Segment{
					StartLine: seg.StartLine,
					EndLine:   delStart - 1,
					UpdatedAt: seg.UpdatedAt,
					State:     seg.State,
				})
			}
			if !insertedChanged {
				result = append(result, Segment{
					StartLine: delStart,
					EndLine:   delStart + h.NewCount - 1,
					UpdatedAt: ts,
					State:     StateChanged,
				})
				insertedChanged = true
			}
			if seg.EndLine > delEnd {
				result = append(result, Segment{
					StartLine: delEnd + 1 + delta,
					EndLine:   seg.EndLine + delta,
					UpdatedAt: seg.UpdatedAt,
					State:     seg.State,
				})
			}
		}
	}
	return mergeSegments(result)
}

func clampSegments(segs []Segment, maxLine int) []Segment {
	var result []Segment
	for _, s := range segs {
		if s.StartLine > maxLine {
			continue
		}
		if s.EndLine > maxLine {
			s.EndLine = maxLine
		}
		result = append(result, s)
	}
	return result
}

func clampImportanceSegments(segs []ImportanceSegment, maxLine int) []ImportanceSegment {
	var result []ImportanceSegment
	for _, s := range segs {
		if s.StartLine > maxLine {
			continue
		}
		if s.EndLine > maxLine {
			s.EndLine = maxLine
		}
		result = append(result, s)
	}
	return result
}

func reviewedPercent(segs []Segment) int {
	if len(segs) == 0 {
		return 0
	}
	var reviewed, total int
	for _, s := range segs {
		lines := s.EndLine - s.StartLine + 1
		total += lines
		if s.State == StateReviewed {
			reviewed += lines
		}
	}
	if total == 0 {
		return 0
	}
	return reviewed * 100 / total
}

func overallReviewedPercent(marks []FileMarks) int {
	var reviewed, total int
	for _, fm := range marks {
		for _, s := range combinedSegments(fm.Reviewers) {
			lines := s.EndLine - s.StartLine + 1
			total += lines
			if s.State == StateReviewed {
				reviewed += lines
			}
		}
	}
	if total == 0 {
		return 0
	}
	return reviewed * 100 / total
}

func applyHunks(segs []Segment, hunks []Hunk) []Segment {
	sort.Slice(hunks, func(i, j int) bool {
		return hunks[i].OldStart > hunks[j].OldStart
	})
	for _, h := range hunks {
		segs = applyHunk(segs, h)
	}
	return mergeSegments(segs)
}
