package main

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	jumpUnreviewed = 0
	jumpChanged    = 1
	jumpImportant  = 2
	jumpComments   = 3
)

type filePane struct {
	content    string
	lines      []string // syntax-highlighted lines
	rawLines   []string // raw source (no ANSI)
	blockStart int
	blockEnd   int
	presetIdx  int // index into presetSizes()
	offset     int
	filePath   string
	isBinary   bool
	tooLarge   bool
	uncommitted bool
	segments           []Segment
	importanceSegments []ImportanceSegment
	comments           []Comment
	viewLabel          string
}

func (p *filePane) loadFile(entry fileEntry) {
	p.filePath = entry.relPath
	p.offset = 0
	p.blockStart = 0
	p.blockEnd = 0
	p.presetIdx = 1

	if entry.dirty {
		p.uncommitted = true
		p.isBinary = false
		p.tooLarge = false
		p.content = ""
		p.lines = nil
		p.rawLines = nil
		return
	}

	p.uncommitted = false
	if isBinary(entry.relPath) {
		p.isBinary = true
		p.tooLarge = false
		p.content = ""
		p.lines = nil
		p.rawLines = nil
		return
	}

	p.isBinary = false
	content, lineCount, err := readFileContent(entry.relPath)
	if err != nil {
		p.content = "Error: " + err.Error()
		p.lines = []string{p.content}
		p.rawLines = []string{p.content}
		p.tooLarge = false
		return
	}

	if lineCount > 10000 {
		p.tooLarge = true
		p.content = ""
		p.lines = nil
		p.rawLines = nil
		return
	}

	p.tooLarge = false
	p.rawLines = strings.Split(content, "\n")
	p.content = highlightCode(content, entry.relPath)
	p.lines = strings.Split(p.content, "\n")
}

func (p *filePane) hasBlock() bool {
	return len(p.rawLines) > 0 && p.blockEnd > p.blockStart
}

func presetSizes(totalLines, visibleHeight int) []int {
	half := visibleHeight / 2
	if half < 1 {
		half = 1
	}
	quarter := visibleHeight / 4
	if quarter < 1 {
		quarter = 1
	}
	return []int{totalLines, half, quarter, 2, 1}
}

func (p *filePane) initBlock(visibleHeight int) {
	n := len(p.rawLines)
	if n == 0 {
		return
	}
	p.presetIdx = 1
	sizes := presetSizes(n, visibleHeight)
	size := sizes[p.presetIdx]
	p.blockStart = 0
	p.blockEnd = size
	if p.blockEnd > n {
		p.blockEnd = n
	}
	p.centerBlock(visibleHeight)
}

func (p *filePane) moveBlockDown(visibleHeight int) {
	n := len(p.rawLines)
	if p.blockEnd >= n {
		return
	}
	p.blockStart++
	p.blockEnd++
	p.centerBlock(visibleHeight)
}

func (p *filePane) moveBlockUp(visibleHeight int) {
	if p.blockStart <= 0 {
		return
	}
	p.blockStart--
	p.blockEnd--
	p.centerBlock(visibleHeight)
}

func (p *filePane) growBlock(visibleHeight int) {
	n := len(p.rawLines)
	if p.blockEnd >= n {
		return
	}
	p.blockEnd++
	p.centerBlock(visibleHeight)
}

func (p *filePane) shrinkBlock(visibleHeight int) {
	if p.blockEnd-p.blockStart <= 1 {
		return
	}
	p.blockEnd--
	p.centerBlock(visibleHeight)
}

func (p *filePane) nextPreset(visibleHeight int) {
	if p.presetIdx >= 4 {
		return
	}
	p.presetIdx++
	p.applyPreset(visibleHeight)
}

func (p *filePane) prevPreset(visibleHeight int) {
	if p.presetIdx <= 0 {
		return
	}
	p.presetIdx--
	p.applyPreset(visibleHeight)
}

func (p *filePane) applyPreset(visibleHeight int) {
	n := len(p.rawLines)
	sizes := presetSizes(n, visibleHeight)
	size := sizes[p.presetIdx]

	// Keep block centered around its current midpoint
	mid := (p.blockStart + p.blockEnd) / 2
	p.blockStart = mid - size/2
	p.blockEnd = p.blockStart + size

	if p.blockStart < 0 {
		p.blockStart = 0
		p.blockEnd = size
	}
	if p.blockEnd > n {
		p.blockEnd = n
		p.blockStart = n - size
		if p.blockStart < 0 {
			p.blockStart = 0
		}
	}
	p.centerBlock(visibleHeight)
}

func (p *filePane) centerBlock(visibleHeight int) {
	n := len(p.lines)
	if n == 0 {
		return
	}
	mid := (p.blockStart + p.blockEnd) / 2
	p.offset = mid - visibleHeight/2
	if p.offset < 0 {
		p.offset = 0
	}
	maxOffset := n - visibleHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if p.offset > maxOffset {
		p.offset = maxOffset
	}
}

