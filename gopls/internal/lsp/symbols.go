// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"context"

	"github.com/Deng-Xian-Sheng/goplus-lsp/gopls/internal/lsp/protocol"
	"github.com/Deng-Xian-Sheng/goplus-lsp/gopls/internal/lsp/source"
	"github.com/Deng-Xian-Sheng/goplus-lsp/gopls/internal/lsp/template"
	"github.com/Deng-Xian-Sheng/goplus-lsp/internal/event"
	"github.com/Deng-Xian-Sheng/goplus-lsp/internal/event/tag"
)

func (s *Server) documentSymbol(ctx context.Context, params *protocol.DocumentSymbolParams) ([]interface{}, error) {
	ctx, done := event.Start(ctx, "lsp.Server.documentSymbol")
	defer done()

	snapshot, fh, ok, release, err := s.beginFileRequest(ctx, params.TextDocument.URI, source.UnknownKind)
	defer release()
	if !ok {
		return []interface{}{}, err
	}
	var docSymbols []protocol.DocumentSymbol
	if snapshot.View().FileKind(fh) == source.Tmpl {
		docSymbols, err = template.DocumentSymbols(snapshot, fh)
	} else {
		docSymbols, err = source.DocumentSymbols(ctx, snapshot, fh)
	}
	if err != nil {
		event.Error(ctx, "DocumentSymbols failed", err, tag.URI.Of(fh.URI()))
		return []interface{}{}, nil
	}
	// Convert the symbols to an interface array.
	// TODO: Remove this once the lsp deprecates SymbolInformation.
	symbols := make([]interface{}, len(docSymbols))
	for i, s := range docSymbols {
		if snapshot.View().Options().HierarchicalDocumentSymbolSupport {
			symbols[i] = s
			continue
		}
		// If the client does not support hierarchical document symbols, then
		// we need to be backwards compatible for now and return SymbolInformation.
		symbols[i] = protocol.SymbolInformation{
			Name:       s.Name,
			Kind:       s.Kind,
			Deprecated: s.Deprecated,
			Location: protocol.Location{
				URI:   params.TextDocument.URI,
				Range: s.Range,
			},
		}
	}
	return symbols, nil
}
