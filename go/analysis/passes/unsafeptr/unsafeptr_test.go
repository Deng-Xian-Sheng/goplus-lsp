// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unsafeptr_test

import (
	"testing"

	"github.com/Deng-Xian-Sheng/goplus-lsp/go/analysis/analysistest"
	"github.com/Deng-Xian-Sheng/goplus-lsp/go/analysis/passes/unsafeptr"
	"github.com/Deng-Xian-Sheng/goplus-lsp/internal/typeparams"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	pkgs := []string{"a"}
	if typeparams.Enabled {
		pkgs = append(pkgs, "typeparams")
	}
	analysistest.Run(t, testdata, unsafeptr.Analyzer, pkgs...)
}
