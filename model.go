package main

import (
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/fsnotify/fsnotify"
)

const (
	viewAll         = 0
	viewCurrentUser = 1
	viewOthers      = 2
)

type fileChangedMsg struct{}

type model struct {
	files    filesPane
	file     filePane
	width    int
	height   int
	ready    bool
	marks    []FileMarks
	userName string
	viewMode int
	watcher      *fsnotify.Watcher
	showHelp     bool
	showFiles    bool
	focusFiles   bool
	searchQuery  string
	jumpTarget   int
	commentInput bool
	commentText  string
}

func (m *model) segmentsForView(fm FileMarks) []Segment {
	switch m.viewMode {
	case viewCurrentUser:
		return fm.Reviewers[m.userName]
	case viewOthers:
		others := make(map[string][]Segment)
		for name, segs := range fm.Reviewers {
			if name != m.userName {
				others[name] = segs
			}
		}
		return combinedSegments(others)
	default:
		return combinedSegments(fm.Reviewers)
	}
}

func (m *model) viewModeLabel() string {
	switch m.viewMode {
	case viewCurrentUser:
		return m.userName
	case viewOthers:
		return "Others"
	default:
		return "All reviewers"
	}
}

func (m *model) refreshFileSegments() {
	for _, fm := range m.marks {
		if fm.Path == m.file.filePath {
			m.file.segments = m.segmentsForView(fm)
			m.file.viewLabel = m.viewModeLabel()
			return
		}
	}
}

func newModel(dirtyPaths []string) (model, error) {
	entries, err := loadFiles(".")
	if err != nil {
		return model{}, err
	}

	dirtySet := make(map[string]bool, len(dirtyPaths))
	for _, p := range dirtyPaths {
		dirtySet[p] = true
	}
	for i := range entries {
		if dirtySet[entries[i].relPath] {
			entries[i].dirty = true
		}
	}

	marks, err := initOrUpdateMarks(entries)
	if err != nil {
		return model{}, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return model{}, err
	}
	filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".bfr" {
				return filepath.SkipDir
			}
			watcher.Add(path)
		}
		return nil
	})

	m := model{
		files:     filesPane{allEntries: entries, entries: entries},
		marks:     marks,
		userName:  gitUserName(),
		watcher:   watcher,
		showFiles: true,
	}
	m.sortEntries()

	m.files.cursor = 0
	m.loadSelectedFile()

	return m, nil
}

func waitForFileChange(w *fsnotify.Watcher) tea.Cmd {
	return func() tea.Msg {
		for {
			select {
			case event, ok := <-w.Events:
				if !ok {
					return nil
				}
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
					return fileChangedMsg{}
				}
			case _, ok := <-w.Errors:
				if !ok {
					return nil
				}
			}
		}
	}
}

func (m *model) findSegments(path string) []Segment {
	for _, fm := range m.marks {
		if fm.Path == path {
			return m.segmentsForView(fm)
		}
	}
	return nil
}

func (m *model) findImportanceSegments(path string) []ImportanceSegment {
	for _, fm := range m.marks {
		if fm.Path == path {
			return fm.ImportanceSegments
		}
	}
	return nil
}

func (m *model) findComments(path string) []Comment {
	for _, fm := range m.marks {
		if fm.Path == path {
			return fm.Comments
		}
	}
	return nil
}

func (m *model) loadSelectedFile() {
	if len(m.files.entries) > 0 && m.files.cursor < len(m.files.entries) {
		e := m.files.entries[m.files.cursor]
		m.file.loadFile(e)
		lineCount := len(m.file.rawLines)
		m.file.segments = clampSegments(m.findSegments(e.relPath), lineCount)
		m.file.importanceSegments = clampImportanceSegments(m.findImportanceSegments(e.relPath), lineCount)
		m.file.comments = m.findComments(e.relPath)
		m.file.viewLabel = m.viewModeLabel()
		m.file.initBlock(m.contentHeight())
	}
}

func (m model) contentHeight() int {
	h := m.height - 2 // header + footer
	if h < 1 {
		h = 1
	}
	return h
}

func (m model) Init() tea.Cmd {
	return waitForFileChange(m.watcher)
}

func (m *model) updateDirtyState() {
	dirtySet := make(map[string]bool)
	for _, p := range detectDirtyPaths() {
		dirtySet[p] = true
	}
	for i := range m.files.allEntries {
		m.files.allEntries[i].dirty = dirtySet[m.files.allEntries[i].relPath]
	}
}

