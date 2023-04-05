// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nilfunc_test

import (
	"github.com/Deng-Xian-Sheng/goplus-lsp/go/analysis/analysistest"
	"github.com/Deng-Xian-Sheng/goplus-lsp/go/analysis/passes/nilfunc"
	"github.com/Deng-Xian-Sheng/goplus-lsp/internal/typeparams"
	"testing"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	tests := []string{"a"}
	if typeparams.Enabled {
		tests = append(tests, "typeparams")
	}
	analysistest.Run(t, testdata, nilfunc.Analyzer, tests...)
}
