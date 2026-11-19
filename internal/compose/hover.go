package compose

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/token"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func Hover(ctx context.Context, params *protocol.HoverParams, doc document.ComposeDocument) (*protocol.Hover, error) {
	file := doc.File()
	if file == nil || len(file.Docs) == 0 {
		return nil, nil
	}

	line := int(params.Position.Line) + 1
	character := int(params.Position.Character) + 1
	lines := strings.Split(string(doc.Input()), "\n")
	if line > len(lines) {
		return nil, nil
	}

	for _, documentNode := range file.Docs {
		if mappingNode, ok := documentNode.Body.(*ast.MappingNode); ok {
			nodePath := constructNodePath([]ast.Node{}, mappingNode, int(params.Position.Line+1), int(params.Position.Character+1))
			result := serviceHover(doc, mappingNode, nodePath)
			if result != nil {
				return result, nil
			}
			result = networkHover(doc, mappingNode, nodePath)
			if result != nil {
				return result, nil
			}
			result = configHover(doc, mappingNode, nodePath)
			if result != nil {
				return result, nil
			}
			result = secretHover(doc, mappingNode, nodePath)
			if result != nil {
				return result, nil
			}
			result = volumeHover(doc, params, mappingNode, nodePath)
			if result != nil {
				return result, nil
			}
			result = modelHover(doc, mappingNode, nodePath)
			if result != nil {
				return result, nil
			}
			result = hover(composeSchema, nodePath, line, character, len(lines[params.Position.Line])+1)
			if result != nil {
				return result, nil
			}

			anchor, aliases := fragmentReference(mappingNode, line, character)
			if anchor != nil {
				t := anchor.Name.GetToken()
				if t.Position.Line == line && t.Position.Column <= character && character <= t.Position.Column+len(t.Value) {
					return createYamlHover(anchor.Value, t), nil
				}
				for i := range aliases {
					t := aliases[i].Value.GetToken()
					if t.Position.Line == line && t.Position.Column <= character && character <= t.Position.Column+len(t.Value) {
						return createYamlHover(anchor.Value, t), nil
					}
				}
			}
		}
	}
	return nil, nil
}

func createYamlHover(node ast.Node, hovered *token.Token) *protocol.Hover {
	split := strings.Split(node.String(), "\n")
	// remove leading empty line inserted by goccy/go-yaml if present
	if strings.TrimSpace(split[0]) == "" {
		split = split[1:]
	}
	skip := -1
	for i := range len(split) {
		if skip == -1 {
			for j := range len(split[i]) {
				if !unicode.IsSpace(rune(split[i][j])) {
					skip = j
					break
				}
			}
		}
		// extra check in case there are empty lines
		if len(split[i]) > skip {
			split[i] = split[i][skip:]
		}
	}
	r := createRange(hovered, len(hovered.Value))
	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: fmt.Sprintf("```YAML\n%v\n```", strings.Join(split, "\n")),
		},
		Range: &r,
	}
}

func serviceHover(doc document.ComposeDocument, mappingNode *ast.MappingNode, nodePath []ast.Node) *protocol.Hover {
	if (len(nodePath) == 4 || len(nodePath) == 5) && nodePath[0].GetToken().Value == "services" {
		t := nodePath[3].GetToken()
		if nodePath[2].GetToken().Value == "extends" {
			if len(nodePath) == 5 && t.Value == "service" {
				if _, ok := nodePath[4].(*ast.StringNode); ok {
					t = nodePath[4].GetToken()
					if t.Next != nil && t.Next.Type == token.MappingValueType {
						return nil
					}
				} else {
					return nil
				}
			} else if t.Next != nil && t.Next.Type == token.MappingValueType {
				return nil
			}
			result := createDependencyHover(doc, mappingNode, t, "services", t.Value)
			if result != nil {
				return result
			}
		}

		if nodePath[2].GetToken().Value == "depends_on" {
			if t.Next != nil && t.Next.Type == token.MappingValueType && t.Prev.Type == token.SequenceEntryType {
				return nil
			}
			result := createDependencyHover(doc, mappingNode, t, "services", t.Value)
			if result != nil {
				return result
			}
		}
	}
	return nil
}

func networkHover(doc document.ComposeDocument, mappingNode *ast.MappingNode, nodePath []ast.Node) *protocol.Hover {
	if len(nodePath) == 4 && nodePath[0].GetToken().Value == "services" {
		if nodePath[2].GetToken().Value == "networks" {
			t := nodePath[3].GetToken()
			return createDependencyHover(doc, mappingNode, t, "networks", t.Value)
		}
	}
	return nil
}

