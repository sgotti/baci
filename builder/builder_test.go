package main

import "testing"

func TestBuildACIExcludeFunc(t *testing.T) {
	tests := []struct {
		path  string
		match bool
	}{
		{
			path:  "dev",
			match: false,
		},
		{
			path:  "dev/null",
			match: true,
		},
		{
			path:  "something/dev/null",
			match: false,
		},
		{
			path:  "baciblabla",
			match: false,
		},
		{
			path:  "baci",
			match: true,
		},
		{
			path:  "baci/root",
			match: true,
		},
		{
			path:  "baci/baci",
			match: true,
		},
		{
			path:  "something/baci",
			match: false,
		},
	}

	excludeFunc := NewExcludeFunc(excludePaths)
	for _, tt := range tests {
		match, err := excludeFunc(tt.path, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if match != tt.match {
			t.Errorf("unexpected match value for %s, want: %t, got: %t", tt.path, tt.match, match)
		}

	}

}
