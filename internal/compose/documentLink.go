package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/v2/format"
	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/token"
)

func createRange(t *token.Token, length int) protocol.Range {
	offset := 0
	if t.Type == token.DoubleQuoteType {
		offset = 1
	}
	return protocol.Range{
		Start: protocol.Position{
			Line:      protocol.UInteger(t.Position.Line - 1),
			Character: protocol.UInteger(t.Position.Column + offset - 1),
		},
		End: protocol.Position{
			Line:      protocol.UInteger(t.Position.Line - 1),
			Character: protocol.UInteger(t.Position.Column + offset + length - 1),
		},
	}
}

func createLink(folderAbsolutePath string, wslDollarSign bool, node *token.Token) *protocol.DocumentLink {
	file := node.Value
	u, path := types.Concatenate(folderAbsolutePath, file, wslDollarSign)
	return &protocol.DocumentLink{
		Range:   createRange(node, len(file)),
		Target:  types.CreateStringPointer(u),
		Tooltip: types.CreateStringPointer(path),
	}
}

func createFileLink(folderAbsolutePath string, wslDollarSign bool, serviceNode *ast.MappingValueNode) *protocol.DocumentLink {
	attributeValue := stringNode(serviceNode.Value)
	if attributeValue != nil {
		return createLink(folderAbsolutePath, wslDollarSign, attributeValue.GetToken())
	}
	return nil
}

func stringNode(value ast.Node) *ast.StringNode {
	if s, ok := resolveAnchor(value).(*ast.StringNode); ok {
		return s
	}
	return nil
}

func createdNestedLink(folderAbsolutePath string, wslDollarSign bool, serviceNode *ast.MappingValueNode, parent, child string) *protocol.DocumentLink {
	if resolveAnchor(serviceNode.Key).GetToken().Value == parent {
		if mappingNode, ok := resolveAnchor(serviceNode.Value).(*ast.MappingNode); ok {
			for _, buildAttribute := range mappingNode.Values {
				if resolveAnchor(buildAttribute.Key).GetToken().Value == child {
					return createFileLink(folderAbsolutePath, wslDollarSign, buildAttribute)
				}
			}
		}
	}
	return nil
}

func createImageLink(serviceNode *ast.MappingValueNode) *protocol.DocumentLink {
	if resolveAnchor(serviceNode.Key).GetToken().Value == "image" {
		service := stringNode(serviceNode.Value)
		if service != nil {
			linkedText, link := extractImageLink(service.Value)
			if linkedText != "" {
				return &protocol.DocumentLink{
					Range:   createRange(service.GetToken(), len(linkedText)),
					Target:  types.CreateStringPointer(link),
					Tooltip: types.CreateStringPointer(link),
				}
			}
		}
	}
	return nil
}

func createFileLinks(folderAbsolutePath string, wslDollarSign bool, serviceNode *ast.MappingValueNode, attributeName string) []protocol.DocumentLink {
	if resolveAnchor(serviceNode.Key).GetToken().Value == attributeName {
		if sequence, ok := resolveAnchor(serviceNode.Value).(*ast.SequenceNode); ok {
			links := []protocol.DocumentLink{}
			for _, node := range sequence.Values {
				if s, ok := resolveAnchor(node).(*ast.StringNode); ok {
					links = append(links, *createLink(folderAbsolutePath, wslDollarSign, s.GetToken()))
				}
			}
			return links
		}

		link := createFileLink(folderAbsolutePath, wslDollarSign, serviceNode)
		if link != nil {
			return []protocol.DocumentLink{*link}
		}
	}
	return nil
}

func createVolumeFileLinks(folderAbsolutePath string, wslDollarSign bool, serviceNode *ast.MappingValueNode) []protocol.DocumentLink {
	if resolveAnchor(serviceNode.Key).GetToken().Value == "volumes" {
		if sequence, ok := resolveAnchor(serviceNode.Value).(*ast.SequenceNode); ok {
			links := []protocol.DocumentLink{}
			for _, node := range sequence.Values {
				if s, ok := resolveAnchor(node).(*ast.StringNode); ok {
					config, err := format.ParseVolume(s.GetToken().Value)
					if err == nil && config.Type == composeTypes.VolumeTypeBind {
						uri, path := createLocalFileLink(folderAbsolutePath, config.Source, wslDollarSign)
						info, err := os.Stat(path)
						if err == nil && !info.IsDir() {
							t := volumeToken(s.GetToken())
							links = append(links, protocol.DocumentLink{
								Range:   createRange(t, len(t.Value)),
								Target:  types.CreateStringPointer(uri),
								Tooltip: types.CreateStringPointer(path),
							})
						}
					}
				}
			}
			return links
		}
	}
	return nil
}

func createLocalFileLink(folderAbsolutePath, fsPath string, wslDollarSign bool) (uri, path string) {
	if filepath.IsAbs(fsPath) {
		return fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(fsPath), "/")), fsPath
	}
	return types.Concatenate(folderAbsolutePath, fsPath, wslDollarSign)
}

