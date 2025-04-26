package compose

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"log"

	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
	"gopkg.in/yaml.v3"
)

func DocumentLink(ctx context.Context, documentURI protocol.URI, document document.ComposeDocument) ([]protocol.DocumentLink, error) {
	url, err := url.Parse(string(documentURI))
	if err != nil {
		return nil, fmt.Errorf("LSP client sent invalid URI: %v", string(documentURI))
	}

	links := []protocol.DocumentLink{}
	rootNode := document.RootNode()
	for _, serviceNode := range rootNode.Content {
		if serviceNode.Kind == yaml.MappingNode {
			for i := 0; i < len(serviceNode.Content); i += 2 {
				keyNode := serviceNode.Content[i]
				valueNode := serviceNode.Content[i+1]
				if keyNode.Value == "services" && valueNode.Kind == yaml.MappingNode {
					for j := 0; j < len(valueNode.Content); j += 2 {
						serviceValueNode := valueNode.Content[j+1]
						if serviceValueNode.Kind == yaml.MappingNode {
							for k := 0; k < len(serviceValueNode.Content); k += 2 {
								buildKeyNode := serviceValueNode.Content[k]
								buildValueNode := serviceValueNode.Content[k+1]
								if buildKeyNode.Value == "build" && buildValueNode.Kind == yaml.MappingNode {
									for l := 0; l < len(buildValueNode.Content); l += 2 {
										dockerfileKeyNode := buildValueNode.Content[l]
										dockerfileValueNode := buildValueNode.Content[l+1]
										if dockerfileKeyNode.Value == "dockerfile" {
											dockerfilePath := dockerfileValueNode.Value
											dockerfilePath, err = types.AbsolutePath(url, dockerfilePath)
											if err == nil {
												links = append(links, protocol.DocumentLink{
													Range: protocol.Range{
														Start: protocol.Position{Line: uint32(dockerfileValueNode.Line) - 1, Character: uint32(dockerfileValueNode.Column)},
														End:   protocol.Position{Line: uint32(dockerfileValueNode.Line) - 1, Character: uint32(dockerfileValueNode.Column + len(dockerfilePath))},
													},
													Target:  types.CreateStringPointer(protocol.URI(fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(dockerfilePath), "/")))),
													Tooltip: types.CreateStringPointer(dockerfilePath),
												})
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return links, nil
}
