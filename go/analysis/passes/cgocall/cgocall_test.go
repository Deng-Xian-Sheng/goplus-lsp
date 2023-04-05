// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cgocall_test

import (
	"testing"

	"github.com/Deng-Xian-Sheng/goplus-lsp/go/analysis/analysistest"
	"github.com/Deng-Xian-Sheng/goplus-lsp/go/analysis/passes/cgocall"
	"github.com/Deng-Xian-Sheng/goplus-lsp/internal/typeparams"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	tests := []string{"a", "b", "c"}
	if typeparams.Enabled {
		// and testdata/src/typeparams/typeparams.go when possible
		tests = append(tests, "typeparams")
	}
	analysistest.Run(t, testdata, cgocall.Analyzer, tests...)
}
