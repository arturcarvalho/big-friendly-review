# Architecture

## Recommended Review Order

1. `go.mod` — dependencies & module name
2. `main.go` — entry point
3. `model.go` — core app model/state
4. `styles.go` — UI styling
5. `blocks.go` — block-level data structures
6. `marks.go` → `marks_io.go` — marks logic then I/O
7. `pane_file.go` → `pane_files.go` — single file pane, then file list pane
8. `utils.go` — helpers
9. Tests (`*_test.go`) — after understanding each module

Foundation → core logic → UI components → utilities → tests.
