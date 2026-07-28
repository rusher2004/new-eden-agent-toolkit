// Command versiongate is the CI gate that keeps the VERSION file honest: a PR
// that changes release-significant files (the shipped binary or the plugin)
// must bump VERSION, while docs/CI/test-only PRs may merge without a bump
// (and therefore without cutting a release -- see release.yml, which only
// tags when v<VERSION> doesn't yet exist).
//
// It classifies the PR's changed files against an exempt allowlist; anything
// NOT on the allowlist is release-significant (unknown paths default to
// significant -- fail-safe). The decision is deterministic, with no escape
// hatch:
//
//	not significant  OR  VERSION strictly-greater than base  -> pass
//	significant, not bumped                                  -> fail (bump it)
//
// The decision is written to GITHUB_OUTPUT (decision=, reason=) for the
// workflow to act on; main exits 0 unless it hits an internal error (exit 2),
// so the workflow owns how a failing decision is surfaced.
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/mod/semver"
)

type decision string

const (
	pass decision = "pass"
	fail decision = "fail"
)

type input struct {
	ChangedFiles []string
	Base         string // base branch VERSION value (may be empty if absent)
	Head         string // this PR's VERSION value
}

// isExempt reports whether a changed path is NOT release-significant: docs,
// CI/workflow config, editor/repo metadata, and test files don't ship in the
// binary or the plugin. Everything else (Go source, go.mod/sum, .goreleaser,
// and anything under claude_resources/ -- the plugin content the marketplace
// reads from the tree) requires a VERSION bump. Unlisted paths are significant.
func isExempt(path string) bool {
	switch {
	case strings.HasSuffix(path, "_test.go"):
		return true
	case strings.HasPrefix(path, "docs/"),
		strings.HasPrefix(path, "reference/"),
		strings.HasPrefix(path, ".github/"),
		strings.HasPrefix(path, ".claude/"):
		return true
	case path == ".gitignore", path == "VERSION":
		return true
	case strings.HasSuffix(path, ".md") && !strings.HasPrefix(path, "claude_resources/"):
		return true
	}
	return false
}

func anySignificant(files []string) bool {
	for _, f := range files {
		if f = strings.TrimSpace(f); f != "" && !isExempt(f) {
			return true
		}
	}
	return false
}

// ensureV canonicalizes a version to a valid semver "vX.Y.Z" string, or
// returns "" if it isn't valid semver.
func ensureV(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if !strings.HasPrefix(s, "v") {
		s = "v" + s
	}
	if !semver.IsValid(s) {
		return ""
	}
	return s
}

// bumped reports whether head is a strictly-greater semver than base. An
// empty base is treated as 0.0.0 (the PR that first introduces VERSION); an
// invalid head can never be a bump.
func bumped(base, head string) bool {
	h := ensureV(head)
	if h == "" {
		return false
	}
	b := ensureV(base)
	if b == "" {
		b = "v0.0.0"
	}
	return semver.Compare(h, b) > 0
}

func decide(in input) (decision, string) {
	if !anySignificant(in.ChangedFiles) {
		return pass, "no release-significant files changed; a VERSION bump is not required"
	}
	if bumped(in.Base, in.Head) {
		return pass, fmt.Sprintf("VERSION bumped %s -> %s", emptyAsNone(in.Base), in.Head)
	}
	return fail, fmt.Sprintf("release-significant files changed but VERSION was not bumped (still %s): bump VERSION", emptyAsNone(in.Base))
}

func emptyAsNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(none)"
	}
	return s
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "versiongate:", err)
		os.Exit(2)
	}
}

func run() error {
	base := os.Getenv("GITHUB_BASE_REF")
	if base == "" {
		base = "main"
	}

	headRaw, err := os.ReadFile("VERSION")
	if err != nil {
		return fmt.Errorf("read VERSION: %w", err)
	}
	head := strings.TrimSpace(string(headRaw))

	files, err := changedFiles(base)
	if err != nil {
		return fmt.Errorf("determine changed files: %w", err)
	}

	baseVer, err := baseVersion(base)
	if err != nil {
		return fmt.Errorf("determine base VERSION: %w", err)
	}

	in := input{
		ChangedFiles: files,
		Base:         baseVer,
		Head:         head,
	}

	d, reason := decide(in)
	fmt.Printf("versiongate: %s -- %s\n", d, reason)
	return writeOutputs(d, reason)
}

// changedFiles lists files changed on the PR branch since its merge base with
// the target branch. A git failure is returned as an error rather than
// defaulted to "no changes" -- the gate has no safe way to know whether an
// unreported change was significant, so it must not silently pass.
func changedFiles(base string) ([]string, error) {
	var stderr bytes.Buffer
	cmd := exec.Command("git", "diff", "--name-only", "origin/"+base+"...HEAD")
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff origin/%s...HEAD: %w (%s)", base, err, strings.TrimSpace(stderr.String()))
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

// baseVersion reads the VERSION file from the tip of the target branch. It
// returns ("", nil) when the target branch has no VERSION file yet (the PR
// that first introduces it) -- a legitimate case, distinguished from a real
// git failure (bad ref, no checkout, etc.) via git's own "does not exist" path
// error, since defaulting an unrecognized error to "" would let the gate
// treat a broken checkout the same as "first version ever".
func baseVersion(base string) (string, error) {
	var stderr bytes.Buffer
	cmd := exec.Command("git", "show", "origin/"+base+":VERSION")
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if isMissingPathErr(stderr.String()) {
			return "", nil
		}
		return "", fmt.Errorf("git show origin/%s:VERSION: %w (%s)", base, err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

// isMissingPathErr reports whether git's stderr describes "the path doesn't
// exist at this revision" (git show's message when a file was never
// committed on that branch), as opposed to some other failure (bad ref, no
// checkout, network/auth issue) that the caller must not paper over. git
// phrases this two ways depending on whether the path also exists in the
// current worktree: "does not exist in '<ref>'" when it doesn't, or "exists
// on disk, but not in '<ref>'" when it does -- VERSION is a normal tracked
// file, so a base branch that predates it always hits the second phrasing.
func isMissingPathErr(stderr string) bool {
	return strings.Contains(stderr, "does not exist in") || strings.Contains(stderr, "exists on disk, but not in")
}

// writeOutputs emits the decision for the workflow via GITHUB_OUTPUT (when set).
func writeOutputs(d decision, reason string) (err error) {
	path := os.Getenv("GITHUB_OUTPUT")
	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	// Surface a Close error (e.g. a failed flush, which would drop the write)
	// when nothing earlier already failed.
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	// GITHUB_OUTPUT is parsed as key=value per line, so a reason built from
	// file content (VERSION's Base/Head) must not be allowed to smuggle in an
	// embedded newline and inject a second key=value pair (e.g. a fake
	// "decision=pass"). singleLine enforces that regardless of what decide()
	// happened to interpolate.
	_, err = fmt.Fprintf(f, "decision=%s\nreason=%s\n", d, singleLine(reason))
	return err
}

// singleLine collapses any line breaks in s to spaces so it's safe to write
// as the value half of a GITHUB_OUTPUT key=value line.
func singleLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.ReplaceAll(s, "\r", " ")
}
