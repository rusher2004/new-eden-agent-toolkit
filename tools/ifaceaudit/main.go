// Command ifaceaudit lists the places where neat's code departs from the
// consumer-defined-interface standard in CLAUDE.md.
//
// It prints, sorted by file then line:
//
//	CONCRETE_PARAM  a parameter, struct field or variable outside a producer
//	                package whose type is a concrete client or store (pkg.Type:
//	                a type with methods or returned by a New... constructor, not
//	                a plain data struct) from a producer package (sde, esi, sso, creds,
//	                config, analytics).
//	CONCRETE_CTOR   a reference to a producer constructor (pkg.New...) outside
//	                a producer package: the place a concrete client is built.
//	PRODUCER_IFACE  an exported interface type declared inside a producer
//	                package (an unexported one is internal, not meant for callers,
//	                and is reported as INTERFACE).
//	INTERFACE       an interface type declared anywhere else (consumer-side).
//	TYPE_ONLY_FILE  a non-test file that declares a type but no func, method or
//	                var, or any file named *_iface.go: a type must live in the
//	                file of the code that uses it.
//
// Every interface line ends with used-by=, the packages that reference it.
// There is no type checker: matching is on selector expressions pkg.Type,
// resolved through each file's imports (so import aliases work). Test files
// and testdata directories are skipped. Results (function results) are not
// reported, except for function-typed variables (typed, or bound to a func
// literal), whose results are reported too.
//
// Usage: go run ./tools/ifaceaudit [root]   (root defaults to ".")
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// producers are the packages whose exported types are concrete clients and
// stores (or their data). Callers must depend on consumer-declared interfaces.
var producers = map[string]bool{
	"sde": true, "esi": true, "sso": true, "creds": true,
	"config": true, "analytics": true,
}

// Finding is one audit line.
type Finding struct {
	Kind    string // CONCRETE_PARAM, CONCRETE_CTOR, PRODUCER_IFACE, INTERFACE or TYPE_ONLY_FILE
	File    string // slash-separated, relative to the audit root
	Line    int
	Context string // enclosing func/struct.field/var, or pkg.Iface for interfaces
	Type    string // pkg.Type for CONCRETE_PARAM
	UsedBy  []string
}

func (f Finding) String() string {
	s := fmt.Sprintf("%s %s:%d %s", f.Kind, f.File, f.Line, f.Context)
	if f.Type != "" {
		s += " " + f.Type
	}
	if f.Kind == "PRODUCER_IFACE" || f.Kind == "INTERFACE" {
		s += " used-by=" + strings.Join(f.UsedBy, ",")
	}
	return s
}

type parsedFile struct {
	rel     string
	dir     string // slash-separated directory of the file, relative to the root
	pkg     string
	imports map[string]string // local name -> imported package name (last path element)
	ast     *ast.File
}

