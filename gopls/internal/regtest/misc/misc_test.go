// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package misc

import (
	"testing"

	"github.com/Deng-Xian-Sheng/goplus-lsp/gopls/internal/hooks"
	"github.com/Deng-Xian-Sheng/goplus-lsp/gopls/internal/lsp/regtest"
	"github.com/Deng-Xian-Sheng/goplus-lsp/internal/bug"
)

func TestMain(m *testing.M) {
	bug.PanicOnBugs = true
	regtest.Main(m, hooks.Options)
}
