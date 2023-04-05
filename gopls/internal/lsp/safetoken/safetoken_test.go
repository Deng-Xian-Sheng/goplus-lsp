// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package safetoken_test

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/Deng-Xian-Sheng/goplus-lsp/go/packages"
	"github.com/Deng-Xian-Sheng/goplus-lsp/internal/testenv"
)

// This test reports any unexpected uses of (*go/token.File).Offset within
// the gopls codebase to ensure that we don't check in more code that is prone
// to panicking. All calls to (*go/token.File).Offset should be replaced with
// calls to safetoken.Offset.
func TestTokenOffset(t *testing.T) {
	testenv.NeedsGoPackages(t)

	fset := token.NewFileSet()
	pkgs, err := packages.Load(&packages.Config{
		Fset: fset,
		Mode: packages.NeedName | packages.NeedModule | packages.NeedCompiledGoFiles | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedImports | packages.NeedDeps,
	}, "go/token", "github.com/Deng-Xian-Sheng/goplus-lsp/gopls/internal/lsp/...", "github.com/Deng-Xian-Sheng/goplus-lsp/gopls/...")
	if err != nil {
		t.Fatal(err)
	}
	var tokenPkg, safePkg *packages.Package
	for _, pkg := range pkgs {
		switch pkg.PkgPath {
		case "go/token":
			tokenPkg = pkg
		case "github.com/Deng-Xian-Sheng/goplus-lsp/gopls/internal/lsp/safetoken":
			safePkg = pkg
		}
	}

	if tokenPkg == nil {
		t.Fatal("missing package go/token")
	}
	if safePkg == nil {
		t.Fatal("missing package github.com/Deng-Xian-Sheng/goplus-lsp/gopls/internal/lsp/safetoken")
	}

	fileObj := tokenPkg.Types.Scope().Lookup("File")
	tokenOffset, _, _ := types.LookupFieldOrMethod(fileObj.Type(), true, fileObj.Pkg(), "Offset")

	safeOffset := safePkg.Types.Scope().Lookup("Offset").(*types.Func)

	for _, pkg := range pkgs {
		if pkg.PkgPath == "go/token" { // Allow usage from within go/token itself.
			continue
		}
		for ident, obj := range pkg.TypesInfo.Uses {
			if obj != tokenOffset {
				continue
			}
			if safeOffset.Pos() <= ident.Pos() && ident.Pos() <= safeOffset.Scope().End() {
				continue // accepted usage
			}
			t.Errorf(`%s: Unexpected use of (*go/token.File).Offset. Please use github.com/Deng-Xian-Sheng/goplus-lsp/gopls/internal/lsp/safetoken.Offset instead.`, fset.Position(ident.Pos()))
		}
	}
}
