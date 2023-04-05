// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cgocall defines an Analyzer that detects some violations of
// the cgo pointer passing rules.
package cgocall

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"strconv"

	"github.com/Deng-Xian-Sheng/goplus-lsp/go/analysis"
	"github.com/Deng-Xian-Sheng/goplus-lsp/go/analysis/passes/internal/analysisutil"
)

const debug = false

const Doc = `detect some violations of the cgo pointer passing rules

Check for invalid cgo pointer passing.
This looks for code that uses cgo to call C code passing values
whose types are almost always invalid according to the cgo pointer
sharing rules.
Specifically, it warns about attempts to pass a Go chan, map, func,
or slice to C, either directly, or via a pointer, array, or struct.`

var Analyzer = &analysis.Analyzer{
	Name:             "cgocall",
	Doc:              Doc,
	RunDespiteErrors: true,
	Run:              run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	if !analysisutil.Imports(pass.Pkg, "runtime/cgo") {
		return nil, nil // doesn't use cgo
	}

	cgofiles, info, err := typeCheckCgoSourceFiles(pass.Fset, pass.Pkg, pass.Files, pass.TypesInfo, pass.TypesSizes)
	if err != nil {
		return nil, err
	}
	for _, f := range cgofiles {
		checkCgo(pass.Fset, f, info, pass.Reportf)
	}
	return nil, nil
}

func checkCgo(fset *token.FileSet, f *ast.File, info *types.Info, reportf func(token.Pos, string, ...interface{})) {
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Is this a C.f() call?
		var name string
		if sel, ok := analysisutil.Unparen(call.Fun).(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "C" {
				name = sel.Sel.Name
			}
		}
		if name == "" {
			return true // not a call we need to check
		}

		// A call to C.CBytes passes a pointer but is always safe.
		if name == "CBytes" {
			return true
		}

		if debug {
			log.Printf("%s: call to C.%s", fset.Position(call.Lparen), name)
		}

		for _, arg := range call.Args {
			if !typeOKForCgoCall(cgoBaseType(info, arg), make(map[types.Type]bool)) {
				reportf(arg.Pos(), "possibly passing Go type with embedded pointer to C")
				break
			}

			// Check for passing the address of a bad type.
			if conv, ok := arg.(*ast.CallExpr); ok && len(conv.Args) == 1 &&
				isUnsafePointer(info, conv.Fun) {
				arg = conv.Args[0]
			}
			if u, ok := arg.(*ast.UnaryExpr); ok && u.Op == token.AND {
				if !typeOKForCgoCall(cgoBaseType(info, u.X), make(map[types.Type]bool)) {
					reportf(arg.Pos(), "possibly passing Go type with embedded pointer to C")
					break
				}
			}
		}
		return true
	})
}

