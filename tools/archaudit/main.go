// Command archaudit enforces the import boundaries between neat's layers
// (docs/superpowers/specs/2026-11-25-cli-interface-design.md, "Import-boundary
// check").
//
// Every top-level directory with Go files has a role:
//
//	adapters     mcp, cli, rest
//	composition  cmd, commands
//	sources      esi, sde, config, creds, sso, analytics
//	pure         nav, cargo, market, dogma, industry
//	contract     operation
//	skipped      tools, docs, claude_resources, bin, hidden and testdata dirs
//	domain       every other directory (character, universe, loyalty, ...)
//
// It reports, sorted by file then import:
//
//	DOMAIN_IMPORTS_ADAPTER  a domain, pure or operation package imports mcp,
//	                        cli, commands, rest or cmd, the MCP SDK, urfave/cli
//	                        or net/http; or a source package imports mcp, cli,
//	                        commands or rest. Sources may use net/http and the
//	                        MCP SDK (esi and sso speak HTTP, analytics holds
//	                        the MCP middleware).
//	CLI_IMPORTS_DOMAIN      cli imports a module package other than operation.
//
// The matching rule for mcp (it may import domain packages) is deliberately not
// enforced yet: mcp legitimately imports character and universe while areas
// migrate to operations; a later plan enforces it.
//
// Imports are resolved against the module path in go.mod. Test files are
// skipped.
//
// Usage: go run ./tools/archaudit [root]   (root defaults to ".")
package main

import (
	"bufio"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type role int

const (
	roleSkip role = iota
	roleAdapterMCP
	roleAdapterCLI
	roleAdapterOther
	roleComposition
	roleSource
	rolePure
	roleContract
	roleDomain
)

var roles = map[string]role{
	"mcp": roleAdapterMCP, "cli": roleAdapterCLI, "rest": roleAdapterOther,
	"cmd": roleComposition, "commands": roleComposition,
	"esi": roleSource, "sde": roleSource, "config": roleSource,
	"creds": roleSource, "sso": roleSource, "analytics": roleSource,
	"nav": rolePure, "cargo": rolePure, "market": rolePure,
	"dogma": rolePure, "industry": rolePure,
	"operation": roleContract,
	"tools":     roleSkip, "docs": roleSkip, "claude_resources": roleSkip, "bin": roleSkip,
}

func roleOf(top string) role {
	if r, ok := roles[top]; ok {
		return r
	}
	return roleDomain
}

// Module packages no domain, pure, source or contract package may import.
var (
	adaptersAndRoots = []string{"mcp", "cli", "commands", "rest", "cmd"}
	// A source package may import none of the module's adapters or command tree.
	adapters = []string{"mcp", "cli", "commands", "rest"}
	// External import prefixes barred from domain, pure and operation packages.
	bannedExternal = []string{
		"github.com/modelcontextprotocol/go-sdk",
		"github.com/urfave/cli",
		"net/http",
	}
)

// Finding is one audit line.
type Finding struct {
	Kind   string // DOMAIN_IMPORTS_ADAPTER or CLI_IMPORTS_DOMAIN
	File   string // slash-separated, relative to the audit root
	Import string // the import path
}

func (f Finding) String() string { return fmt.Sprintf("%s %s %s", f.Kind, f.File, f.Import) }

// modulePath reads the module path from root/go.mod.
func modulePath(root string) (string, error) {
	p := filepath.Join(root, "go.mod")
	f, err := os.Open(p)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", p, err)
	}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), "module "); ok {
			if err := f.Close(); err != nil {
				return "", fmt.Errorf("close %s: %w", p, err)
			}
			return strings.Trim(strings.TrimSpace(rest), `"`), nil
		}
	}
	scanErr := sc.Err()
	if err := f.Close(); err != nil && scanErr == nil {
		scanErr = err
	}
	if scanErr != nil {
		return "", fmt.Errorf("read %s: %w", p, scanErr)
	}
	return "", fmt.Errorf("%s: no module line", p)
}

// hasPathPrefix reports whether p is prefix or lies beneath it.
func hasPathPrefix(p, prefix string) bool {
	return p == prefix || strings.HasPrefix(p, prefix+"/")
}

// modulePackage returns the top-level directory of a module-internal import,
// or false when the import is not inside the module.
func modulePackage(imp, mod string) (string, bool) {
	rest, ok := strings.CutPrefix(imp, mod+"/")
	if !ok {
		return "", false
	}
	top, _, _ := strings.Cut(rest, "/")
	return top, true
}

func inList(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// Audit scans every non-test .go file under root.
func Audit(root string) ([]Finding, error) {
	mod, err := modulePath(root)
	if err != nil {
		return nil, fmt.Errorf("audit %s: %w", root, err)
	}
	fset := token.NewFileSet()
	var out []Finding
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", p, err)
		}
		if d.IsDir() {
			hidden := p != root && strings.HasPrefix(d.Name(), ".")
			if d.Name() == "testdata" && p != root || d.Name() == "vendor" || hidden {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return fmt.Errorf("relative path of %s: %w", p, err)
		}
		rel = filepath.ToSlash(rel)
		top, _, nested := strings.Cut(rel, "/")
		if !nested {
			return nil // files at the module root belong to no layer
		}
		r := roleOf(top)
		if r == roleSkip {
			return nil
		}
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			return fmt.Errorf("parse: %w", err) // the parser's error names the file
		}
		for _, im := range f.Imports {
			imp := strings.Trim(im.Path.Value, `"`)
			if kind := classify(r, imp, mod); kind != "" {
				out = append(out, Finding{Kind: kind, File: rel, Import: imp})
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("audit %s: %w", root, err)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Import < out[j].Import
	})
	return out, nil
}

// classify returns the finding kind for one import in a package of role r, or
// "" when the import is allowed.
func classify(r role, imp, mod string) string {
	pkg, internal := modulePackage(imp, mod)
	switch r {
	case roleDomain, rolePure, roleContract:
		if internal && inList(adaptersAndRoots, pkg) {
			return "DOMAIN_IMPORTS_ADAPTER"
		}
		for _, b := range bannedExternal {
			if hasPathPrefix(imp, b) {
				return "DOMAIN_IMPORTS_ADAPTER"
			}
		}
	case roleSource:
		if internal && inList(adapters, pkg) {
			return "DOMAIN_IMPORTS_ADAPTER"
		}
	case roleAdapterCLI:
		// cli may use the standard library, third-party libraries and
		// operation; any other module package is a violation.
		if internal && pkg != "operation" {
			return "CLI_IMPORTS_DOMAIN"
		}
	}
	return ""
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	findings, err := Audit(root)
	if err == nil {
		err = printFindings(os.Stdout, findings)
		if err == nil && len(findings) > 0 {
			os.Exit(1)
		}
	}
	if err != nil {
		// If stderr itself fails there is nowhere left to say so, so exit 2.
		if _, werr := fmt.Fprintln(os.Stderr, "archaudit:", err); werr != nil {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

// printFindings writes one line per finding to w.
func printFindings(w io.Writer, findings []Finding) error {
	for _, f := range findings {
		if _, err := fmt.Fprintln(w, f); err != nil {
			return fmt.Errorf("print findings: %w", err)
		}
	}
	return nil
}