func (m *model) sortEntries() {
	curPath := ""
	if len(m.files.allEntries) > 0 && m.files.cursor < len(m.files.entries) {
		curPath = m.files.entries[m.files.cursor].relPath
	}

	marks := m.marks
	filePct := func(e fileEntry) int {
		for _, fm := range marks {
			if fm.Path == e.relPath {
				return reviewedPercent(combinedSegments(fm.Reviewers))
			}
		}
		return 0
	}
	sort.Slice(m.files.allEntries, func(i, j int) bool {
		return filePct(m.files.allEntries[i]) < filePct(m.files.allEntries[j])
	})
	m.files.filterEntries(m.searchQuery)

	for i, e := range m.files.entries {
		if e.relPath == curPath {
			m.files.cursor = i
			break
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case fileChangedMsg:
		m.updateDirtyState()
		m.sortEntries()
		m.loadSelectedFile()
		return m, waitForFileChange(m.watcher)

	case tea.WindowSizeMsg:
		first := !m.ready
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		if first {
			m.file.initBlock(m.contentHeight())
		}
		return m, nil

	case tea.KeyPressMsg:
		key := msg.String()

		// Comment input mode — capture all keys
		if m.commentInput {
			switch key {
			case "esc":
				m.commentInput = false
				m.commentText = ""
			case "enter":
				if m.commentText != "" && m.file.hasBlock() {
					c := Comment{
						StartLine: m.file.blockStart + 1,
						EndLine:   m.file.blockEnd,
						Text:      m.commentText,
						Author:    m.userName,
						CreatedAt: nowStr(),
					}
					for i, fm := range m.marks {
						if fm.Path == m.file.filePath {
							m.marks[i].Comments = append(m.marks[i].Comments, c)
							m.file.comments = m.marks[i].Comments
							saveMarks(m.marks)
							break
						}
					}
				}
				m.commentInput = false
				m.commentText = ""
			case "backspace":
				if len(m.commentText) > 0 {
					runes := []rune(m.commentText)
					m.commentText = string(runes[:len(runes)-1])
				}
			case "space":
				m.commentText += " "
			default:
				runes := []rune(key)
				if len(runes) == 1 && runes[0] >= 32 {
					m.commentText += key
				}
			}
			return m, nil
		}

		// Global keys (always active)
		switch key {
		case "q", "ctrl+c":
			if m.watcher != nil {
				m.watcher.Close()
			}
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "super+b":
			m.showFiles = !m.showFiles
			if !m.showFiles {
				m.focusFiles = false
			}
			return m, nil
		}

		// Files pane focused
		if m.focusFiles {
			switch key {
			case "esc":
				m.focusFiles = false
			case "enter":
				m.focusFiles = false
				m.loadSelectedFile()
			case "up":
				m.files.moveUp()
				m.loadSelectedFile()
			case "down":
				m.files.moveDown()
				m.loadSelectedFile()
			case "backspace":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					m.files.filterEntries(m.searchQuery)
					m.loadSelectedFile()
				}
			default:
				if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
					m.searchQuery += key
					m.files.filterEntries(m.searchQuery)
					m.loadSelectedFile()
				}
			}
			return m, nil
		}

		// Code pane focused
		switch key {
		case "f":
			m.focusFiles = true
			m.showFiles = true
		case "j", "down":
			m.file.moveBlockDown(m.contentHeight())
		case "k", "up":
			m.file.moveBlockUp(m.contentHeight())
		case "J", "shift+down":
			m.file.growBlock(m.contentHeight())
		case "K", "shift+up":
			m.file.shrinkBlock(m.contentHeight())
		case "r":
			m.viewMode = (m.viewMode + 1) % 3
			m.refreshFileSegments()
		case "space":
			if m.file.hasBlock() {
				blockStart := m.file.blockStart + 1
				blockEnd := m.file.blockEnd
				for i, fm := range m.marks {
					if fm.Path == m.file.filePath {
						if m.marks[i].Reviewers == nil {
							m.marks[i].Reviewers = make(map[string][]Segment)
						}
						userSegs := m.marks[i].Reviewers[m.userName]
						if userSegs == nil {
							userSegs = newFileSegments(len(m.file.rawLines))
						}
						m.marks[i].Reviewers[m.userName] = toggleBlockSegments(userSegs, blockStart, blockEnd)
						m.viewMode = viewCurrentUser
						m.file.segments = m.marks[i].Reviewers[m.userName]
						m.file.viewLabel = m.viewModeLabel()
						saveMarks(m.marks)
						break
					}
				}
			}
		case "i":
			if m.file.hasBlock() {
				blockStart := m.file.blockStart + 1
				blockEnd := m.file.blockEnd
				totalLines := len(m.file.rawLines)
				for i, fm := range m.marks {
					if fm.Path == m.file.filePath {
						m.marks[i].ImportanceSegments = toggleBlockImportance(fm.ImportanceSegments, blockStart, blockEnd, totalLines)
						m.file.importanceSegments = m.marks[i].ImportanceSegments
						saveMarks(m.marks)
						break
					}
				}
			}
		case "h", "left":
			m.file.prevPreset(m.contentHeight())
		case "l", "right":
			m.file.nextPreset(m.contentHeight())
		case "n":
			m.file.jumpToNext(m.file.segments, m.file.importanceSegments, m.file.comments, m.jumpTarget, m.contentHeight())
		case "N":
			m.file.jumpToPrev(m.file.segments, m.file.importanceSegments, m.file.comments, m.jumpTarget, m.contentHeight())
		case "t":
			m.jumpTarget = (m.jumpTarget + 1) % 4
		case "c":
			if m.file.hasBlock() {
				m.commentInput = true
				m.commentText = ""
			}
		case "d":
			if m.file.hasBlock() {
				blockStart := m.file.blockStart + 1
				blockEnd := m.file.blockEnd
				for i, fm := range m.marks {
					if fm.Path == m.file.filePath {
						var kept []Comment
						for _, c := range fm.Comments {
							if c.StartLine > blockEnd || c.EndLine < blockStart {
								kept = append(kept, c)
							}
						}
						m.marks[i].Comments = kept
						m.file.comments = kept
						saveMarks(m.marks)
						break
					}
				}
			}
		case "e":
			editor := os.Getenv("EDITOR")
			if editor != "" && m.file.filePath != "" {
				parts := strings.Fields(editor)
				args := append(parts[1:], m.file.filePath)
				cmd := exec.Command(parts[0], args...)
				return m, tea.ExecProcess(cmd, nil)
			}
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	if !m.ready {
		v := tea.NewView("Loading...")
		v.AltScreen = true
		return v
	}

	extraLines := 0
	if m.commentInput {
		extraLines++
	}
	// Check if comment display will be shown
	if !m.commentInput && m.file.hasBlock() {
		for _, c := range m.file.comments {
			if m.file.blockStart+1 <= c.EndLine && m.file.blockEnd >= c.StartLine {
				extraLines++
				break
			}
		}
	}

	paneHeight := m.height - 2 - extraLines

	var panes string
	if m.showFiles {
		leftWidth := m.width / 4
		if leftWidth < 20 {
			leftWidth = 20
		}
		rightWidth := m.width - leftWidth - 1
		left := m.files.render(leftWidth, paneHeight, m.marks, m.focusFiles, m.searchQuery)
		sep := renderSeparator(paneHeight)
		right := m.file.render(rightWidth, paneHeight, !m.focusFiles)
		panes = lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)
	} else {
		panes = m.file.render(m.width, paneHeight, true)
	}
	var commentBar string
	if m.commentInput {
		commentBar = commentStyle.Width(m.width).Render("comment: " + m.commentText + "█")
	}

	// Show comment text if block overlaps a comment
	var commentDisplay string
	if !m.commentInput && m.file.hasBlock() {
		for _, c := range m.file.comments {
			if m.file.blockStart+1 <= c.EndLine && m.file.blockEnd >= c.StartLine {
				commentDisplay = commentStyle.Width(m.width).Render("◆ " + c.Author + ": " + c.Text)
				break
			}
		}
	}

	footer := renderFooter(m.width, m.jumpTarget)
	parts := []string{panes}
	if commentBar != "" {
		parts = append(parts, commentBar)
	}
	if commentDisplay != "" {
		parts = append(parts, commentDisplay)
	}
	parts = append(parts, footer)
	bg := lipgloss.JoinVertical(lipgloss.Left, parts...)

	var content string
	if m.showHelp {
		dimmed := dimBackground(bg)
		modal := renderHelpModal()
		modalW := lipgloss.Width(modal)
		modalH := lipgloss.Height(modal)
		x := (m.width - modalW) / 2
		y := (m.height - modalH) / 2
		bgLayer := lipgloss.NewLayer(dimmed)
		fgLayer := lipgloss.NewLayer(modal).X(x).Y(y).Z(1)
		content = lipgloss.NewCompositor(bgLayer, fgLayer).Render()
	} else {
		content = bg
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func dimBackground(s string) string {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#504945"))
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		plain := ansiRe.ReplaceAllString(line, "")
		lines[i] = dimStyle.Render(plain)
	}
	return strings.Join(lines, "\n")
}

func renderSeparator(height int) string {
	var sb strings.Builder
	for i := 0; i < height; i++ {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString("│")
	}
	return separatorStyle.Render(sb.String())
}

func jumpTargetIcon(target int) string {
	switch target {
	case jumpChanged:
		return changedGutterStyle.Render("▲")
	case jumpImportant:
		return gutterStyle.Render("★")
	case jumpComments:
		return commentStyle.Render("◆")
	default:
		return unreviewedGutterStyle.Render("●")
	}
}

func renderFooter(width, jumpTarget int) string {
	left := "  ? help"
	icon := jumpTargetIcon(jumpTarget)
	var right string
	if BuildTime != "" {
		if t, err := time.Parse("2006-01-02T15:04", BuildTime); err == nil {
			now := time.Now()
			if t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day() {
				right = "build " + t.Format("15:04")
			} else {
				right = "build " + t.Format("2006-01-02 15:04")
			}
		}
	}
	leftW := len(left)
	iconW := lipgloss.Width(icon)
	rightW := len(right)
	padLeft := (width-iconW)/2 - leftW
	if padLeft < 1 {
		padLeft = 1
	}
	padRight := width - leftW - padLeft - iconW - rightW
	if padRight < 1 {
		padRight = 1
	}
	border := separatorStyle.Render(strings.Repeat("─", width))
	line := left + strings.Repeat(" ", padLeft) + icon + strings.Repeat(" ", padRight) + right
	return border + "\n" + footerStyle.Width(width).Render(line)
}

func renderHelpModal() string {
	k := func(s string) string { return helpKeyStyle.Render(s) }
	d := func(s string) string { return helpDescStyle.Render(s) }

	row := func(key, desc string) string {
		return k(fmt.Sprintf("%-15s", key)) + " " + d(desc)
	}

	lines := []string{
		helpTitleStyle.Render("Keyboard Shortcuts"),
		"",
		d("Code pane"),
		row("j / ↓", "Move selection down"),
		row("k / ↑", "Move selection up"),
		row("S-j / S-↓", "Grow selection"),
		row("S-k / S-↑", "Shrink selection"),
		row("h / ←", "Larger selection preset"),
		row("l / →", "Smaller selection preset"),
		row("n", "Next target"),
		row("N", "Prev target"),
		row("t", "Cycle target: ") + unreviewedGutterStyle.Render("●") + d(" unreviewed  ") + changedGutterStyle.Render("▲") + d(" changed  ") + gutterStyle.Render("★") + d(" important  ") + commentStyle.Render("◆") + d(" comments"),
		"",
		d("Files pane"),
		row("f", "Search / focus files pane"),
		row("↑ / ↓", "Navigate files"),
		row("esc", "Back to code pane"),
		row("Cmd+b", "Toggle files pane"),
		"",
		d("Review"),
		row("space", "Mark selection reviewed"),
		row("c", "Add comment to selection"),
		row("d", "Delete comment on selection"),
		row("r", "Cycle reviewers"),
		row("i", "Importance: ⠿ medium  █ high  ⠸ ignore"),
		"",
		d("Other"),
		row("e", "Open in $EDITOR"),
		row("q", "Quit"),
		"",
		d("Symbols"),
		k("filename*") + " " + d("Uncommitted changes"),
		"",
		k("?") + " " + d("close help"),
	}

	content := strings.Join(lines, "\n")
	// Center the last line
	contentLines := strings.Split(content, "\n")
	last := contentLines[len(contentLines)-1]
	contentW := 0
	for _, l := range contentLines {
		if w := lipgloss.Width(l); w > contentW {
			contentW = w
		}
	}
	lastW := lipgloss.Width(last)
	pad := (contentW - lastW) / 2
	if pad > 0 {
		contentLines[len(contentLines)-1] = strings.Repeat(" ", pad) + last
	}
	content = strings.Join(contentLines, "\n")

	return helpModalStyle.Render(content)
}