// typeCheckCgoSourceFiles returns type-checked syntax trees for the raw
// cgo files of a package (those that import "C"). Such files are not
// Go, so there may be gaps in type information around C.f references.
//
// This checker was initially written in vet to inspect raw cgo source
// files using partial type information. However, Analyzers in the new
// analysis API are presented with the type-checked, "cooked" Go ASTs
// resulting from cgo-processing files, so we must choose between
// working with the cooked file generated by cgo (which was tried but
// proved fragile) or locating the raw cgo file (e.g. from //line
// directives) and working with that, as we now do.
//
// Specifically, we must type-check the raw cgo source files (or at
// least the subtrees needed for this analyzer) in an environment that
// simulates the rest of the already type-checked package.
//
// For example, for each raw cgo source file in the original package,
// such as this one:
//
//	package p
//	import "C"
//	import "fmt"
//	type T int
//	const k = 3
//	var x, y = fmt.Println()
//	func f() { ... }
//	func g() { ... C.malloc(k) ... }
//	func (T) f(int) string { ... }
//
// we synthesize a new ast.File, shown below, that dot-imports the
// original "cooked" package using a special name ("·this·"), so that all
// references to package members resolve correctly. (References to
// unexported names cause an "unexported" error, which we ignore.)
//
// To avoid shadowing names imported from the cooked package,
// package-level declarations in the new source file are modified so
// that they do not declare any names.
// (The cgocall analysis is concerned with uses, not declarations.)
// Specifically, type declarations are discarded;
// all names in each var and const declaration are blanked out;
// each method is turned into a regular function by turning
// the receiver into the first parameter;
// and all functions are renamed to "_".
//
//	package p
//	import . "·this·" // declares T, k, x, y, f, g, T.f
//	import "C"
//	import "fmt"
//	const _ = 3
//	var _, _ = fmt.Println()
//	func _() { ... }
//	func _() { ... C.malloc(k) ... }
//	func _(T, int) string { ... }
//
// In this way, the raw function bodies and const/var initializer
// expressions are preserved but refer to the "cooked" objects imported
// from "·this·", and none of the transformed package-level declarations
// actually declares anything. In the example above, the reference to k
// in the argument of the call to C.malloc resolves to "·this·".k, which
// has an accurate type.
//
// This approach could in principle be generalized to more complex
// analyses on raw cgo files. One could synthesize a "C" package so that
// C.f would resolve to "·this·"._C_func_f, for example. But we have
// limited ourselves here to preserving function bodies and initializer
// expressions since that is all that the cgocall analyzer needs.
func typeCheckCgoSourceFiles(fset *token.FileSet, pkg *types.Package, files []*ast.File, info *types.Info, sizes types.Sizes) ([]*ast.File, *types.Info, error) {
	const thispkg = "·this·"

	// Which files are cgo files?
	var cgoFiles []*ast.File
	importMap := map[string]*types.Package{thispkg: pkg}
	for _, raw := range files {
		// If f is a cgo-generated file, Position reports
		// the original file, honoring //line directives.
		filename := fset.Position(raw.Pos()).Filename
		f, err := parser.ParseFile(fset, filename, nil, parser.Mode(0))
		if err != nil {
			return nil, nil, fmt.Errorf("can't parse raw cgo file: %v", err)
		}
		found := false
		for _, spec := range f.Imports {
			if spec.Path.Value == `"C"` {
				found = true
				break
			}
		}
		if !found {
			continue // not a cgo file
		}

		// Record the original import map.
		for _, spec := range raw.Imports {
			path, _ := strconv.Unquote(spec.Path.Value)
			importMap[path] = imported(info, spec)
		}

		// Add special dot-import declaration:
		//    import . "·this·"
		var decls []ast.Decl
		decls = append(decls, &ast.GenDecl{
			Tok: token.IMPORT,
			Specs: []ast.Spec{
				&ast.ImportSpec{
					Name: &ast.Ident{Name: "."},
					Path: &ast.BasicLit{
						Kind:  token.STRING,
						Value: strconv.Quote(thispkg),
					},
				},
			},
		})

		// Transform declarations from the raw cgo file.
		for _, decl := range f.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				switch decl.Tok {
				case token.TYPE:
					// Discard type declarations.
					continue
				case token.IMPORT:
					// Keep imports.
				case token.VAR, token.CONST:
					// Blank the declared var/const names.
					for _, spec := range decl.Specs {
						spec := spec.(*ast.ValueSpec)
						for i := range spec.Names {
							spec.Names[i].Name = "_"
						}
					}
				}
			case *ast.FuncDecl:
				// Blank the declared func name.
				decl.Name.Name = "_"

				// Turn a method receiver:  func (T) f(P) R {...}
				// into regular parameter:  func _(T, P) R {...}
				if decl.Recv != nil {
					var params []*ast.Field
					params = append(params, decl.Recv.List...)
					params = append(params, decl.Type.Params.List...)
					decl.Type.Params.List = params
					decl.Recv = nil
				}
			}
			decls = append(decls, decl)
		}
		f.Decls = decls
		if debug {
			format.Node(os.Stderr, fset, f) // debugging
		}
		cgoFiles = append(cgoFiles, f)
	}
	if cgoFiles == nil {
		return nil, nil, nil // nothing to do (can't happen?)
	}

	// Type-check the synthetic files.
	tc := &types.Config{
		FakeImportC: true,
		Importer: importerFunc(func(path string) (*types.Package, error) {
			return importMap[path], nil
		}),
		Sizes: sizes,
		Error: func(error) {}, // ignore errors (e.g. unused import)
	}

	// It's tempting to record the new types in the
	// existing pass.TypesInfo, but we don't own it.
	altInfo := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
	}
	tc.Check(pkg.Path(), fset, cgoFiles, altInfo)

	return cgoFiles, altInfo, nil
}

