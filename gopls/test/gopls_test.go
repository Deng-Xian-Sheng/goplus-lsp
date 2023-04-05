// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gopls_test

import (
	"os"
	"testing"

	"github.com/Deng-Xian-Sheng/goplus-lsp/gopls/internal/hooks"
	cmdtest "github.com/Deng-Xian-Sheng/goplus-lsp/gopls/internal/lsp/cmd/test"
	"github.com/Deng-Xian-Sheng/goplus-lsp/gopls/internal/lsp/source"
	"github.com/Deng-Xian-Sheng/goplus-lsp/gopls/internal/lsp/tests"
	"github.com/Deng-Xian-Sheng/goplus-lsp/internal/bug"
	"github.com/Deng-Xian-Sheng/goplus-lsp/internal/event"
	"github.com/Deng-Xian-Sheng/goplus-lsp/internal/testenv"
)

func TestMain(m *testing.M) {
	bug.PanicOnBugs = true
	testenv.ExitIfSmallMachine()

	// Set the global exporter to nil so that we don't log to stderr. This avoids
	// a lot of misleading noise in test output.
	//
	// See also ../internal/lsp/lsp_test.go.
	event.SetExporter(nil)

	os.Exit(m.Run())
}

func TestCommandLine(t *testing.T) {
	cmdtest.TestCommandLine(t, "../internal/lsp/testdata", commandLineOptions)
}

func commandLineOptions(options *source.Options) {
	options.Staticcheck = true
	options.GoDiff = false
	tests.DefaultOptions(options)
	hooks.Options(options)
}