func lineMatchesTarget(segs []Segment, isegs []ImportanceSegment, comments []Comment, line1 int, target int) bool {
	switch target {
	case jumpUnreviewed:
		return segmentStateForLine(segs, line1) == StateUnreviewed
	case jumpChanged:
		return segmentStateForLine(segs, line1) == StateChanged
	case jumpImportant:
		return importanceForLine(isegs, line1) == ImportanceHigh
	case jumpComments:
		return lineHasComment(comments, line1)
	}
	return false
}

func (p *filePane) jumpToNext(segs []Segment, isegs []ImportanceSegment, comments []Comment, target, visibleHeight int) {
	n := len(p.rawLines)
	size := p.blockEnd - p.blockStart
	for i := p.blockEnd; i < n; i++ {
		if lineMatchesTarget(segs, isegs, comments, i+1, target) {
			p.blockStart = i
			p.blockEnd = i + size
			if p.blockEnd > n {
				p.blockEnd = n
			}
			p.centerBlock(visibleHeight)
			return
		}
	}
}

func (p *filePane) jumpToPrev(segs []Segment, isegs []ImportanceSegment, comments []Comment, target, visibleHeight int) {
	size := p.blockEnd - p.blockStart
	for i := p.blockStart - 1; i >= 0; i-- {
		if lineMatchesTarget(segs, isegs, comments, i+1, target) {
			start := i
			for start > 0 && lineMatchesTarget(segs, isegs, comments, start, target) {
				start--
			}
			if !lineMatchesTarget(segs, isegs, comments, start+1, target) {
				start++
			}
			p.blockStart = start
			p.blockEnd = start + size
			n := len(p.rawLines)
			if p.blockEnd > n {
				p.blockEnd = n
			}
			p.centerBlock(visibleHeight)
			return
		}
	}
}

func (p *filePane) render(width, height int, focused bool) string {
	style := headerUnfocusedStyle
	if focused {
		style = headerFocusedStyle
	}
	pct := reviewedPercent(p.segments)
	headerText := fmt.Sprintf("%s [%d%% reviewed] (%s)", p.filePath, pct, p.viewLabel)
	header := style.Width(width).Render(headerText)

	contentHeight := height - 1

	var content string
	if p.uncommitted {
		content = warningStyle.Render("⚠ File has uncommitted changes — commit before reviewing")
	} else if p.isBinary {
		content = warningStyle.Render("⚠ Cannot display binary file")
	} else if p.tooLarge {
		content = warningStyle.Render("⚠ File exceeds 10,000 lines")
	} else if len(p.lines) == 0 {
		content = dimStyle.Render("(empty)")
	} else {
		end := p.offset + contentHeight
		if end > len(p.lines) {
			end = len(p.lines)
		}

		var selStart, selEnd int
		if p.hasBlock() {
			selStart = p.blockStart
			selEnd = p.blockEnd
		} else {
			selStart = -1
			selEnd = -1
		}

		gutterWidth := len(fmt.Sprintf("%d", len(p.lines)))
		hasComments := len(p.comments) > 0

		var lines []string
		for i := p.offset; i < end; i++ {
			imp := importanceForLine(p.importanceSegments, i+1)
			gutterCh := "⠿ "
			switch imp {
			case ImportanceIgnore:
				gutterCh = "⠸ "
			case ImportanceHigh:
				gutterCh = "█ "
			}

			gutterStyle := unreviewedGutterStyle
			if len(p.segments) > 0 {
				switch segmentStateForLine(p.segments, i+1) {
				case StateReviewed:
					gutterStyle = reviewedGutterStyle
				case StateChanged:
					gutterStyle = changedGutterStyle
				}
			}
			markGutter := gutterStyle.Render(gutterCh)

			commentMark := ""
			if hasComments {
				if lineHasComment(p.comments, i+1) {
					commentMark = commentStyle.Render("◆ ")
				} else {
					commentMark = "  "
				}
			}

			lineNum := lineNumberStyle.Render(fmt.Sprintf("%*d ", gutterWidth, i+1))

			selected := i >= selStart && i < selEnd
			selMark := "  "
			if selected {
				selMark = blockSelectionGutterStyle.Render("┃ ")
			}

			if selected {
				line := ""
				if i < len(p.lines) {
					line = p.lines[i]
				}
				lines = append(lines, lineNum+markGutter+commentMark+selMark+line)
			} else {
				raw := ""
				if i < len(p.rawLines) {
					raw = p.rawLines[i]
				}
				style := dimBlockStyle
				if len(p.segments) > 0 && segmentStateForLine(p.segments, i+1) == StateReviewed {
					style = reviewedBlockStyle
				}
				lines = append(lines, lineNum+markGutter+commentMark+selMark+style.Render(raw))
			}
		}
		content = strings.Join(lines, "\n")
	}

	contentBlock := lipgloss.NewStyle().
		Width(width).
		MaxWidth(width).
		Height(contentHeight).
		MaxHeight(contentHeight).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, header, contentBlock)
}
