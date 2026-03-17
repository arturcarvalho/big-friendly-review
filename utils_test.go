package main

import "testing"

func TestIsExcludedPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{".git", true},
		{".git/config", true},
		{".bfr", true},
		{".bfr/bfr.json", true},
		{".bfrignore", true},
		{"README.md", false},
		{"main.go", false},
		{"src/app.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isExcludedPath(tt.path)
			if got != tt.want {
				t.Errorf("isExcludedPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
