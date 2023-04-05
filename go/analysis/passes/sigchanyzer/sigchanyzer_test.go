// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sigchanyzer_test

import (
	"testing"

	"github.com/Deng-Xian-Sheng/goplus-lsp/go/analysis/analysistest"
	"github.com/Deng-Xian-Sheng/goplus-lsp/go/analysis/passes/sigchanyzer"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, sigchanyzer.Analyzer, "a")
}