// Audit scans every non-test .go file under root.
func Audit(root string) ([]Finding, error) {
	fset := token.NewFileSet()
	var files []parsedFile
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", p, err)
		}
		if d.IsDir() {
			// Hidden directories (.git, .claude/worktrees, ...) can hold whole
			// copies of the repo; scanning them would double every finding.
			hidden := p != root && strings.HasPrefix(d.Name(), ".")
			if d.Name() == "testdata" && p != root || d.Name() == "vendor" || hidden {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			return fmt.Errorf("parse: %w", err) // the parser's error names the file
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return fmt.Errorf("relative path of %s: %w", p, err)
		}
		pf := parsedFile{rel: filepath.ToSlash(rel), dir: path.Dir(filepath.ToSlash(rel)), pkg: f.Name.Name, imports: map[string]string{}, ast: f}
		for _, im := range f.Imports {
			ip := strings.Trim(im.Path.Value, `"`)
			// Known limit: keyed by path.Base, so a versioned path (.../v2) or an
			// external package named like a producer is not resolved correctly.
			base := path.Base(ip)
			name := base
			if im.Name != nil {
				name = im.Name.Name
			}
			if name == "_" || name == "." {
				continue
			}
			pf.imports[name] = base
		}
		files = append(files, pf)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("audit %s: %w", root, err)
	}

	// Only clients and stores are reported as CONCRETE_PARAM, not plain data
	// structs: a producer type with methods, or one a New... constructor returns.
	clients := map[string]bool{}
	for _, pf := range files {
		if !producers[pf.pkg] {
			continue
		}
		for _, d := range pf.ast.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fd.Recv != nil && len(fd.Recv.List) > 0 {
				clients[pf.pkg+"."+recvName(fd.Recv.List[0].Type)] = true
				continue
			}
			if strings.HasPrefix(fd.Name.Name, "New") && fd.Type.Results != nil {
				for _, r := range fd.Type.Results.List {
					if id, ok := baseIdent(r.Type); ok {
						clients[pf.pkg+"."+id] = true
					}
				}
			}
		}
	}

	var out []Finding
	// Keyed by directory, not package name: several directories share a package
	// name (every tool is package main; sde and sdetest), and keying on the name
	// would let one overwrite another's interfaces and merge their usage.
	type ifaceKey struct{ dir, name string }
	pkgDirs := map[string][]string{} // package name -> directories declaring it
	for _, pf := range files {
		if !slices.Contains(pkgDirs[pf.pkg], pf.dir) {
			pkgDirs[pf.pkg] = append(pkgDirs[pf.pkg], pf.dir)
		}
	}
	ifaces := map[ifaceKey]*Finding{}
	for _, pf := range files {
		inProducer := producers[pf.pkg]
		if fd, ok := typeOnlyFile(fset, pf); ok {
			out = append(out, fd)
		}
		for _, d := range pf.ast.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if inProducer {
					continue
				}
				name := d.Name.Name
				if d.Recv != nil && len(d.Recv.List) > 0 {
					name = recvName(d.Recv.List[0].Type) + "." + name
				}
				for _, p := range d.Type.Params.List {
					out = append(out, concrete(fset, pf, clients, name, p.Type, false)...)
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						switch t := s.Type.(type) {
						case *ast.InterfaceType:
							k := "INTERFACE"
							if inProducer && ast.IsExported(s.Name.Name) {
								k = "PRODUCER_IFACE"
							}
							fd := &Finding{Kind: k, File: pf.rel, Line: fset.Position(s.Pos()).Line, Context: pf.pkg + "." + s.Name.Name}
							ifaces[ifaceKey{pf.dir, s.Name.Name}] = fd
						case *ast.StructType:
							if inProducer {
								continue
							}
							for _, fld := range t.Fields.List {
								names := []string{"(embedded)"}
								if len(fld.Names) > 0 {
									names = nil
									for _, n := range fld.Names {
										names = append(names, n.Name)
									}
								}
								for _, n := range names {
									out = append(out, concrete(fset, pf, clients, s.Name.Name+"."+n, fld.Type, false)...)
								}
							}
						}
					case *ast.ValueSpec:
						if inProducer {
							continue
						}
						for _, n := range s.Names {
							if s.Type != nil {
								out = append(out, concrete(fset, pf, clients, n.Name, s.Type, true)...)
							}
							for _, v := range s.Values {
								if fl, ok := v.(*ast.FuncLit); ok {
									out = append(out, concrete(fset, pf, clients, n.Name, fl.Type, true)...)
								}
							}
						}
					}
				}
			}
		}
	}

	// used-by: packages referencing an interface, via pkg.Name selectors or
	// (within its own package) a bare identifier.
	used := map[ifaceKey]map[string]bool{}
	for k := range ifaces {
		used[k] = map[string]bool{}
	}
	for _, pf := range files {
		ast.Inspect(pf.ast, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.SelectorExpr:
				if x, ok := n.X.(*ast.Ident); ok {
					if base, ok := pf.imports[x.Name]; ok {
						for _, dir := range pkgDirs[base] {
							if u, ok := used[ifaceKey{dir, n.Sel.Name}]; ok && dir != pf.dir {
								u[pf.pkg] = true
							}
						}
					}
				}
			case *ast.Field:
				// bare identifier uses of a same-package interface type
				if id, ok := baseIdent(n.Type); ok {
					k := ifaceKey{pf.dir, id}
					if u, ok := used[k]; ok {
						u[pf.pkg] = true
					}
				}
			}
			return true
		})
	}
	for _, pf := range files {
		if producers[pf.pkg] {
			continue
		}
		ast.Inspect(pf.ast, func(n ast.Node) bool {
			se, ok := n.(*ast.SelectorExpr)
			if !ok || !strings.HasPrefix(se.Sel.Name, "New") {
				return true
			}
			if x, ok := se.X.(*ast.Ident); ok {
				if base, ok := pf.imports[x.Name]; ok && producers[base] {
					out = append(out, Finding{
						Kind: "CONCRETE_CTOR", File: pf.rel, Line: fset.Position(se.Pos()).Line,
						Context: pf.pkg, Type: base + "." + se.Sel.Name,
					})
				}
			}
			return true
		})
	}
	for k, fd := range ifaces {
		for p := range used[k] {
			fd.UsedBy = append(fd.UsedBy, p)
		}
		sort.Strings(fd.UsedBy)
		if len(fd.UsedBy) == 0 {
			fd.UsedBy = []string{"-"}
		}
		out = append(out, *fd)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Context != b.Context {
			return a.Context < b.Context
		}
		return a.Type < b.Type
	})
	return out, nil
}

