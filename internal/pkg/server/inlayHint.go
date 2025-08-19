package server

import (
	"github.com/docker/docker-language-server/internal/bake/hcl"
	"github.com/docker/docker-language-server/internal/compose"
	"github.com/docker/docker-language-server/internal/dockerfile"
	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"go.lsp.dev/uri"
)

func (s *Server) TextDocumentInlayHint(ctx *glsp.Context, params *protocol.InlayHintParams) (any, error) {
	doc, err := s.docs.Read(ctx.Context, uri.URI(params.TextDocument.URI))
	if err != nil {
		return nil, err
	}
	defer doc.Close()
	if doc.LanguageIdentifier() == protocol.DockerComposeLanguage && s.composeSupport {
		return compose.InlayHint(doc.(document.ComposeDocument), params.Range)
	} else if doc.LanguageIdentifier() == protocol.DockerBakeLanguage {
		return hcl.InlayHint(s.docs, doc.(document.BakeHCLDocument), params.Range)
	} else if doc.LanguageIdentifier() == protocol.DockerfileLanguage {
		return dockerfile.InlayHint(*s.hubService, doc.(document.DockerfileDocument), params.Range)
	}
	return nil, nil
}
