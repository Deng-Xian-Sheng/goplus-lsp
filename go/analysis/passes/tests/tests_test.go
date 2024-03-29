// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tests_test

import (
	"testing"

	"github.com/Deng-Xian-Sheng/goplus-lsp/internal/analysisinternal"

	"github.com/Deng-Xian-Sheng/goplus-lsp/go/analysis/analysistest"
	"github.com/Deng-Xian-Sheng/goplus-lsp/go/analysis/passes/tests"
	"github.com/Deng-Xian-Sheng/goplus-lsp/internal/typeparams"
)

func Test(t *testing.T) {
	// In 1.18, diagnostic for Fuzz Tests must not be used by cmd/vet.
	// So the code for Fuzz tests diagnostics is guarded behind flag analysisinternal.DiagnoseFuzzTests
	// Turn on the flag DiagnoseFuzzTests for analysis tests and then turn it off.
	analysisinternal.DiagnoseFuzzTests = true
	defer func() {
		analysisinternal.DiagnoseFuzzTests = false
	}()
	testdata := analysistest.TestData()
	pkgs := []string{
		"a",        // loads "a", "a [a.test]", and "a.test"
		"b_x_test", // loads "b" and "b_x_test"
		"divergent",
	}
	if typeparams.Enabled {
		pkgs = append(pkgs, "typeparams")
	}
	analysistest.Run(t, testdata, tests.Analyzer, pkgs...)
}
