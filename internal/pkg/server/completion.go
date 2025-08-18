package server

import (
	"github.com/docker/docker-language-server/internal/bake/hcl"
	"github.com/docker/docker-language-server/internal/compose"
	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"go.lsp.dev/uri"
)

func (s *Server) TextDocumentCompletion(ctx *glsp.Context, params *protocol.CompletionParams) (any, error) {
	doc, err := s.docs.Read(ctx.Context, uri.URI(params.TextDocument.URI))
	if err != nil {
		return nil, err
	}
	defer doc.Close()

	if doc.LanguageIdentifier() == protocol.DockerBakeLanguage {
		return hcl.Completion(ctx.Context, params, s.docs, doc.(document.BakeHCLDocument))
	} else if doc.LanguageIdentifier() == protocol.DockerComposeLanguage && s.composeSupport && s.composeCompletion {
		return compose.Completion(ctx.Context, params, s.docs, s.hubService, doc.(document.ComposeDocument))
	}
	return nil, nil
}