func configHover(doc document.ComposeDocument, mappingNode *ast.MappingNode, nodePath []ast.Node) *protocol.Hover {
	if len(nodePath) == 4 && nodePath[0].GetToken().Value == "services" {
		// array string
		if nodePath[2].GetToken().Value == "configs" {
			t := nodePath[3].GetToken()
			return createDependencyHover(doc, mappingNode, t, "configs", t.Value)
		}
	} else if len(nodePath) == 5 && nodePath[0].GetToken().Value == "services" {
		// array object
		if nodePath[2].GetToken().Value == "configs" && nodePath[3].GetToken().Value == "source" {
			t := nodePath[4].GetToken()
			return createDependencyHover(doc, mappingNode, t, "configs", t.Value)
		}
	}
	return nil
}

func secretHover(doc document.ComposeDocument, mappingNode *ast.MappingNode, nodePath []ast.Node) *protocol.Hover {
	if len(nodePath) >= 4 && nodePath[0].GetToken().Value == "services" {
		if len(nodePath) == 4 {
			// array string
			if nodePath[2].GetToken().Value == "secrets" {
				t := nodePath[3].GetToken()
				return createDependencyHover(doc, mappingNode, t, "secrets", t.Value)
			}
		} else if len(nodePath) == 5 {
			// array object
			if nodePath[2].GetToken().Value == "secrets" && nodePath[3].GetToken().Value == "source" {
				t := nodePath[4].GetToken()
				return createDependencyHover(doc, mappingNode, t, "secrets", t.Value)
			} else if nodePath[2].GetToken().Value == "build" && nodePath[3].GetToken().Value == "secrets" {
				// array string in the build object
				t := nodePath[4].GetToken()
				return createDependencyHover(doc, mappingNode, t, "secrets", t.Value)
			}
		} else if len(nodePath) == 6 {
			// array object in the build object
			if nodePath[2].GetToken().Value == "build" && nodePath[3].GetToken().Value == "secrets" && nodePath[4].GetToken().Value == "source" {
				t := nodePath[5].GetToken()
				return createDependencyHover(doc, mappingNode, t, "secrets", t.Value)
			}
		}
	}
	return nil
}

func volumeHover(doc document.ComposeDocument, params *protocol.HoverParams, mappingNode *ast.MappingNode, nodePath []ast.Node) *protocol.Hover {
	if len(nodePath) >= 4 && nodePath[0].GetToken().Value == "services" {
		if len(nodePath) == 4 {
			// array string
			if nodePath[2].GetToken().Value == "volumes" {
				volumeName := nodePath[3].GetToken().Value
				idx := strings.Index(volumeName, ":")
				t := nodePath[3].GetToken()
				if idx >= 0 {
					ch := int(params.Position.Character)
					col := t.Position.Column - 1
					if col > ch || ch >= col+idx {
						return nil
					}
					volumeName = volumeName[0:idx]
					t = &token.Token{
						Type:     t.Type,
						Value:    t.Value[0:idx],
						Position: t.Position,
					}
				}
				return createDependencyHover(doc, mappingNode, t, "volumes", volumeName)
			}
		} else if len(nodePath) == 5 {
			// array object
			if nodePath[2].GetToken().Value == "volumes" && nodePath[3].GetToken().Value == "source" {
				t := nodePath[4].GetToken()
				return createDependencyHover(doc, mappingNode, t, "volumes", t.Value)
			}
		}
	}
	return nil
}

func modelHover(doc document.ComposeDocument, mappingNode *ast.MappingNode, nodePath []ast.Node) *protocol.Hover {
	if len(nodePath) == 4 && nodePath[0].GetToken().Value == "services" {
		if nodePath[2].GetToken().Value == "models" {
			t := nodePath[3].GetToken()
			return createDependencyHover(doc, mappingNode, t, "models", t.Value)
		}
	}
	return nil
}

func createDependencyHover(doc document.ComposeDocument, mappingNode *ast.MappingNode, hovered *token.Token, dependencyType, dependencyName string) *protocol.Hover {
	for _, node := range mappingNode.Values {
		if s, ok := node.Key.(*ast.StringNode); ok && s.Value == dependencyType {
			if mappingNode, ok := node.Value.(*ast.MappingNode); ok {
				for _, service := range mappingNode.Values {
					if service.Key.GetToken().Value == dependencyName {
						return createYamlHover(service, hovered)
					}
				}
			}
		}
	}

	node, _ := dependencyLookup(doc, dependencyType, dependencyName)
	if node != nil {
		return createYamlHover(node, hovered)
	}
	return nil
}

