package main

import "testing"

func TestIsExempt(t *testing.T) {
	exempt := []string{
		"README.md",
		"CLAUDE.md",
		"reference/cli-interface.md",
		"docs/reference/esi-coverage-roadmap.md",
		"docs/superpowers/plans/2026-07-28-x.md",
		".github/workflows/test.yml",
		".claude/settings.local.json",
		".gitignore",
		"VERSION",
		"sde/sync_test.go",
	}
	for _, p := range exempt {
		if !isExempt(p) {
			t.Errorf("isExempt(%q) = false, want true (should not require a bump)", p)
		}
	}

	significant := []string{
		"cmd/main.go",
		"sde/sync.go",
		"go.mod",
		"go.sum",
		".goreleaser.yaml",
		"claude_resources/neat/.mcp.json",
		"claude_resources/neat/.claude-plugin/plugin.json",
		"claude_resources/neat/CHANGELOG.md",
	}
	for _, p := range significant {
		if isExempt(p) {
			t.Errorf("isExempt(%q) = true, want false (should require a bump)", p)
		}
	}
}

func TestBumped(t *testing.T) {
	cases := []struct {
		base, head string
		want       bool
	}{
		{"0.1.0", "0.2.0", true},
		{"0.1.0", "0.1.1", true},
		{"v0.1.0", "v0.2.0", true}, // tolerate a leading v
		{"0.1.0", "0.1.0", false},  // unchanged
		{"0.2.0", "0.1.0", false},  // downgrade is not a bump
		{"", "0.1.0", true},        // base had no VERSION yet (treated as 0.0.0)
		{"0.1.0", "", false},       // missing/invalid head is not a bump
		{"0.9.0", "0.10.0", true},  // semver, not lexical (10 > 9)
	}
	for _, c := range cases {
		if got := bumped(c.base, c.head); got != c.want {
			t.Errorf("bumped(%q, %q) = %v, want %v", c.base, c.head, got, c.want)
		}
	}
}

func TestDecide(t *testing.T) {
	goChange := []string{"sde/sync.go"}
	docsOnly := []string{"README.md", "docs/x.md", "reference/cli-interface.md"}

	cases := []struct {
		name    string
		in      input
		want    decision
		wantSub string
	}{
		{"docs only, no bump", input{ChangedFiles: docsOnly, Base: "0.1.0", Head: "0.1.0"}, pass, "no release-significant"},
		{"code, no bump", input{ChangedFiles: goChange, Base: "0.1.0", Head: "0.1.0"}, fail, "bump VERSION"},
		{"code, bumped", input{ChangedFiles: goChange, Base: "0.1.0", Head: "0.2.0"}, pass, "bumped"},
		{"test-only change", input{ChangedFiles: []string{"sde/sync_test.go"}, Base: "0.1.0", Head: "0.1.0"}, pass, "no release-significant"},
		{"mixed docs+code, no bump", input{ChangedFiles: []string{"README.md", "sde/sync.go"}, Base: "0.1.0", Head: "0.1.0"}, fail, "bump VERSION"},
		{"plugin content, no bump", input{ChangedFiles: []string{"claude_resources/neat/.mcp.json"}, Base: "0.1.0", Head: "0.1.0"}, fail, "bump VERSION"},
		{"equal version with v prefix", input{ChangedFiles: goChange, Base: "0.1.0", Head: "v0.1.0"}, fail, "bump VERSION"},
	}
	for _, c := range cases {
		got, reason := decide(c.in)
		if got != c.want {
			t.Errorf("%s: decide = %v, want %v (reason: %s)", c.name, got, c.want, reason)
		}
		if c.wantSub != "" && !contains(reason, c.wantSub) {
			t.Errorf("%s: reason %q missing %q", c.name, reason, c.wantSub)
		}
	}
}

func TestSingleLine(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"no newline here", "no newline here"},
		{"line one\nline two", "line one line two"},
		{"crlf\r\nsplit", "crlf split"},
		{"lone\rcr", "lone cr"},
	}
	for _, c := range cases {
		if got := singleLine(c.in); got != c.want {
			t.Errorf("singleLine(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestIsMissingPathErr(t *testing.T) {
	cases := []struct {
		name, stderr string
		want         bool
	}{
		{"missing-path, absent from worktree too", "fatal: path 'VERSION' does not exist in 'origin/main'", true},
		{"missing-path, present on disk (VERSION's actual case)", "fatal: path 'VERSION' exists on disk, but not in 'origin/main'", true},
		{"bad ref", "fatal: invalid object name 'origin/nonexistent-branch'", false},
		{"no checkout / not a git repo", "fatal: not a git repository (or any of the parent directories): .git", false},
		{"empty", "", false},
	}
	for _, c := range cases {
		if got := isMissingPathErr(c.stderr); got != c.want {
			t.Errorf("%s: isMissingPathErr(%q) = %v, want %v", c.name, c.stderr, got, c.want)
		}
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