func createObjectFileLink(folderAbsolutePath string, wslDollarSign bool, serviceNode *ast.MappingValueNode) *protocol.DocumentLink {
	if resolveAnchor(serviceNode.Key).GetToken().Value == "file" {
		return createFileLink(folderAbsolutePath, wslDollarSign, serviceNode)
	}
	return nil
}

func createModelLink(serviceNode *ast.MappingValueNode) *protocol.DocumentLink {
	if resolveAnchor(serviceNode.Key).GetToken().Value == "model" {
		service := stringNode(serviceNode.Value)
		if service != nil {
			linkedText, link := extractModelLink(service.Value)
			if linkedText != "" {
				return &protocol.DocumentLink{
					Range:   createRange(service.GetToken(), len(linkedText)),
					Target:  types.CreateStringPointer(link),
					Tooltip: types.CreateStringPointer(link),
				}
			}
		}
	}
	return nil
}

func includedPaths(nodes []ast.Node) []*token.Token {
	tokens := []*token.Token{}
	for _, entry := range nodes {
		if mappingNode, ok := resolveAnchor(entry).(*ast.MappingNode); ok {
			for _, value := range mappingNode.Values {
				attributeName := resolveAnchor(value.Key).GetToken().Value
				if attributeName == "path" || attributeName == "env_file" {
					if paths, ok := resolveAnchor(value.Value).(*ast.SequenceNode); ok {
						// include:
						//   - path:
						//     - ../commons/compose.yaml
						//     - ./commons-override.yaml
						//   - env_file:
						//     - ../another/.env
						//     - ../another/dev.env
						for _, path := range paths.Values {
							if _, ok := path.(*ast.AliasNode); !ok {
								tokens = append(tokens, resolveAnchor(path).GetToken())
							}
						}
					} else {
						// include:
						// - path: ../commons/compose.yaml
						//   project_directory: ..
						//   env_file: ../another/.env
						if _, ok := value.Value.(*ast.AliasNode); !ok {
							tokens = append(tokens, resolveAnchor(value.Value).GetToken())
						}
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

func scanForLinks(folderAbsolutePath string, wslDollarSign bool, n *ast.MappingValueNode) []protocol.DocumentLink {
	if s, ok := resolveAnchor(n.Key).(*ast.StringNode); ok {
		links := []protocol.DocumentLink{}
		switch s.Value {
		case "include":
			if sequence, ok := resolveAnchor(n.Value).(*ast.SequenceNode); ok {
				for _, token := range includedPaths(sequence.Values) {
					link := createLink(folderAbsolutePath, wslDollarSign, token)
					if link != nil {
						links = append(links, *link)
					}
				}
			}
		case "services":
			if mappingNode, ok := resolveAnchor(n.Value).(*ast.MappingNode); ok {
				for _, node := range mappingNode.Values {
					if serviceAttributes, ok := resolveAnchor(node.Value).(*ast.MappingNode); ok {
						for _, serviceAttribute := range serviceAttributes.Values {
							link := createImageLink(serviceAttribute)
							if link != nil {
								links = append(links, *link)
							}

							link = createdNestedLink(folderAbsolutePath, wslDollarSign, serviceAttribute, "build", "dockerfile")
							if link != nil {
								links = append(links, *link)
							}

							link = createdNestedLink(folderAbsolutePath, wslDollarSign, serviceAttribute, "credential_spec", "file")
							if link != nil {
								links = append(links, *link)
							}

							link = createdNestedLink(folderAbsolutePath, wslDollarSign, serviceAttribute, "extends", "file")
							if link != nil {
								links = append(links, *link)
							}

							envFileLinks := createFileLinks(folderAbsolutePath, wslDollarSign, serviceAttribute, "env_file")
							links = append(links, envFileLinks...)

							labelFileLinks := createFileLinks(folderAbsolutePath, wslDollarSign, serviceAttribute, "label_file")
							links = append(links, labelFileLinks...)

							volumeFileLinks := createVolumeFileLinks(folderAbsolutePath, wslDollarSign, serviceAttribute)
							links = append(links, volumeFileLinks...)
						}
					}
				}
			}
		case "configs":
			if mappingNode, ok := resolveAnchor(n.Value).(*ast.MappingNode); ok {
				for _, node := range mappingNode.Values {
					if configAttributes, ok := resolveAnchor(node.Value).(*ast.MappingNode); ok {
						for _, configAttribute := range configAttributes.Values {
							link := createObjectFileLink(folderAbsolutePath, wslDollarSign, configAttribute)
							if link != nil {
								links = append(links, *link)
							}
						}
					}
				}
			}
		case "secrets":
			if mappingNode, ok := resolveAnchor(n.Value).(*ast.MappingNode); ok {
				for _, node := range mappingNode.Values {
					if configAttributes, ok := resolveAnchor(node.Value).(*ast.MappingNode); ok {
						for _, configAttribute := range configAttributes.Values {
							link := createObjectFileLink(folderAbsolutePath, wslDollarSign, configAttribute)
							if link != nil {
								links = append(links, *link)
							}
						}
					}
				}
			}
		case "models":
			if mappingNode, ok := resolveAnchor(n.Value).(*ast.MappingNode); ok {
				for _, node := range mappingNode.Values {
					if serviceAttributes, ok := resolveAnchor(node.Value).(*ast.MappingNode); ok {
						for _, serviceAttribute := range serviceAttributes.Values {
							link := createModelLink(serviceAttribute)
							if link != nil {
								links = append(links, *link)
							}
						}
					}
				}
			}
		}
		return links
	}
	return nil
}

func DocumentLink(ctx context.Context, documentURI protocol.URI, doc document.ComposeDocument) ([]protocol.DocumentLink, error) {
	d, err := doc.DocumentPath()
	if err != nil {
		return nil, err
	}

	file := doc.File()
	if file == nil || len(file.Docs) == 0 {
		return nil, nil
	}

	links := []protocol.DocumentLink{}
	for _, documentNode := range file.Docs {
		if mappingNode, ok := documentNode.Body.(*ast.MappingNode); ok {
			for _, node := range mappingNode.Values {
				links = append(links, scanForLinks(d.Folder, d.WSLDollarSignHost, node)...)
			}
		}
	}
	return links, nil
}

func extractNonDockerHubImageLink(nodeValue, prefix, uriPrefix string, startIndex uint) (string, string) {
	if len(nodeValue) <= len(prefix)+1 {
		return "", ""
	}
	idx := strings.LastIndex(nodeValue, ":")
	lastSlashIdx := strings.LastIndex(nodeValue, "/")
	if (idx != -1 && lastSlashIdx > idx) || strings.Index(nodeValue, "/") == lastSlashIdx {
		return "", ""
	}
	if idx == -1 {
		return nodeValue, fmt.Sprintf("%v%v", uriPrefix, nodeValue[startIndex:])
	}
	return nodeValue[0:idx], fmt.Sprintf("%v%v", uriPrefix, nodeValue[startIndex:idx])
}

func extractImageLink(nodeValue string) (string, string) {
	if strings.HasPrefix(nodeValue, "ghcr.io") {
		return extractNonDockerHubImageLink(nodeValue, "ghcr.io", "https://", 0)
	}

	if strings.HasPrefix(nodeValue, "mcr.microsoft.com") {
		if len(nodeValue) <= 18 {
			return "", ""
		}
		idx := strings.LastIndex(nodeValue, ":")
		if idx == 17 {
			return "", ""
		}
		lastSlashIdx := strings.LastIndex(nodeValue, "/")
		if lastSlashIdx == idx-1 || (idx != -1 && lastSlashIdx > idx) {
			return "", ""
		}
		if idx == -1 {
			return nodeValue, fmt.Sprintf("https://mcr.microsoft.com/artifact/mar/%v", nodeValue[18:])
		}
		return nodeValue[0:idx], fmt.Sprintf("https://mcr.microsoft.com/artifact/mar/%v", nodeValue[18:idx])
	}

	if strings.HasPrefix(nodeValue, "quay.io") {
		return extractNonDockerHubImageLink(nodeValue, "quay.io", "https://quay.io/repository/", 8)
	}

	idx := strings.LastIndex(nodeValue, ":")
	if idx == -1 {
		idx := strings.Index(nodeValue, "/")
		if idx == -1 {
			return nodeValue, fmt.Sprintf("https://hub.docker.com/_/%v", nodeValue)
		}
		return nodeValue, fmt.Sprintf("https://hub.docker.com/r/%v", nodeValue)
	}

	slashIndex := strings.Index(nodeValue, "/")
	if slashIndex == -1 {
		return nodeValue[0:idx], fmt.Sprintf("https://hub.docker.com/_/%v", nodeValue[0:idx])
	}
	return nodeValue[0:idx], fmt.Sprintf("https://hub.docker.com/r/%v", nodeValue[0:idx])
}

func extractModelLink(nodeValue string) (string, string) {
	if strings.HasPrefix(nodeValue, "hf.co") {
		if len(nodeValue) <= 6 {
			return "", ""
		}
		idx := strings.LastIndex(nodeValue, ":")
		if idx == -1 {
			return nodeValue, fmt.Sprintf("https://%v", nodeValue)
		}
		return nodeValue[0:idx], fmt.Sprintf("https://%v", nodeValue[0:idx])
	}

	idx := strings.LastIndex(nodeValue, ":")
	if idx == -1 {
		idx := strings.Index(nodeValue, "/")
		if idx == -1 {
			return nodeValue, fmt.Sprintf("https://hub.docker.com/_/%v", nodeValue)
		}
		return nodeValue, fmt.Sprintf("https://hub.docker.com/r/%v", nodeValue)
	}

	slashIndex := strings.Index(nodeValue, "/")
	if slashIndex == -1 {
		return nodeValue[0:idx], fmt.Sprintf("https://hub.docker.com/_/%v", nodeValue[0:idx])
	}
	return nodeValue[0:idx], fmt.Sprintf("https://hub.docker.com/r/%v", nodeValue[0:idx])
}
