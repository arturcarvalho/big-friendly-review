# Marks (Review Tracking) — Before/After Examples

## 1. Initial creation (no bfr.json exists)
**Before:** File doesn't exist.
**After:** (all files from files pane, each fully unreviewed, 1-based lines)
```json
[
  {
    "path": "main.go",
    "fileName": "main.go",
    "commit": "4a55246",
    "segments": [
      {"startLine": 1, "endLine": 25, "updatedAt": "2026-03-17T10:00:00Z", "state": "unreviewed"}
    ]
  },
  {
    "path": "utils.go",
    "fileName": "utils.go",
    "commit": "4a55246",
    "segments": [
      {"startLine": 1, "endLine": 128, "updatedAt": "2026-03-17T10:00:00Z", "state": "unreviewed"}
    ]
  }
]
```

## 2. User presses space on block (0-based block lines 5–12 → 1-based 6–13)
**Before:**
```json
{
  "path": "main.go", "commit": "4a55246",
  "segments": [
    {"startLine": 1, "endLine": 25, "state": "unreviewed"}
  ]
}
```
**After:** (segment splits, block range → reviewed)
```json
{
  "path": "main.go", "commit": "4a55246",
  "segments": [
    {"startLine": 1, "endLine": 5, "state": "unreviewed"},
    {"startLine": 6, "endLine": 13, "state": "reviewed"},
    {"startLine": 14, "endLine": 25, "state": "unreviewed"}
  ]
}
```

## 3. User presses space again on same block (all reviewed → unreviewed)
**Before:**
```json
"segments": [
  {"startLine": 1, "endLine": 5, "state": "unreviewed"},
  {"startLine": 6, "endLine": 13, "state": "reviewed"},
  {"startLine": 14, "endLine": 25, "state": "unreviewed"}
]
```
**After:** (reviewed → unreviewed, adjacent contiguous segments merge)
```json
"segments": [
  {"startLine": 1, "endLine": 25, "state": "unreviewed"}
]
```

## 4. Mixed states — block spans reviewed + unreviewed
**Before:** Block covers lines 10–25 (1-based), boundary at 15.
```json
"segments": [
  {"startLine": 1, "endLine": 15, "state": "reviewed"},
  {"startLine": 16, "endLine": 100, "state": "unreviewed"}
]
```
**After:** (mixed → all reviewed, merges with prefix)
```json
"segments": [
  {"startLine": 1, "endLine": 25, "state": "reviewed"},
  {"startLine": 26, "endLine": 100, "state": "unreviewed"}
]
```

## 5. File changed — 10 lines inserted at line 50
**Before (commit abc123):**
```json
{
  "path": "bla.go", "commit": "abc123",
  "segments": [
    {"startLine": 1, "endLine": 100, "state": "reviewed"},
    {"startLine": 101, "endLine": 200, "state": "unreviewed"}
  ]
}
```
**After (commit def456, 10 lines added after line 50):**
```json
{
  "path": "bla.go", "commit": "def456",
  "segments": [
    {"startLine": 1, "endLine": 50, "state": "reviewed"},
    {"startLine": 51, "endLine": 60, "state": "changed"},
    {"startLine": 61, "endLine": 110, "state": "reviewed"},
    {"startLine": 111, "endLine": 210, "state": "unreviewed"}
  ]
}
```

## 6. File changed — 5 lines deleted at line 20
**Before (commit abc123):**
```json
{
  "path": "bla.go", "commit": "abc123",
  "segments": [
    {"startLine": 1, "endLine": 100, "state": "reviewed"}
  ]
}
```
**After (commit def456, lines 20–24 deleted):**
```json
{
  "path": "bla.go", "commit": "def456",
  "segments": [
    {"startLine": 1, "endLine": 95, "state": "reviewed"}
  ]
}
```

## 7. File changed — 3 lines replaced with 7 at line 10
**Before:**
```json
"segments": [{"startLine": 1, "endLine": 50, "state": "reviewed"}]
```
**After:** (old lines removed, new lines marked changed, rest shifted +4)
```json
"segments": [
  {"startLine": 1, "endLine": 9, "state": "reviewed"},
  {"startLine": 10, "endLine": 16, "state": "changed"},
  {"startLine": 17, "endLine": 54, "state": "reviewed"}
]
```

## 8. New file added between commits
**Before:** File not in bfr.json.
**After:** Added as fully unreviewed with current HEAD commit.

## 9. File deleted between commits
**Before:** Entry exists in bfr.json.
**After:** Entry removed from bfr.json (file no longer in files pane).
