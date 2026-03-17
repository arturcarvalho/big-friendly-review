package main

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	chromaStyles "github.com/alecthomas/chroma/v2/styles"
	gitignore "github.com/sabhiram/go-gitignore"
)

func loadFiles(root string) ([]fileEntry, error) {
	gi, _ := gitignore.CompileIgnoreFile(filepath.Join(root, ".bfrignore"))

	var entries []fileEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		if rel == "." {
			return nil
		}

		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			if d.Name() == ".bfr" {
				return filepath.SkipDir
			}
			if gi != nil && gi.MatchesPath(rel) {
				return filepath.SkipDir
			}
			return nil
		}

		if d.Name() == ".bfrignore" {
			return nil
		}

		if gi != nil && gi.MatchesPath(rel) {
			return nil
		}

		if isBinary(rel) {
			return nil
		}

		entries = append(entries, fileEntry{
			relPath: rel,
			name:    d.Name(),
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].relPath < entries[j].relPath
	})

	return entries, nil
}

func detectDirtyPaths() []string {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(bytes.TrimSpace(out)) == 0 {
		return nil
	}
	var paths []string
	for _, line := range strings.Split(string(out), "\n") {
		if len(line) < 4 {
			continue
		}
		path := line[3:]
		if !isExcludedPath(path) {
			paths = append(paths, path)
		}
	}
	return paths
}

var cachedIgnore *gitignore.GitIgnore
var cachedIgnoreLoaded bool

func isExcludedPath(path string) bool {
	if path == ".git" || strings.HasPrefix(path, ".git/") {
		return true
	}
	if path == ".bfr" || strings.HasPrefix(path, ".bfr/") {
		return true
	}
	if path == ".bfrignore" {
		return true
	}
	if !cachedIgnoreLoaded {
		cachedIgnore, _ = gitignore.CompileIgnoreFile(".bfrignore")
		cachedIgnoreLoaded = true
	}
	if cachedIgnore != nil && cachedIgnore.MatchesPath(path) {
		return true
	}
	if isBinary(path) {
		return true
	}
	return false
}

func isBinary(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil {
		return false
	}

	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}
	return false
}

func readFileContent(path string) (string, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	content := string(data)
	lineCount := len(strings.Split(content, "\n"))
	return content, lineCount, nil
}

func highlightCode(source, filename string) string {
	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := chromaStyles.Get("monokai")
	if style == nil {
		style = chromaStyles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, source)
	if err != nil {
		return source
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return source
	}

	return buf.String()
}
