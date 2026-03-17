package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func marksPath() string {
	return filepath.Join(".bfr", "bfr.json")
}

func loadMarks() ([]FileMarks, error) {
	data, err := os.ReadFile(marksPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var marks []FileMarks
	if err := json.Unmarshal(data, &marks); err != nil {
		return nil, err
	}
	return marks, nil
}

func saveMarks(marks []FileMarks) error {
	if err := os.MkdirAll(".bfr", 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(marks, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(marksPath(), data, 0644); err != nil {
		return err
	}
	if err := updateBadge(marks); err != nil {
		return err
	}
	return nil
}

type badgeJSON struct {
	SchemaVersion int    `json:"schemaVersion"`
	Label         string `json:"label"`
	Message       string `json:"message"`
	Color         string `json:"color"`
}

func updateBadge(marks []FileMarks) error {
	pct := overallReviewedPercent(marks)
	color := "red"
	if pct >= 80 {
		color = "green"
	} else if pct >= 50 {
		color = "yellow"
	}

	// Build message: "74% (Artur: 60% · Jane: 40%)"
	stats := reviewerStats(marks)
	var names []string
	for name := range stats {
		names = append(names, name)
	}
	sort.Strings(names)

	msg := fmt.Sprintf("%d%%", pct)
	if len(names) > 0 {
		var parts []string
		for _, name := range names {
			parts = append(parts, fmt.Sprintf("%s: %d%%", name, stats[name]))
		}
		msg += " (" + strings.Join(parts, " · ") + ")"
	}

	badge := badgeJSON{
		SchemaVersion: 1,
		Label:         "Human Reviewed",
		Message:       msg,
		Color:         color,
	}
	data, err := json.MarshalIndent(badge, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(".bfr", "badge.json"), data, 0644)
}

func reviewerStats(marks []FileMarks) map[string]int {
	// Collect all reviewer names
	names := make(map[string]bool)
	for _, fm := range marks {
		for name := range fm.Reviewers {
			names[name] = true
		}
	}

	stats := make(map[string]int)
	for name := range names {
		var reviewed, total int
		for _, fm := range marks {
			segs := fm.Reviewers[name]
			for _, s := range segs {
				lines := s.EndLine - s.StartLine + 1
				total += lines
				if s.State == StateReviewed {
					reviewed += lines
				}
			}
		}
		if total > 0 {
			stats[name] = reviewed * 100 / total
		} else {
			stats[name] = 0
		}
	}
	return stats
}

func gitHeadCommit() (string, error) {
	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		return "", fmt.Errorf("current directory is not a git repository")
	}
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git repository has no commits yet")
	}
	return strings.TrimSpace(string(out)), nil
}

func gitDiffHunks(oldCommit, newCommit, path string) ([]Hunk, error) {
	out, err := exec.Command("git", "diff", "--unified=0", oldCommit, newCommit, "--", path).Output()
	if err != nil {
		return nil, err
	}

	var hunks []Hunk
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "@@") {
			h, err := parseHunkHeader(line)
			if err != nil {
				continue
			}
			hunks = append(hunks, h)
		}
	}
	return hunks, nil
}

var hunkRe = regexp.MustCompile(`@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

func parseHunkHeader(line string) (Hunk, error) {
	m := hunkRe.FindStringSubmatch(line)
	if m == nil {
		return Hunk{}, fmt.Errorf("invalid hunk header: %s", line)
	}

	oldStart, err := strconv.Atoi(m[1])
	if err != nil {
		return Hunk{}, fmt.Errorf("invalid old start in hunk: %s", line)
	}
	oldCount := 1
	if m[2] != "" {
		oldCount, err = strconv.Atoi(m[2])
		if err != nil {
			return Hunk{}, fmt.Errorf("invalid old count in hunk: %s", line)
		}
	}
	newStart, err := strconv.Atoi(m[3])
	if err != nil {
		return Hunk{}, fmt.Errorf("invalid new start in hunk: %s", line)
	}
	newCount := 1
	if m[4] != "" {
		newCount, err = strconv.Atoi(m[4])
		if err != nil {
			return Hunk{}, fmt.Errorf("invalid new count in hunk: %s", line)
		}
	}

	return Hunk{
		OldStart: oldStart,
		OldCount: oldCount,
		NewStart: newStart,
		NewCount: newCount,
	}, nil
}

func initOrUpdateMarks(entries []fileEntry) ([]FileMarks, error) {
	marks, err := loadMarks()
	if err != nil {
		return nil, err
	}

	headCommit, err := gitHeadCommit()
	if err != nil {
		return nil, err
	}

	marksByPath := make(map[string]*FileMarks)
	for i := range marks {
		marksByPath[marks[i].Path] = &marks[i]
	}

	userName := gitUserName()

	var result []FileMarks
	for _, entry := range entries {
		if isBinary(entry.relPath) {
			continue
		}
		_, lineCount, err := readFileContent(entry.relPath)
		if err != nil || lineCount > 10000 {
			continue
		}

		existing, found := marksByPath[entry.relPath]
		if !found {
			result = append(result, FileMarks{
				Path:               entry.relPath,
				FileName:           entry.name,
				Commit:             headCommit,
				Reviewers:          map[string][]Segment{userName: newFileSegments(lineCount)},
				ImportanceSegments: newFileImportanceSegments(lineCount),
			})
		} else if existing.Commit == headCommit {
			for name, segs := range existing.Reviewers {
				existing.Reviewers[name] = clampSegments(segs, lineCount)
			}
			existing.ImportanceSegments = clampImportanceSegments(existing.ImportanceSegments, lineCount)
			result = append(result, *existing)
		} else {
			hunks, err := gitDiffHunks(existing.Commit, headCommit, entry.relPath)
			if err != nil {
				result = append(result, FileMarks{
					Path:               entry.relPath,
					FileName:           entry.name,
					Commit:             headCommit,
					Reviewers:          map[string][]Segment{userName: newFileSegments(lineCount)},
					ImportanceSegments: newFileImportanceSegments(lineCount),
				})
				continue
			}

			reviewers := make(map[string][]Segment)
			for name, segs := range existing.Reviewers {
				if len(hunks) > 0 {
					reviewers[name] = clampSegments(applyHunks(segs, hunks), lineCount)
				} else {
					reviewers[name] = clampSegments(segs, lineCount)
				}
			}

			result = append(result, FileMarks{
				Path:               entry.relPath,
				FileName:           entry.name,
				Commit:             headCommit,
				Reviewers:          reviewers,
				ImportanceSegments: clampImportanceSegments(existing.ImportanceSegments, lineCount),
			})
		}
	}

	if err := saveMarks(result); err != nil {
		return nil, err
	}

	return result, nil
}
