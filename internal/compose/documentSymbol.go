package compose

import (
	"context"

	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/token"
)

var symbolKinds = map[string]protocol.SymbolKind{
	"services": protocol.SymbolKindClass,
	"networks": protocol.SymbolKindInterface,
	"volumes":  protocol.SymbolKindFile,
	"configs":  protocol.SymbolKindVariable,
	"secrets":  protocol.SymbolKindKey,
	"models":   protocol.SymbolKindModule,
}

func findSymbols(value string, n *ast.MappingValueNode, mapping map[string]protocol.SymbolKind) (result []any) {
	if kind, ok := mapping[value]; ok {
		if mappingNode, ok := n.Value.(*ast.MappingNode); ok {
			for _, service := range mappingNode.Values {
				result = append(result, createSymbol(service.Key.GetToken(), kind))
			}
		} else if n, ok := n.Value.(*ast.MappingValueNode); ok {
			result = append(result, createSymbol(n.Key.GetToken(), kind))
		}
	} else if value == "include" {
		if sequenceNode, ok := n.Value.(*ast.SequenceNode); ok {
			for _, token := range includedFiles(sequenceNode.Values) {
				result = append(result, createSymbol(token, protocol.SymbolKindModule))
			}
		}
	}
	return result
}

func includedFiles(nodes []ast.Node) []*token.Token {
	tokens := []*token.Token{}
	for _, entry := range nodes {
		if mappingNode, ok := resolveAnchor(entry).(*ast.MappingNode); ok {
			for _, value := range mappingNode.Values {
				if resolveAnchor(value.Key).GetToken().Value == "path" {
					if paths, ok := resolveAnchor(value.Value).(*ast.SequenceNode); ok {
						// include:
						//   - path:
						//     - ../commons/compose.yaml
						//     - ./commons-override.yaml
						for _, path := range paths.Values {
							tokens = append(tokens, resolveAnchor(path).GetToken())
						}
					} else {
						// include:
						// - path: ../commons/compose.yaml
						//   project_directory: ..
						//   env_file: ../another/.env
						tokens = append(tokens, resolveAnchor(value.Value).GetToken())
					}
				}
			}
		} else {
			// include:
			//   - abc.yml
			//   - def.yml
			stringNode := stringNode(entry)
			if stringNode != nil {
				tokens = append(tokens, stringNode.GetToken())
			}

		}
	}
	return tokens
}

func DocumentSymbol(ctx context.Context, doc document.ComposeDocument) (result []any, err error) {
	file := doc.File()
	if file == nil || len(file.Docs) == 0 {
		return nil, nil
	}

	for _, documentNode := range file.Docs {
		if mappingNode, ok := documentNode.Body.(*ast.MappingNode); ok {
			for _, n := range mappingNode.Values {
				if s, ok := n.Key.(*ast.StringNode); ok {
					result = append(result, findSymbols(s.Value, n, symbolKinds)...)
				}
			}
		}
	}
	return result, nil
}

func createSymbol(t *token.Token, kind protocol.SymbolKind) *protocol.DocumentSymbol {
	rng := protocol.Range{
		Start: protocol.Position{
			Line:      uint32(t.Position.Line - 1),
			Character: uint32(t.Position.Column - 1),
		},
		End: protocol.Position{
			Line:      uint32(t.Position.Line - 1),
			Character: uint32(t.Position.Column - 1 + len(t.Value)),
		},
	}
	return &protocol.DocumentSymbol{
		Name:           t.Value,
		Kind:           kind,
		Range:          rng,
		SelectionRange: rng,
	}
}