func hover(schema *jsonschema.Schema, nodes []ast.Node, line, column, lineLength int) *protocol.Hover {
	for _, match := range nodes {
		if schema.Ref != nil {
			schema = schema.Ref
		}

		if nested, ok := schema.Items.(*jsonschema.Schema); ok {
			if nested.Ref != nil {
				nested = nested.Ref
			}
			for _, n := range nested.OneOf {
				if n.Types != nil && slices.Contains(n.Types.ToStrings(), "object") {
					if len(n.Properties) > 0 {
						if _, ok := n.Properties[match.GetToken().Value]; ok {
							schema = n
							break
						}
					}
				}
			}

			if _, ok := nested.Properties[match.GetToken().Value]; ok {
				schema = nested
			}
		}

		for _, nested := range schema.OneOf {
			if nested.Types != nil && slices.Contains(nested.Types.ToStrings(), "object") {
				schema = nested
				break
			}
		}

		if property, ok := schema.Properties[match.GetToken().Value]; ok {
			if property.Ref != nil {
				property = property.Ref
			}
			if property.Enum != nil {
				if match.GetToken().Position.Column <= column && column <= lineLength {
					var builder strings.Builder
					if property.Description != "" {
						builder.WriteString(property.Description)
						builder.WriteString("\n\n")
					}
					builder.WriteString("Allowed values:\n")
					enumValues := []string{}
					for _, value := range property.Enum.Values {
						enumValues = append(enumValues, fmt.Sprintf("%v", value))
					}
					slices.Sort(enumValues)
					for _, value := range enumValues {
						fmt.Fprintf(&builder, "- `%v`\n", value)
					}
					builder.WriteString("\nSchema: [compose-spec.json](https://raw.githubusercontent.com/compose-spec/compose-spec/master/schema/compose-spec.json)")
					fmt.Fprintf(&builder,
						"\n\n[Online documentation](https://docs.docker.com/reference/compose-file/%v/#%v)",
						nodes[0].GetToken().Value,
						nodes[2].GetToken().Value,
					)
					return &protocol.Hover{
						Contents: protocol.MarkupContent{
							Kind:  protocol.MarkupKindMarkdown,
							Value: builder.String(),
						},
					}
				}
			}

			if match.GetToken().Position.Line == line && match.GetToken().Position.Column+len(match.GetToken().Value) >= column && property.Description != "" {
				var builder strings.Builder
				builder.WriteString(property.Description)
				builder.WriteString("\n\nSchema: [compose-spec.json](https://raw.githubusercontent.com/compose-spec/compose-spec/master/schema/compose-spec.json)")
				switch nodes[0].GetToken().Value {
				case "name":
					builder.WriteString("\n\n[Online documentation](https://docs.docker.com/reference/compose-file/version-and-name/)")
				case "version":
					builder.WriteString("\n\n[Online documentation](https://docs.docker.com/reference/compose-file/version-and-name/)")
				case "include":
					if len(nodes) == 1 {
						fmt.Fprintf(&builder,
							"\n\n[Online documentation](https://docs.docker.com/reference/compose-file/%v/)",
							nodes[0].GetToken().Value,
						)
					} else {
						fmt.Fprintf(&builder,
							"\n\n[Online documentation](https://docs.docker.com/reference/compose-file/%v/#%v)",
							nodes[0].GetToken().Value,
							nodes[1].GetToken().Value,
						)
					}
				default:
					if len(nodes) == 1 {
						fmt.Fprintf(&builder,
							"\n\n[Online documentation](https://docs.docker.com/reference/compose-file/%v/)",
							nodes[0].GetToken().Value,
						)
					} else {
						fmt.Fprintf(&builder,
							"\n\n[Online documentation](https://docs.docker.com/reference/compose-file/%v/#%v)",
							nodes[0].GetToken().Value,
							nodes[2].GetToken().Value,
						)
					}
				}
				return &protocol.Hover{
					Contents: protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: builder.String(),
					},
				}
			}
			schema = property
			continue
		}

		matched := false
		for regexp, property := range schema.PatternProperties {
			if regexp.MatchString(match.GetToken().Value) {
				if property.Ref == nil {
					schema = property
				} else {
					schema = property.Ref
				}
				matched = true
				break
			}
		}

		if !matched {
			// the node does not appear to be described by the schema
			return nil
		}
	}
	return nil
}

func constructNodePath(matches []ast.Node, node ast.Node, line, col int) []ast.Node {
	node = resolveAnchor(node)
	switch n := node.(type) {
	case *ast.MappingValueNode:
		nodeKey := resolveAnchor(n.Key)
		if m := constructNodePath(matches, nodeKey, line, col); m != nil {
			matches = append(matches, m...)
			return matches
		}
		if m := constructNodePath(matches, n.Value, line, col); m != nil {
			matches = append(matches, nodeKey)
			matches = append(matches, m...)
			return matches
		}
	case *ast.MappingNode:
		for _, kv := range n.Values {
			if m := constructNodePath(matches, kv, line, col); m != nil {
				matches = append(matches, m...)
				return matches
			}
		}
	case *ast.SequenceNode:
		for _, item := range n.Values {
			if m := constructNodePath(matches, item, line, col); m != nil {
				matches = append(matches, m...)
				return matches
			}
		}
	}

	token := node.GetToken()
	if token.Position.Line == line && token.Position.Column <= col && col <= token.Position.Column+len(token.Value) {
		return []ast.Node{node}
	}
	return nil
}

func resolveAnchor(node ast.Node) ast.Node {
	if anchor, ok := node.(*ast.AnchorNode); ok {
		return anchor.Value
	}
	return node
}
