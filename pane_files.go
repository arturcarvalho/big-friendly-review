package main

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

type fileEntry struct {
	relPath string
	name    string
	dirty   bool
}

type filesPane struct {
	allEntries []fileEntry
	entries    []fileEntry
	cursor     int
	offset     int
}

func (p *filesPane) filterEntries(query string) {
	if query == "" {
		p.entries = p.allEntries
	} else {
		q := strings.ToLower(query)
		var filtered []fileEntry
		for _, e := range p.allEntries {
			if strings.Contains(strings.ToLower(e.name), q) {
				filtered = append(filtered, e)
			}
		}
		p.entries = filtered
	}
	p.cursor = 0
	p.offset = 0
}

func (p *filesPane) moveDown() {
	if len(p.entries) == 0 {
		return
	}
	p.cursor++
	if p.cursor >= len(p.entries) {
		p.cursor = 0
	}
}

func (p *filesPane) moveUp() {
	if len(p.entries) == 0 {
		return
	}
	p.cursor--
	if p.cursor < 0 {
		p.cursor = len(p.entries) - 1
	}
}

func (p *filesPane) findFileMarks(path string, marks []FileMarks) *FileMarks {
	for i := range marks {
		if marks[i].Path == path {
			return &marks[i]
		}
	}
	return nil
}

func (p *filesPane) render(width, height int, marks []FileMarks, focused bool, query string) string {
	left := "Files"
	right := fmt.Sprintf("%d%% reviewed", overallReviewedPercent(marks))
	pad := width - len(left) - len(right) - 2 // -2 for padding(0,1)
	if pad < 1 {
		pad = 1
	}
	headerText := left + strings.Repeat(" ", pad) + right
	hStyle := headerUnfocusedStyle
	if focused {
		hStyle = headerFocusedStyle
	}
	header := hStyle.Width(width).Render(headerText)

	var searchBar string
	if focused {
		searchBar = normalStyle.Width(width).Render("> " + query + "█")
	} else if query != "" {
		searchBar = dimStyle.Width(width).Render("> " + query)
	} else {
		searchBar = dimStyle.Width(width).Render("> search (f)")
	}

	contentHeight := height - 2 // header + search bar

	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+contentHeight {
		p.offset = p.cursor - contentHeight + 1
	}

	var lines []string
	end := p.offset + contentHeight
	if end > len(p.entries) {
		end = len(p.entries)
	}
	for i := p.offset; i < end; i++ {
		entry := p.entries[i]
		prefix := "  "
		if i == p.cursor {
			prefix = "▸ "
		}

		pct := 0
		if fm := p.findFileMarks(entry.relPath, marks); fm != nil {
			combined := combinedSegments(fm.Reviewers)
			pct = reviewedPercent(combined)
		}

		name := entry.name
		dirtyMark := ""
		if entry.dirty {
			name = changedFileStyle.Render(name)
			dirtyMark = changedFileStyle.Render("*")
		}

		pctStr := fmt.Sprintf("%d%%", pct)
		nameText := prefix + name + dirtyMark
		nameLen := lipgloss.Width(nameText)
		gap := width - nameLen - len(pctStr)
		if gap < 1 {
			gap = 1
		}
		line := nameText + strings.Repeat(" ", gap) + pctStr

		if i == p.cursor {
			line = selectedStyle.Width(width).Render(line)
		} else {
			line = normalStyle.Width(width).Render(line)
		}
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	contentBlock := lipgloss.NewStyle().
		Width(width).
		Height(contentHeight).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, header, searchBar, contentBlock)
}
