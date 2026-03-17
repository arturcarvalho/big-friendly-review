package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Set via: go build -ldflags "-X main.BuildTime=$(date +%Y-%m-%dT%H:%M)"
var BuildTime string

func main() {
	showComments := flag.Bool("comments", false, "output comments and exit")
	flag.Parse()

	if *showComments {
		printComments()
		return
	}

	m, err := newModel(detectDirtyPaths())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func printComments() {
	marks, err := loadMarks()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	found := false
	for _, fm := range marks {
		for _, c := range fm.Comments {
			found = true
			date := c.CreatedAt
			if i := strings.IndexByte(date, 'T'); i >= 0 {
				date = date[:i]
			}
			if c.StartLine == c.EndLine {
				fmt.Printf("%s:%d (%s, %s)\n", fm.Path, c.StartLine, c.Author, date)
			} else {
				fmt.Printf("%s:%d-%d (%s, %s)\n", fm.Path, c.StartLine, c.EndLine, c.Author, date)
			}
			fmt.Printf("  %s\n\n", c.Text)
		}
	}
	if !found {
		fmt.Println("No comments.")
	}
}
