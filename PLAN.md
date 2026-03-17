# BFR - File Review TUI

## Stack
- Go + Bubbletea v2 + Lipgloss
- go-gitignore (filter ignored files)
- Chroma (syntax highlighting)

## Layout

Two-pane TUI: Files (left) | File (right) + footer

### Left Pane (Files)
- Header: "Files" — light gray if focused, dark gray if not
- List of files in current dir (filename only, no path)
- j/k scrolls list when focused

### Right Pane (File)
- Header: relative path+filename (e.g. `store/main.go`) — light gray if focused, dark gray if not
- Shows file contents of selected file
- Binary files → warning message instead of content
- j/k scrolls content when focused

### Footer
- Hotkey descriptions: `tab` switch pane · `j/k` navigate · `q` quit

## Hotkeys
| Key | Action |
|-----|--------|
| `tab` | Swap focus between panes |
| `j` | Move down (list or scroll) |
| `k` | Move up (list or scroll) |
| `q` | Quit |

## File Structure
```
main.go          — entrypoint, tea.Program setup
model.go         — root model, Update, View
pane_files.go    — left pane: file list model
pane_file.go     — right pane: file content viewer
styles.go        — lipgloss styles (headers, panes, footer)
utils.go         — binary detection, file reading, syntax highlight via chroma
```

## Binary Detection
- Read first 512 bytes, check for null bytes → binary

## Syntax Highlighting
- Chroma lexer by filename extension
- Render highlighted lines for right pane

## Misc
1. Recurse into subdirectories
2. go-gitignore filtering is on by default
3. If file has more than 10K rows, show warning
