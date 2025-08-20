package scout

import (
	"context"
	"strings"

	"github.com/docker/docker-language-server/internal/cache"
	"github.com/docker/docker-language-server/internal/configuration"
	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/pkg/lsp/textdocument"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
)

type Service interface {
	textdocument.DiagnosticsCollector
	Analyze(documentURI protocol.DocumentUri, image string) ([]Diagnostic, error)
	Hover(ctx context.Context, documentURI protocol.DocumentUri, image string) (*protocol.Hover, error)
}

type ServiceImpl struct {
	manager cache.CacheManager[ImageResponse]
}

func NewService() Service {
	client := NewLanguageGatewayClient()
	return &ServiceImpl{
		manager: cache.NewManager(client),
	}
}

func (s *ServiceImpl) Hover(ctx context.Context, documentURI protocol.DocumentUri, image string) (*protocol.Hover, error) {
	config := configuration.Get(documentURI)
	if !config.Experimental.VulnerabilityScanning {
		return nil, nil
	}

	resp, err := s.manager.Get(&ScoutImageKey{Image: image})
	if err == nil {
		hovers := []string{}
		for _, info := range resp.Infos {
			if !config.Experimental.Scout.CriticalHighVulnerabilities && info.Kind == "critical_high_vulnerabilities" {
				continue
			}
			if !config.Experimental.Scout.RecommendedTag && info.Kind == "recommended_tag" {
				continue
			}
			if !config.Experimental.Scout.Vulnerabilites && info.Kind == "vulnerabilities" {
				continue
			}
			hovers = append(hovers, info.Description.Markdown)
		}

		if len(hovers) > 0 {
			return &protocol.Hover{
				Contents: protocol.MarkupContent{
					Value: strings.Join(hovers, "\r\n\r\n"),
					Kind:  protocol.MarkupKindMarkdown,
				},
			}, nil
		}
		return nil, nil
	}
	return nil, err
}

func (s *ServiceImpl) Analyze(documentURI protocol.DocumentUri, image string) ([]Diagnostic, error) {
	config := configuration.Get(documentURI)
	if !config.Experimental.VulnerabilityScanning {
		return nil, nil
	}

	resp, err := s.manager.Get(&ScoutImageKey{Image: image})
	if err != nil {
		return nil, err
	}

	diagnostics := make([]Diagnostic, len(resp.Diagnostics))
	for _, diagnostic := range resp.Diagnostics {
		if !config.Experimental.Scout.CriticalHighVulnerabilities && diagnostic.Kind == "critical_high_vulnerabilities" {
			continue
		}
		if !config.Experimental.Scout.NotPinnedDigest && diagnostic.Kind == "not_pinned_digest" {
			continue
		}
		if !config.Experimental.Scout.RecommendedTag && diagnostic.Kind == "recommended_tag" {
			continue
		}
		if !config.Experimental.Scout.Vulnerabilites && diagnostic.Kind == "vulnerabilities" {
			continue
		}
		diagnostics = append(diagnostics, diagnostic)
	}
	return diagnostics, nil
}

func (s *ServiceImpl) CalculateDiagnostics(ctx context.Context, source string, doc document.Document) ([]protocol.Diagnostic, error) {
	config := configuration.Get(protocol.DocumentUri(doc.URI()))
	if !config.Experimental.VulnerabilityScanning {
		return nil, nil
	}

	lspDiagnostics := []protocol.Diagnostic{}
	lines := strings.Split(string(doc.Input()), "\n")
	for _, child := range doc.(document.DockerfileDocument).Nodes() {
		if strings.EqualFold(child.Value, "FROM") && child.Next != nil {
			resp, err := s.manager.Get(&ScoutImageKey{Image: child.Next.Value})
			if err == nil {
				next := child.Next
				prefix := []string{child.Value}
				prefix = append(prefix, child.Flags...)
				suffix := []string{}
				next = next.Next
				for next != nil {
					suffix = append(suffix, next.Value)
					next = next.Next
				}

				for _, diagnostic := range resp.Diagnostics {
					if !config.Experimental.Scout.CriticalHighVulnerabilities && diagnostic.Kind == "critical_high_vulnerabilities" {
						continue
					}
					if !config.Experimental.Scout.NotPinnedDigest && diagnostic.Kind == "not_pinned_digest" {
						continue
					}
					if !config.Experimental.Scout.RecommendedTag && diagnostic.Kind == "recommended_tag" {
						continue
					}
					if !config.Experimental.Scout.Vulnerabilites && diagnostic.Kind == "vulnerabilities" {
						continue
					}

					namedEdits := []types.NamedEdit{}
					for _, edit := range resp.Edits {
						if diagnostic.Kind == edit.Diagnostic {
							content := []string{}
							content = append(content, prefix...)
							content = append(content, edit.Edit)
							content = append(content, suffix...)
							namedEdits = append(namedEdits, types.NamedEdit{
								Title: edit.Title,
								Edit:  strings.Join(content, " "),
							})
						}
					}

					lspDiagnostic := ConvertDiagnostic(diagnostic, source, protocol.Range{
						Start: protocol.Position{
							Line:      uint32(child.StartLine - 1),
							Character: 0,
						},
						End: protocol.Position{
							Line:      uint32(child.EndLine - 1),
							Character: uint32(len(lines[child.StartLine-1])),
						},
					}, namedEdits)
					lspDiagnostics = append(lspDiagnostics, lspDiagnostic)
				}
				continue
			}
		}
	}
	return lspDiagnostics, nil
}

func (s *ServiceImpl) CollectDiagnostics(source, workspaceFolder string, doc document.Document, text string) []protocol.Diagnostic {
	diagnostics, _ := s.CalculateDiagnostics(context.Background(), source, doc)
	return diagnostics
}

func (c *ServiceImpl) SupportsLanguageIdentifier(languageIdentifier protocol.LanguageIdentifier) bool {
	return languageIdentifier == protocol.DockerfileLanguage
}