// cgoBaseType tries to look through type conversions involving
// unsafe.Pointer to find the real type. It converts:
//
//	unsafe.Pointer(x) => x
//	*(*unsafe.Pointer)(unsafe.Pointer(&x)) => x
func cgoBaseType(info *types.Info, arg ast.Expr) types.Type {
	switch arg := arg.(type) {
	case *ast.CallExpr:
		if len(arg.Args) == 1 && isUnsafePointer(info, arg.Fun) {
			return cgoBaseType(info, arg.Args[0])
		}
	case *ast.StarExpr:
		call, ok := arg.X.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			break
		}
		// Here arg is *f(v).
		t := info.Types[call.Fun].Type
		if t == nil {
			break
		}
		ptr, ok := t.Underlying().(*types.Pointer)
		if !ok {
			break
		}
		// Here arg is *(*p)(v)
		elem, ok := ptr.Elem().Underlying().(*types.Basic)
		if !ok || elem.Kind() != types.UnsafePointer {
			break
		}
		// Here arg is *(*unsafe.Pointer)(v)
		call, ok = call.Args[0].(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			break
		}
		// Here arg is *(*unsafe.Pointer)(f(v))
		if !isUnsafePointer(info, call.Fun) {
			break
		}
		// Here arg is *(*unsafe.Pointer)(unsafe.Pointer(v))
		u, ok := call.Args[0].(*ast.UnaryExpr)
		if !ok || u.Op != token.AND {
			break
		}
		// Here arg is *(*unsafe.Pointer)(unsafe.Pointer(&v))
		return cgoBaseType(info, u.X)
	}

	return info.Types[arg].Type
}

// typeOKForCgoCall reports whether the type of arg is OK to pass to a
// C function using cgo. This is not true for Go types with embedded
// pointers. m is used to avoid infinite recursion on recursive types.
func typeOKForCgoCall(t types.Type, m map[types.Type]bool) bool {
	if t == nil || m[t] {
		return true
	}
	m[t] = true
	switch t := t.Underlying().(type) {
	case *types.Chan, *types.Map, *types.Signature, *types.Slice:
		return false
	case *types.Pointer:
		return typeOKForCgoCall(t.Elem(), m)
	case *types.Array:
		return typeOKForCgoCall(t.Elem(), m)
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			if !typeOKForCgoCall(t.Field(i).Type(), m) {
				return false
			}
		}
	}
	return true
}

func isUnsafePointer(info *types.Info, e ast.Expr) bool {
	t := info.Types[e].Type
	return t != nil && t.Underlying() == types.Typ[types.UnsafePointer]
}

type importerFunc func(path string) (*types.Package, error)

func (f importerFunc) Import(path string) (*types.Package, error) { return f(path) }

// TODO(adonovan): make this a library function or method of Info.
func imported(info *types.Info, spec *ast.ImportSpec) *types.Package {
	obj, ok := info.Implicits[spec]
	if !ok {
		obj = info.Defs[spec.Name] // renaming import
	}
	return obj.(*types.PkgName).Imported()
}