// typeOnlyFile reports a file that exists only to declare types: it declares at
// least one type and no func, method or var (consts do not count as code), or it
// is named *_iface.go. A type belongs in the file of the code that uses it.
func typeOnlyFile(fset *token.FileSet, pf parsedFile) (Finding, bool) {
	firstType := 0
	hasCode := false
	for _, d := range pf.ast.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			hasCode = true
		case *ast.GenDecl:
			switch d.Tok {
			case token.VAR:
				hasCode = true
			case token.TYPE:
				if firstType == 0 {
					firstType = fset.Position(d.Pos()).Line
				}
			}
		}
	}
	isIface := strings.HasSuffix(pf.rel, "_iface.go")
	if !isIface && (firstType == 0 || hasCode) {
		return Finding{}, false
	}
	if firstType == 0 {
		firstType = 1
	}
	return Finding{Kind: "TYPE_ONLY_FILE", File: pf.rel, Line: firstType, Context: pf.pkg}, true
}

func baseIdent(e ast.Expr) (string, bool) {
	for {
		switch t := e.(type) {
		case *ast.StarExpr:
			e = t.X
		case *ast.ArrayType:
			e = t.Elt
		case *ast.Ident:
			return t.Name, true
		default:
			return "", false
		}
	}
}

func recvName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return recvName(t.X)
	case *ast.IndexExpr:
		return recvName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return "?"
}

// concrete reports every pkg.Type selector in a type expression where pkg is a
// producer package. Function types contribute their parameters, and their results when results is set.
func concrete(fset *token.FileSet, pf parsedFile, clients map[string]bool, ctx string, e ast.Expr, results bool) []Finding {
	var out []Finding
	var walk func(ast.Expr)
	walk = func(e ast.Expr) {
		switch t := e.(type) {
		case *ast.StarExpr:
			walk(t.X)
		case *ast.ArrayType:
			walk(t.Elt)
		case *ast.Ellipsis:
			walk(t.Elt)
		case *ast.MapType:
			walk(t.Key)
			walk(t.Value)
		case *ast.ChanType:
			walk(t.Value)
		case *ast.IndexExpr:
			walk(t.X)
			walk(t.Index)
		case *ast.IndexListExpr:
			walk(t.X)
			for _, i := range t.Indices {
				walk(i)
			}
		case *ast.FuncType:
			for _, p := range t.Params.List {
				walk(p.Type)
			}
			if results && t.Results != nil {
				for _, r := range t.Results.List {
					walk(r.Type)
				}
			}
		case *ast.SelectorExpr:
			if x, ok := t.X.(*ast.Ident); ok {
				if base, ok := pf.imports[x.Name]; ok && producers[base] && clients[base+"."+t.Sel.Name] {
					out = append(out, Finding{
						Kind: "CONCRETE_PARAM", File: pf.rel, Line: fset.Position(t.Pos()).Line,
						Context: ctx, Type: base + "." + t.Sel.Name,
					})
				}
			}
		}
	}
	walk(e)
	return out
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	findings, err := Audit(root)
	if err == nil {
		err = printFindings(os.Stdout, findings)
	}
	if err != nil {
		// main is the last place to report. If stderr itself fails there is
		// nowhere left to say so, so exit 2 (not 1) carries that.
		if _, werr := fmt.Fprintln(os.Stderr, "ifaceaudit:", err); werr != nil {
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
