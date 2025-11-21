package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/uri"
)

func documentLinkTooltip(testsFolder, fileName string) *string {
	tooltip := filepath.Join(testsFolder, fileName)
	return &tooltip
}

func documentLinkTarget(testsFolder, fileName string) *string {
	path := filepath.Join(testsFolder, fileName)
	target := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(path), "/"))
	return &target
}

func TestDocumentLink_WSL(t *testing.T) {
	composeStringURI := "file://wsl%24/docker-desktop/tmp/compose.yaml"
	testCases := []struct {
		name    string
		content string
		links   []protocol.DocumentLink
	}{
		{
			name: "included files, string array",
			content: `include:
  - file.yaml
  - ./file2.yaml
  - ../other/file3.yaml`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 4},
						End:   protocol.Position{Line: 1, Character: 13},
					},
					Target:  types.CreateStringPointer("file://wsl%24/docker-desktop/tmp/file.yaml"),
					Tooltip: types.CreateStringPointer("\\\\wsl$\\docker-desktop\\tmp\\file.yaml"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 4},
						End:   protocol.Position{Line: 2, Character: 16},
					},
					Target:  types.CreateStringPointer("file://wsl%24/docker-desktop/tmp/file2.yaml"),
					Tooltip: types.CreateStringPointer("\\\\wsl$\\docker-desktop\\tmp\\file2.yaml"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 4},
						End:   protocol.Position{Line: 3, Character: 23},
					},
					Target:  types.CreateStringPointer("file://wsl%24/docker-desktop/other/file3.yaml"),
					Tooltip: types.CreateStringPointer("\\\\wsl$\\docker-desktop\\other\\file3.yaml"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := document.NewComposeDocument(document.NewDocumentManager(), uri.URI(composeStringURI), 1, []byte(tc.content))
			links, err := DocumentLink(context.Background(), composeStringURI, doc)
			require.NoError(t, err)
			require.Equal(t, tc.links, links)
		})
	}
}

func TestDocumentLink_IncludedFiles(t *testing.T) {
	testsFolder := filepath.Join(os.TempDir(), "composeDocumentLinkTests")
	composeFilePath := filepath.Join(testsFolder, "docker-compose.yml")
	composeStringURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(composeFilePath), "/"))

	testCases := []struct {
		name    string
		content string
		links   []protocol.DocumentLink
	}{
		{
			name:    "empty file",
			content: "",
			links:   []protocol.DocumentLink{},
		},
		{
			name: "included files, string array",
			content: `include:
  - file.yml
  - "file2.yml"`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 4},
						End:   protocol.Position{Line: 1, Character: 12},
					},
					Target:  documentLinkTarget(testsFolder, "file.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file.yml"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 5},
						End:   protocol.Position{Line: 2, Character: 14},
					},
					Target:  documentLinkTarget(testsFolder, "file2.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file2.yml"),
				},
			},
		},
		{
			name: "included files, string array with single quotes",
			content: `include:
  - file.yml
  - 'file2.yml'`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 4},
						End:   protocol.Position{Line: 1, Character: 12},
					},
					Target:  documentLinkTarget(testsFolder, "file.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file.yml"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 5},
						End:   protocol.Position{Line: 2, Character: 14},
					},
					Target:  documentLinkTarget(testsFolder, "file2.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file2.yml"),
				},
			},
		},
		{
			name: "included files, path as a string",
			content: `include:
  - path: file.yml
  - path: "file2.yml"`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 10},
						End:   protocol.Position{Line: 1, Character: 18},
					},
					Target:  documentLinkTarget(testsFolder, "file.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file.yml"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 11},
						End:   protocol.Position{Line: 2, Character: 20},
					},
					Target:  documentLinkTarget(testsFolder, "file2.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file2.yml"),
				},
			},
		},
		{
			name: "included files, path as a string with single quotes",
			content: `include:
  - path: file.yml
  - path: 'file2.yml'`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 10},
						End:   protocol.Position{Line: 1, Character: 18},
					},
					Target:  documentLinkTarget(testsFolder, "file.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file.yml"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 11},
						End:   protocol.Position{Line: 2, Character: 20},
					},
					Target:  documentLinkTarget(testsFolder, "file2.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file2.yml"),
				},
			},
		},
		{
			name: "included files, attribute misspelt",
			content: `include:
  - path2: file.yml`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "included files, mixed paths",
			content: `
include:
  - file.yml
  - path: file2.yml
  - path:
    - file3.yml`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 4},
						End:   protocol.Position{Line: 2, Character: 12},
					},
					Target:  documentLinkTarget(testsFolder, "file.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file.yml"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 10},
						End:   protocol.Position{Line: 3, Character: 19},
					},
					Target:  documentLinkTarget(testsFolder, "file2.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file2.yml"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 5, Character: 6},
						End:   protocol.Position{Line: 5, Character: 15},
					},
					Target:  documentLinkTarget(testsFolder, "file3.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file3.yml"),
				},
			},
		},
		{
			name: "anchor on the include object's attribute",
			content: `
include: &anchor
  - file.yml
  - path: file2.yml
  - path:
    - file3.yml`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 4},
						End:   protocol.Position{Line: 2, Character: 12},
					},
					Target:  documentLinkTarget(testsFolder, "file.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file.yml"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 10},
						End:   protocol.Position{Line: 3, Character: 19},
					},
					Target:  documentLinkTarget(testsFolder, "file2.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file2.yml"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 5, Character: 6},
						End:   protocol.Position{Line: 5, Character: 15},
					},
					Target:  documentLinkTarget(testsFolder, "file3.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file3.yml"),
				},
			},
		},
		{
			name:    "include declared with no content",
			content: "include:",
			links:   []protocol.DocumentLink{},
		},
		{
			name: "regular file",
			content: `
services:
  backend:

include:
  - file.yml
  - "file2.yml"`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 5, Character: 4},
						End:   protocol.Position{Line: 5, Character: 12},
					},
					Target:  documentLinkTarget(testsFolder, "file.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file.yml"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 6, Character: 5},
						End:   protocol.Position{Line: 6, Character: 14},
					},
					Target:  documentLinkTarget(testsFolder, "file2.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file2.yml"),
				},
			},
		},
		{
			name: "regular file with single quotes",
			content: `
services:
  backend:

include:
  - file.yml
  - 'file2.yml'`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 5, Character: 4},
						End:   protocol.Position{Line: 5, Character: 12},
					},
					Target:  documentLinkTarget(testsFolder, "file.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file.yml"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 6, Character: 5},
						End:   protocol.Position{Line: 6, Character: 14},
					},
					Target:  documentLinkTarget(testsFolder, "file2.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file2.yml"),
				},
			},
		},
		{
			name: "anchors and aliases",
			content: `
include:
  - &link compose.other.yaml
  - *link`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 10},
						End:   protocol.Position{Line: 2, Character: 28},
					},
					Target:  documentLinkTarget(testsFolder, "compose.other.yaml"),
					Tooltip: documentLinkTooltip(testsFolder, "compose.other.yaml"),
				},
			},
		},
		{
			name: "anchor on the include object itself",
			content: `
&anchor include:
  - compose.other.yaml`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 4},
						End:   protocol.Position{Line: 2, Character: 22},
					},
					Target:  documentLinkTarget(testsFolder, "compose.other.yaml"),
					Tooltip: documentLinkTooltip(testsFolder, "compose.other.yaml"),
				},
			},
		},
		{
			name: "anchor on the path's string attribute",
			content: `
include:
  - path: &anchor file2.yml`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 18},
						End:   protocol.Position{Line: 2, Character: 27},
					},
					Target:  documentLinkTarget(testsFolder, "file2.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file2.yml"),
				},
			},
		},
		{
			name: "anchor on the path's string attribute name",
			content: `
include:
  - &anchor path: file2.yml`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 18},
						End:   protocol.Position{Line: 2, Character: 27},
					},
					Target:  documentLinkTarget(testsFolder, "file2.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file2.yml"),
				},
			},
		},
		{
			name: "anchor on the path object's value",
			content: `
include:
  - path: &anchor
    - file2.yml`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 6},
						End:   protocol.Position{Line: 3, Character: 15},
					},
					Target:  documentLinkTarget(testsFolder, "file2.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file2.yml"),
				},
			},
		},
		{
			name: "anchor on the path array item's string value",
			content: `
include:
  - path:
    - &anchor file2.yml`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 14},
						End:   protocol.Position{Line: 3, Character: 23},
					},
					Target:  documentLinkTarget(testsFolder, "file2.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file2.yml"),
				},
			},
		},
		{
			name: "anchor on the include array item itself",
			content: `
include:
  - &anchor { path: file2.yml }`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 20},
						End:   protocol.Position{Line: 2, Character: 29},
					},
					Target:  documentLinkTarget(testsFolder, "file2.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file2.yml"),
				},
			},
		},
		{
			name: "anchor on the path object",
			content: `
include:
  - &anchor path:
    - file2.yml`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 6},
						End:   protocol.Position{Line: 3, Character: 15},
					},
					Target:  documentLinkTarget(testsFolder, "file2.yml"),
					Tooltip: documentLinkTooltip(testsFolder, "file2.yml"),
				},
			},
		},
		{
			name: "alias on the path's string attribute",
			content: `
include:
  - path: *anchor`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "alias on the path attribute's string array item value",
			content: `
include:
  - path:
    - *anchor`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "env_file string attribute",
			content: `
include:
  - env_file: .env
  - env_file: '.env'
  - env_file: ".env"`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 14},
						End:   protocol.Position{Line: 2, Character: 18},
					},
					Target:  documentLinkTarget(testsFolder, ".env"),
					Tooltip: documentLinkTooltip(testsFolder, ".env"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 15},
						End:   protocol.Position{Line: 3, Character: 19},
					},
					Target:  documentLinkTarget(testsFolder, ".env"),
					Tooltip: documentLinkTooltip(testsFolder, ".env"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 4, Character: 15},
						End:   protocol.Position{Line: 4, Character: 19},
					},
					Target:  documentLinkTarget(testsFolder, ".env"),
					Tooltip: documentLinkTooltip(testsFolder, ".env"),
				},
			},
		},
		{
			name: "env_file string attribute with an anchor",
			content: `
include:
  - env_file: &anchor .env`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 22},
						End:   protocol.Position{Line: 2, Character: 26},
					},
					Target:  documentLinkTarget(testsFolder, ".env"),
					Tooltip: documentLinkTooltip(testsFolder, ".env"),
				},
			},
		},
		{
			name: "env_file string attribute with an alias",
			content: `
include:
  - env_file: *alias`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "env_file array attribute",
			content: `
include:
  - env_file:
    - .env
    - '.env'
    - ".env"`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 6},
						End:   protocol.Position{Line: 3, Character: 10},
					},
					Target:  documentLinkTarget(testsFolder, ".env"),
					Tooltip: documentLinkTooltip(testsFolder, ".env"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 4, Character: 7},
						End:   protocol.Position{Line: 4, Character: 11},
					},
					Target:  documentLinkTarget(testsFolder, ".env"),
					Tooltip: documentLinkTooltip(testsFolder, ".env"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 5, Character: 7},
						End:   protocol.Position{Line: 5, Character: 11},
					},
					Target:  documentLinkTarget(testsFolder, ".env"),
					Tooltip: documentLinkTooltip(testsFolder, ".env"),
				},
			},
		},
		{
			name: "env_file array attribute with an anchor",
			content: `
include:
  - env_file:
    - &anchor .env`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 14},
						End:   protocol.Position{Line: 3, Character: 18},
					},
					Target:  documentLinkTarget(testsFolder, ".env"),
					Tooltip: documentLinkTooltip(testsFolder, ".env"),
				},
			},
		},
		{
			name: "env_file array attribute with an alias",
			content: `
include:
  - env_file:
    - *alias`,
			links: []protocol.DocumentLink{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := document.NewComposeDocument(document.NewDocumentManager(), uri.URI(composeStringURI), 1, []byte(tc.content))
			links, err := DocumentLink(context.Background(), composeStringURI, doc)
			require.NoError(t, err)
			require.Equal(t, tc.links, links)
		})
	}
}

func TestDocumentLink_ServiceImageLinks(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		links   []protocol.DocumentLink
	}{
		{
			name: "image: alpine",
			content: `
services:
  test:
    image: alpine`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 17},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/_/alpine"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/_/alpine"),
				},
			},
		},
		{
			name: "double quoted string",
			content: `
services:
  test:
    image: "alpine"`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 12},
						End:   protocol.Position{Line: 3, Character: 18},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/_/alpine"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/_/alpine"),
				},
			},
		},
		{
			name: "single quoted string",
			content: `
services:
  test:
    image: 'alpine'`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 12},
						End:   protocol.Position{Line: 3, Character: 18},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/_/alpine"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/_/alpine"),
				},
			},
		},
		{
			name: "two services",
			content: `
services:
  test:
    image: alpine
  test2:
    image: postgres`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 17},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/_/alpine"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/_/alpine"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 5, Character: 11},
						End:   protocol.Position{Line: 5, Character: 19},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/_/postgres"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/_/postgres"),
				},
			},
		},
		{
			name: "image: alpine:1.23",
			content: `
services:
  test:
    image: alpine:1.23`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 17},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/_/alpine"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/_/alpine"),
				},
			},
		},
		{
			name: "image: grafana/grafana",
			content: `
services:
  test:
    image: grafana/grafana`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 26},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/r/grafana/grafana"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/r/grafana/grafana"),
				},
			},
		},
		{
			name: "image: grafana/grafana:1.23",
			content: `
services:
  test:
    image: grafana/grafana:1.23`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 26},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/r/grafana/grafana"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/r/grafana/grafana"),
				},
			},
		},
		{
			name: "image: ghcr.io/super-linter/super-linter",
			content: `
services:
  test:
    image: ghcr.io/super-linter/super-linter`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 44},
					},
					Target:  types.CreateStringPointer("https://ghcr.io/super-linter/super-linter"),
					Tooltip: types.CreateStringPointer("https://ghcr.io/super-linter/super-linter"),
				},
			},
		},
		{
			name: "image: ghcr.io/super-linter/super-linter:1.23",
			content: `
services:
  test:
    image: ghcr.io/super-linter/super-linter:1.23`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 44},
					},
					Target:  types.CreateStringPointer("https://ghcr.io/super-linter/super-linter"),
					Tooltip: types.CreateStringPointer("https://ghcr.io/super-linter/super-linter"),
				},
			},
		},
		{
			name: "image: ghcr.io",
			content: `
services:
  test:
    image: ghcr.io`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: ghcr.io/",
			content: `
services:
  test:
    image: ghcr.io/`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: ghcr.io:",
			content: `
services:
  test:
    image: ghcr.io:`,
			links: nil,
		},
		{
			name: "image: ghcr.io:tag",
			content: `
services:
  test:
    image: ghcr.io:tag`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: ghcr.io/:tag",
			content: `
services:
  test:
    image: ghcr.io/:tag`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: ghcr.io/:tag",
			content: `
services:
  test:
    image: ghcr.io:tag/`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: ghcr.io/:tag",
			content: `
services:
  test:
    image: ghcr.io/:tag/`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: \"ghcr.io:\"",
			content: `
services:
  test:
    image: "ghcr.io:tag"`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: 'ghcr.io:'",
			content: `
services:
  test:
    image: 'ghcr.io:tag'`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: mcr.microsoft.com/powershell",
			content: `
services:
  test:
    image: mcr.microsoft.com/powershell`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 39},
					},
					Target:  types.CreateStringPointer("https://mcr.microsoft.com/artifact/mar/powershell"),
					Tooltip: types.CreateStringPointer("https://mcr.microsoft.com/artifact/mar/powershell"),
				},
			},
		},
		{
			name: "image: mcr.microsoft.com/powershell:1.23",
			content: `
services:
  test:
    image: mcr.microsoft.com/powershell:1.23`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 39},
					},
					Target:  types.CreateStringPointer("https://mcr.microsoft.com/artifact/mar/powershell"),
					Tooltip: types.CreateStringPointer("https://mcr.microsoft.com/artifact/mar/powershell"),
				},
			},
		},
		{
			name: "image: mcr.microsoft.com/windows/servercore",
			content: `
services:
  test:
    image: mcr.microsoft.com/windows/servercore`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 47},
					},
					Target:  types.CreateStringPointer("https://mcr.microsoft.com/artifact/mar/windows/servercore"),
					Tooltip: types.CreateStringPointer("https://mcr.microsoft.com/artifact/mar/windows/servercore"),
				},
			},
		},
		{
			name: "image: mcr.microsoft.com/windows/servercore:1.23",
			content: `
services:
  test:
    image: mcr.microsoft.com/windows/servercore:1.23`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 47},
					},
					Target:  types.CreateStringPointer("https://mcr.microsoft.com/artifact/mar/windows/servercore"),
					Tooltip: types.CreateStringPointer("https://mcr.microsoft.com/artifact/mar/windows/servercore"),
				},
			},
		},
		{
			name: "image: mcr.microsoft.com",
			content: `
services:
  test:
    image: mcr.microsoft.com`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: mcr.microsoft.com/",
			content: `
services:
  test:
    image: mcr.microsoft.com/`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: mcr.microsoft.com:",
			content: `
services:
  test:
    image: mcr.microsoft.com:`,
			links: nil,
		},
		{
			name: "image: mcr.microsoft.com:tag",
			content: `
services:
  test:
    image: mcr.microsoft.com:tag`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: mcr.microsoft.com/:tag",
			content: `
services:
  test:
    image: mcr.microsoft.com/:tag`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: mcr.microsoft.com:tag/",
			content: `
services:
  test:
    image: mcr.microsoft.com:tag/`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: mcr.microsoft.com/:tag/",
			content: `
services:
  test:
    image: mcr.microsoft.com/:tag/`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: \"mcr.microsoft.com:\"",
			content: `
services:
  test:
    image: "mcr.microsoft.com:"`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: 'mcr.microsoft.com:'",
			content: `
services:
  test:
    image: 'mcr.microsoft.com:'`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: quay.io/prometheus/node-exporter",
			content: `
services:
  test:
    image: quay.io/prometheus/node-exporter`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 43},
					},
					Target:  types.CreateStringPointer("https://quay.io/repository/prometheus/node-exporter"),
					Tooltip: types.CreateStringPointer("https://quay.io/repository/prometheus/node-exporter"),
				},
			},
		},
		{
			name: "image: quay.io/prometheus/node-exporter:v1.9.1",
			content: `
services:
  test:
    image: quay.io/prometheus/node-exporter:v1.9.1`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 43},
					},
					Target:  types.CreateStringPointer("https://quay.io/repository/prometheus/node-exporter"),
					Tooltip: types.CreateStringPointer("https://quay.io/repository/prometheus/node-exporter"),
				},
			},
		},
		{
			name: "image: quay.io",
			content: `
services:
  test:
    image: quay.io`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: quay.io/",
			content: `
services:
  test:
    image: quay.io/`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: quay.io:",
			content: `
services:
  test:
    image: quay.io:`,
			links: nil,
		},
		{
			name: "image: quay.io:tag",
			content: `
services:
  test:
    image: quay.io:tag`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: quay.io/:tag",
			content: `
services:
  test:
    image: quay.io/:tag`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: quay.io:tag/",
			content: `
services:
  test:
    image: quay.io:tag/`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: quay.io/:tag/",
			content: `
services:
  test:
    image: quay.io/:tag/`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: \"quay.io:\"",
			content: `
services:
  test:
    image: "quay.io:"`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "image: 'quay.io:'",
			content: `
services:
  test:
    image: 'quay.io:'`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "anchors and aliases to nothing",
			content: `
services:
  backend:
    image: &alpine
  backend2:
    image: *alpine`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "anchor has string content",
			content: `
services:
  backend:
    image: &alpine alpine:3.21`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 19},
						End:   protocol.Position{Line: 3, Character: 25},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/_/alpine"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/_/alpine"),
				},
			},
		},
		{
			name: "anchor on the services object itself",
			content: `
&anchor services:
  test:
    image: alpine`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 17},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/_/alpine"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/_/alpine"),
				},
			},
		},
		{
			name: "anchor on the services object's value",
			content: `
services: &anchor
  test:
    image: alpine`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 17},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/_/alpine"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/_/alpine"),
				},
			},
		},
		{
			name: "anchor on the service object itself",
			content: `
services:
  test: &anchor { image: alpine }`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 25},
						End:   protocol.Position{Line: 2, Character: 31},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/_/alpine"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/_/alpine"),
				},
			},
		},
		{
			name: "anchor on the image attribute",
			content: `
services:
  test:
    &anchor image: alpine`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 19},
						End:   protocol.Position{Line: 3, Character: 25},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/_/alpine"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/_/alpine"),
				},
			},
		},
		{
			name: "anchor on the service object",
			content: `
services:
  test: &anchor
    image: alpine`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 17},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/_/alpine"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/_/alpine"),
				},
			},
		},
		{
			name: "invalid services",
			content: `
services:
  - `,
			links: []protocol.DocumentLink{},
		},
		{
			name: "two documents",
			content: `
---
services:
  backend:
    image: alpine:3.20
---
services:
  backend2:
    image: alpine:3.21`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 4, Character: 11},
						End:   protocol.Position{Line: 4, Character: 17},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/_/alpine"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/_/alpine"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 8, Character: 11},
						End:   protocol.Position{Line: 8, Character: 17},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/_/alpine"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/_/alpine"),
				},
			},
		},
	}

	composeStringURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(os.TempDir(), "compose.yaml")), "/"))
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := document.NewDocumentManager()
			doc := document.NewComposeDocument(mgr, "docker-compose.yml", 1, []byte(tc.content))
			links, err := DocumentLink(context.Background(), composeStringURI, doc)
			require.NoError(t, err)
			require.Equal(t, tc.links, links)
		})
	}
}

func TestDocumentLink_ServiceBuildDockerfileLinks(t *testing.T) {
	testsFolder := filepath.Join(os.TempDir(), t.Name())
	composeStringURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(testsFolder, "compose.yaml")), "/"))

	testCases := []struct {
		name      string
		content   string
		path      string
		linkRange protocol.Range
	}{
		{
			name: "./Dockerfile2",
			content: `
services:
  test:
    build:
      dockerfile: ./Dockerfile2`,
			path: filepath.Join(testsFolder, "Dockerfile2"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 18},
				End:   protocol.Position{Line: 4, Character: 31},
			},
		},
		{
			name: `"./Dockerfile2"`,
			content: `
services:
  test:
    build:
      dockerfile: "./Dockerfile2"`,
			path: filepath.Join(testsFolder, "Dockerfile2"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 19},
				End:   protocol.Position{Line: 4, Character: 32},
			},
		},
		{
			name: `'./Dockerfile2'`,
			content: `
services:
  test:
    build:
      dockerfile: './Dockerfile2'`,
			path: filepath.Join(testsFolder, "Dockerfile2"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 19},
				End:   protocol.Position{Line: 4, Character: 32},
			},
		},
		{
			name: "anchors and aliases to nothing",
			content: `
services:
  test:
    build:
      dockerfile: &file
  test2:
    build:
      dockerfile: *file`,
		},
		{
			name: "anchor has string content",
			content: `
services:
  test:
    build:
      dockerfile: &file ./Dockerfile2
  test2:
    build:
      dockerfile: *file`,
			path: filepath.Join(testsFolder, "Dockerfile2"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 24},
				End:   protocol.Position{Line: 4, Character: 37},
			},
		},
		{
			name: "anchor on the services object itself",
			content: `
&anchor services:
  test:
    build:
      dockerfile: ./Dockerfile2`,
			path: filepath.Join(testsFolder, "Dockerfile2"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 18},
				End:   protocol.Position{Line: 4, Character: 31},
			},
		},
		{
			name: "anchor on the services object's value",
			content: `
services: &anchor
  test:
    build:
      dockerfile: ./Dockerfile2`,
			path: filepath.Join(testsFolder, "Dockerfile2"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 18},
				End:   protocol.Position{Line: 4, Character: 31},
			},
		},
		{
			name: "anchor on the service JSON object",
			content: `
services:
  test: &anchor { build: { dockerfile: ./Dockerfile2 } }`,
			path: filepath.Join(testsFolder, "Dockerfile2"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 39},
				End:   protocol.Position{Line: 2, Character: 52},
			},
		},
		{
			name: "anchor on the service object",
			content: `
services:
  test: &anchor
    build:
      dockerfile: ./Dockerfile2`,
			path: filepath.Join(testsFolder, "Dockerfile2"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 18},
				End:   protocol.Position{Line: 4, Character: 31},
			},
		},
		{
			name: "anchor on the build attribute inside a JSON object",
			content: `
services:
    backend: {
      &anchor build: { dockerfile: Dockerfile2 }
    }`,
			path: filepath.Join(testsFolder, "Dockerfile2"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 35},
				End:   protocol.Position{Line: 3, Character: 46},
			},
		},
		{
			name: "anchor on the build object",
			content: `
services:
  test:
    build: &anchor
      dockerfile: ./Dockerfile2`,
			path: filepath.Join(testsFolder, "Dockerfile2"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 18},
				End:   protocol.Position{Line: 4, Character: 31},
			},
		},
		{
			name: "anchor on the dockerfile attribute",
			content: `
services:
  test:
    build:
      &anchor dockerfile: ./Dockerfile2`,
			path: filepath.Join(testsFolder, "Dockerfile2"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 26},
				End:   protocol.Position{Line: 4, Character: 39},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := document.NewDocumentManager()
			doc := document.NewComposeDocument(mgr, uri.URI(composeStringURI), 1, []byte(tc.content))
			links, err := DocumentLink(context.Background(), composeStringURI, doc)
			require.NoError(t, err)
			if tc.path == "" {
				require.Equal(t, []protocol.DocumentLink{}, links)
			} else {
				link := protocol.DocumentLink{
					Range:   tc.linkRange,
					Target:  types.CreateStringPointer(fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(tc.path), "/"))),
					Tooltip: types.CreateStringPointer(filepath.FromSlash(tc.path)),
				}
				require.Equal(t, []protocol.DocumentLink{link}, links)
			}
		})
	}
}

func TestDocumentLink_ServiceCredentialSpecFileLinks(t *testing.T) {
	testsFolder := filepath.Join(os.TempDir(), t.Name())
	composeStringURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(testsFolder, "compose.yaml")), "/"))

	testCases := []struct {
		name      string
		content   string
		path      string
		linkRange protocol.Range
	}{
		{
			name: "./credential-spec.json",
			content: `
services:
  test:
    credential_spec:
      file: ./credential-spec.json`,
			path: filepath.Join(testsFolder, "credential-spec.json"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 12},
				End:   protocol.Position{Line: 4, Character: 34},
			},
		},
		{
			name: `"./credential-spec.json"`,
			content: `
services:
  test:
    credential_spec:
      file: "./credential-spec.json"`,
			path: filepath.Join(testsFolder, "credential-spec.json"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 13},
				End:   protocol.Position{Line: 4, Character: 35},
			},
		},
		{
			name: `'./credential-spec.json'`,
			content: `
services:
  test:
    credential_spec:
      file: './credential-spec.json'`,
			path: filepath.Join(testsFolder, "credential-spec.json"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 13},
				End:   protocol.Position{Line: 4, Character: 35},
			},
		},
		{
			name: "anchors and aliases to nothing",
			content: `
secrets:
  test:
    credential_spec:
      file: &credentialSpecFile
  test2:
    credential_spec:
      file: *credentialSpecFile`,
		},
		{
			name: "anchor has string content",
			content: `
services:
  test:
    credential_spec:
      file: &credentialSpecFile ./credential-spec.json
  test2:
    credential_spec:
      file: *credentialSpecFile`,
			path: filepath.Join(testsFolder, "credential-spec.json"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 32},
				End:   protocol.Position{Line: 4, Character: 54},
			},
		},
		{
			name: "anchor on the services object itself",
			content: `
&anchor services:
  test:
    credential_spec:
      file: ./credential-spec.json`,
			path: filepath.Join(testsFolder, "credential-spec.json"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 12},
				End:   protocol.Position{Line: 4, Character: 34},
			},
		},
		{
			name: "anchor on the services object's value",
			content: `
services: &anchor
  test:
    credential_spec:
      file: ./credential-spec.json`,
			path: filepath.Join(testsFolder, "credential-spec.json"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 12},
				End:   protocol.Position{Line: 4, Character: 34},
			},
		},
		{
			name: "anchor on the service JSON object",
			content: `
services:
  test: &anchor { credential_spec: { file: ./credential-spec.json } }`,
			path: filepath.Join(testsFolder, "credential-spec.json"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 43},
				End:   protocol.Position{Line: 2, Character: 65},
			},
		},
		{
			name: "anchor on the service object",
			content: `
services:
  test: &anchor
    credential_spec:
      file: ./credential-spec.json`,
			path: filepath.Join(testsFolder, "credential-spec.json"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 12},
				End:   protocol.Position{Line: 4, Character: 34},
			},
		},
		{
			name: "anchor on the credential_spec attribute inside a JSON object",
			content: `
services:
    backend: {
      &anchor credential_spec: { file: ./credential-spec.json }
    }`,
			path: filepath.Join(testsFolder, "credential-spec.json"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 39},
				End:   protocol.Position{Line: 3, Character: 61},
			},
		},
		{
			name: "anchor on the credential_spec object",
			content: `
services:
  test:
    credential_spec: &anchor
      file: ./credential-spec.json`,
			path: filepath.Join(testsFolder, "credential-spec.json"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 12},
				End:   protocol.Position{Line: 4, Character: 34},
			},
		},
		{
			name: "anchor on the file attribute",
			content: `
services:
  test:
    credential_spec:
      &anchor file: ./credential-spec.json`,
			path: filepath.Join(testsFolder, "credential-spec.json"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 20},
				End:   protocol.Position{Line: 4, Character: 42},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := document.NewDocumentManager()
			doc := document.NewComposeDocument(mgr, uri.URI(composeStringURI), 1, []byte(tc.content))
			links, err := DocumentLink(context.Background(), composeStringURI, doc)
			require.NoError(t, err)
			if tc.path == "" {
				require.Equal(t, []protocol.DocumentLink{}, links)
			} else {
				link := protocol.DocumentLink{
					Range:   tc.linkRange,
					Target:  types.CreateStringPointer(fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(tc.path), "/"))),
					Tooltip: types.CreateStringPointer(filepath.FromSlash(tc.path)),
				}
				require.Equal(t, []protocol.DocumentLink{link}, links)
			}
		})
	}
}

func TestDocumentLink_ServiceEnvFileLinks(t *testing.T) {
	testsFolder := filepath.Join(os.TempDir(), t.Name())
	composeStringURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(testsFolder, "compose.yaml")), "/"))

	testCases := []struct {
		name      string
		content   string
		path      string
		linkRange protocol.Range
	}{
		{
			name: "string value app.labels",
			content: `
services:
  test:
    env_file: .env`,
			path: filepath.Join(testsFolder, "./.env"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 14},
				End:   protocol.Position{Line: 3, Character: 18},
			},
		},
		{
			name: "string value ./.env",
			content: `
services:
  test:
    env_file: ./.env`,
			path: filepath.Join(testsFolder, "./.env"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 14},
				End:   protocol.Position{Line: 3, Character: 20},
			},
		},
		{
			name: "double quoted string value \"./.env\"",
			content: `
services:
  test:
    env_file: "./.env"`,
			path: filepath.Join(testsFolder, "./.env"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 15},
				End:   protocol.Position{Line: 3, Character: 21},
			},
		},
		{
			name: "single quoted string value './.env'",
			content: `
services:
  test:
    env_file: './.env'`,
			path: filepath.Join(testsFolder, "./.env"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 15},
				End:   protocol.Position{Line: 3, Character: 21},
			},
		},
		{
			name: "attribute value is null",
			content: `
services:
  test:
    env_file: null`,
		},
		{
			name: "array item as a string",
			content: `
services:
  test:
    env_file:
      - ./.env`,
			path: filepath.Join(testsFolder, "./.env"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 8},
				End:   protocol.Position{Line: 4, Character: 14},
			},
		},
		{
			name: "array item as a double quoted string",
			content: `
services:
  test:
    env_file:
      - "./.env"`,
			path: filepath.Join(testsFolder, "./.env"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 9},
				End:   protocol.Position{Line: 4, Character: 15},
			},
		},
		{
			name: "array item as a single quoted string",
			content: `
services:
  test:
    env_file:
      - './.env'`,
			path: filepath.Join(testsFolder, "./.env"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 9},
				End:   protocol.Position{Line: 4, Character: 15},
			},
		},
		{
			name: "array item is null",
			content: `
services:
  test:
    env_file:
      - null`,
		},
		{
			name: "anchors and aliases to nothing",
			content: `
services:
  test:
    env_file: &anchor
  test2:
    env_file: *anchor`,
		},
		{
			name: "anchor on the services object itself",
			content: `
&anchor services:
  test:
    env_file: ./.env`,
			path: filepath.Join(testsFolder, "./.env"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 14},
				End:   protocol.Position{Line: 3, Character: 20},
			},
		},
		{
			name: "anchor on the services object's value",
			content: `
services: &anchor
  test:
    env_file: ./.env`,
			path: filepath.Join(testsFolder, "./.env"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 14},
				End:   protocol.Position{Line: 3, Character: 20},
			},
		},
		{
			name: "anchor on the service object itself",
			content: `
services:
  &anchor test:
    env_file: ./.env`,
			path: filepath.Join(testsFolder, "./.env"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 14},
				End:   protocol.Position{Line: 3, Character: 20},
			},
		},
		{
			name: "anchor on the service object's value",
			content: `
services:
  test: &anchor
    env_file: ./.env`,
			path: filepath.Join(testsFolder, "./.env"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 14},
				End:   protocol.Position{Line: 3, Character: 20},
			},
		},
		{
			name: "anchor on the service object's value as JSON",
			content: `
services:
  test: &anchor { env_file: ./.env }`,
			path: filepath.Join(testsFolder, "./.env"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 28},
				End:   protocol.Position{Line: 2, Character: 34},
			},
		},
		{
			name: "anchor on the env_file string attribute itself",
			content: `
services:
  test: &anchor
    &anchor env_file: ./.env`,
			path: filepath.Join(testsFolder, "./.env"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 22},
				End:   protocol.Position{Line: 3, Character: 28},
			},
		},
		{
			name: "anchor on the env_file string attribute's value",
			content: `
services:
  test: &anchor
    env_file: &anchor ./.env`,
			path: filepath.Join(testsFolder, "./.env"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 22},
				End:   protocol.Position{Line: 3, Character: 28},
			},
		},
		{
			name: "anchor on the env_file array attribute's value",
			content: `
services:
  test: &anchor
    env_file: &anchor
      - ./.env`,
			path: filepath.Join(testsFolder, "./.env"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 8},
				End:   protocol.Position{Line: 4, Character: 14},
			},
		},
		{
			name: "anchor on the env_file array item's value",
			content: `
services:
  test: &anchor
    env_file:
      - &anchor ./.env`,
			path: filepath.Join(testsFolder, "./.env"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 16},
				End:   protocol.Position{Line: 4, Character: 22},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := document.NewDocumentManager()
			doc := document.NewComposeDocument(mgr, uri.URI(composeStringURI), 1, []byte(tc.content))
			links, err := DocumentLink(context.Background(), composeStringURI, doc)
			require.NoError(t, err)
			if tc.path == "" {
				require.Equal(t, []protocol.DocumentLink{}, links)
			} else {
				link := protocol.DocumentLink{
					Range:   tc.linkRange,
					Target:  types.CreateStringPointer(fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(tc.path), "/"))),
					Tooltip: types.CreateStringPointer(filepath.FromSlash(tc.path)),
				}
				require.Equal(t, []protocol.DocumentLink{link}, links)
			}
		})
	}
}

func TestDocumentLink_ServiceExtendsFileLinks(t *testing.T) {
	testsFolder := filepath.Join(os.TempDir(), t.Name())
	composeStringURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(testsFolder, "compose.yaml")), "/"))

	testCases := []struct {
		name      string
		content   string
		path      string
		linkRange protocol.Range
	}{
		{
			name: "string value",
			content: `
services:
  test2:
    extends:
      service: test
      file: ./compose.other.yaml`,
			path: filepath.Join(testsFolder, "compose.other.yaml"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 12},
				End:   protocol.Position{Line: 5, Character: 32},
			},
		},
		{
			name: "string value in single quotes",
			content: `
services:
  test2:
    extends:
      service: test
      file: './compose.other.yaml'`,
			path: filepath.Join(testsFolder, "compose.other.yaml"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 13},
				End:   protocol.Position{Line: 5, Character: 33},
			},
		},
		{
			name: "string value in double quotes",
			content: `
services:
  test2:
    extends:
      service: test
      file: "./compose.other.yaml"`,
			path: filepath.Join(testsFolder, "compose.other.yaml"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 13},
				End:   protocol.Position{Line: 5, Character: 33},
			},
		},
		{
			name: "anchors and aliases to nothing",
			content: `
services:
  test:
    extends:
      file: &file
  test2:
    extends:
      file: *file`,
		},
		{
			name: "anchor has string content",
			content: `
services:
  test:
    extends:
      file: &file ./compose.other.yaml
  test2:
    extends:
      file: *file`,
			path: filepath.Join(testsFolder, "compose.other.yaml"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 18},
				End:   protocol.Position{Line: 4, Character: 38},
			},
		},
		{
			name: "anchor on the services object itself",
			content: `
&anchor services:
  test:
    extends:
      file: ./compose.other.yaml`,
			path: filepath.Join(testsFolder, "compose.other.yaml"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 12},
				End:   protocol.Position{Line: 4, Character: 32},
			},
		},
		{
			name: "anchor on the services object's value",
			content: `
services: &anchor
  test:
    extends:
      file: ./compose.other.yaml`,
			path: filepath.Join(testsFolder, "compose.other.yaml"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 12},
				End:   protocol.Position{Line: 4, Character: 32},
			},
		},
		{
			name: "anchor on the service JSON object",
			content: `
services:
  test: &anchor { extends: { file: ./compose.other.yaml } }`,
			path: filepath.Join(testsFolder, "compose.other.yaml"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 35},
				End:   protocol.Position{Line: 2, Character: 55},
			},
		},
		{
			name: "anchor on the service object",
			content: `
services:
  test: &anchor
    extends:
      file: ./compose.other.yaml`,
			path: filepath.Join(testsFolder, "compose.other.yaml"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 12},
				End:   protocol.Position{Line: 4, Character: 32},
			},
		},
		{
			name: "anchor on the build attribute inside a JSON object",
			content: `
services:
  backend: {
    &anchor extends: { file: ./compose.other.yaml } 
  }`,
			path: filepath.Join(testsFolder, "compose.other.yaml"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 29},
				End:   protocol.Position{Line: 3, Character: 49},
			},
		},
		{
			name: "anchor on the extends object",
			content: `
services:
  test:
    extends: &anchor
      file: ./compose.other.yaml`,
			path: filepath.Join(testsFolder, "compose.other.yaml"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 12},
				End:   protocol.Position{Line: 4, Character: 32},
			},
		},
		{
			name: "anchor on the file attribute",
			content: `
services:
  test:
    extends:
      &anchor file: ./compose.other.yaml`,
			path: filepath.Join(testsFolder, "./compose.other.yaml"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 20},
				End:   protocol.Position{Line: 4, Character: 40},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := document.NewDocumentManager()
			doc := document.NewComposeDocument(mgr, uri.URI(composeStringURI), 1, []byte(tc.content))
			links, err := DocumentLink(context.Background(), composeStringURI, doc)
			require.NoError(t, err)
			if tc.path == "" {
				require.Equal(t, []protocol.DocumentLink{}, links)
			} else {
				link := protocol.DocumentLink{
					Range:   tc.linkRange,
					Target:  types.CreateStringPointer(fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(tc.path), "/"))),
					Tooltip: types.CreateStringPointer(filepath.FromSlash(tc.path)),
				}
				require.Equal(t, []protocol.DocumentLink{link}, links)
			}
		})
	}
}

func TestDocumentLink_ServiceLabelFileLinks(t *testing.T) {
	testsFolder := filepath.Join(os.TempDir(), t.Name())
	composeStringURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(testsFolder, "compose.yaml")), "/"))

	testCases := []struct {
		name      string
		content   string
		path      string
		linkRange protocol.Range
	}{
		{
			name: "string value app.labels",
			content: `
services:
  test:
    label_file: app.labels`,
			path: filepath.Join(testsFolder, "./app.labels"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 16},
				End:   protocol.Position{Line: 3, Character: 26},
			},
		},
		{
			name: "string value ./app.labels",
			content: `
services:
  test:
    label_file: ./app.labels`,
			path: filepath.Join(testsFolder, "./app.labels"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 16},
				End:   protocol.Position{Line: 3, Character: 28},
			},
		},
		{
			name: "quoted string value \"./app.labels\"",
			content: `
services:
  test:
    label_file: "./app.labels"`,
			path: filepath.Join(testsFolder, "./app.labels"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 17},
				End:   protocol.Position{Line: 3, Character: 29},
			},
		},
		{
			name: "single quoted string value './app.labels'",
			content: `
services:
  test:
    label_file: './app.labels'`,
			path: filepath.Join(testsFolder, "./app.labels"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 17},
				End:   protocol.Position{Line: 3, Character: 29},
			},
		},
		{
			name: "attribute value is null",
			content: `
services:
  test:
    label_file: null`,
		},
		{
			name: "array items",
			content: `
services:
  test:
    label_file:
      - ./app.labels`,
			path: filepath.Join(testsFolder, "./app.labels"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 8},
				End:   protocol.Position{Line: 4, Character: 20},
			},
		},
		{
			name: "array item is null",
			content: `
services:
  test:
    label_file:
      - null`,
		},
		{
			name: "anchors and aliases to nothing",
			content: `
services:
  test:
    label_file: &anchor
  test2:
    label_file: *anchor`,
		},
		{
			name: "anchor on the services object itself",
			content: `
&anchor services:
  test:
    label_file: ./app.labels`,
			path: filepath.Join(testsFolder, "./app.labels"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 16},
				End:   protocol.Position{Line: 3, Character: 28},
			},
		},
		{
			name: "anchor on the services object's value",
			content: `
services: &anchor
  test:
    label_file: ./app.labels`,
			path: filepath.Join(testsFolder, "./app.labels"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 16},
				End:   protocol.Position{Line: 3, Character: 28},
			},
		},
		{
			name: "anchor on the service object itself",
			content: `
services:
  &anchor test:
    label_file: ./app.labels`,
			path: filepath.Join(testsFolder, "./app.labels"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 16},
				End:   protocol.Position{Line: 3, Character: 28},
			},
		},
		{
			name: "anchor on the service object's value",
			content: `
services:
  test: &anchor
    label_file: ./app.labels`,
			path: filepath.Join(testsFolder, "./app.labels"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 16},
				End:   protocol.Position{Line: 3, Character: 28},
			},
		},
		{
			name: "anchor on the service object's value as JSON",
			content: `
services:
  test: &anchor { label_file: ./app.labels }`,
			path: filepath.Join(testsFolder, "./app.labels"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 30},
				End:   protocol.Position{Line: 2, Character: 42},
			},
		},
		{
			name: "anchor on the label_file string attribute itself",
			content: `
services:
  test: &anchor
    &anchor label_file: ./app.labels`,
			path: filepath.Join(testsFolder, "./app.labels"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 24},
				End:   protocol.Position{Line: 3, Character: 36},
			},
		},
		{
			name: "anchor on the label_file string attribute's value",
			content: `
services:
  test: &anchor
    label_file: &anchor ./app.labels`,
			path: filepath.Join(testsFolder, "./app.labels"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 24},
				End:   protocol.Position{Line: 3, Character: 36},
			},
		},
		{
			name: "anchor on the label_file array attribute's value",
			content: `
services:
  test: &anchor
    label_file: &anchor
      - ./app.labels`,
			path: filepath.Join(testsFolder, "./app.labels"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 8},
				End:   protocol.Position{Line: 4, Character: 20},
			},
		},
		{
			name: "anchor on the label_file array item's value",
			content: `
services:
  test: &anchor
    label_file:
      - &anchor ./app.labels`,
			path: filepath.Join(testsFolder, "./app.labels"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 16},
				End:   protocol.Position{Line: 4, Character: 28},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := document.NewDocumentManager()
			doc := document.NewComposeDocument(mgr, uri.URI(composeStringURI), 1, []byte(tc.content))
			links, err := DocumentLink(context.Background(), composeStringURI, doc)
			require.NoError(t, err)
			if tc.path == "" {
				require.Equal(t, []protocol.DocumentLink{}, links)
			} else {
				link := protocol.DocumentLink{
					Range:   tc.linkRange,
					Target:  types.CreateStringPointer(fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(tc.path), "/"))),
					Tooltip: types.CreateStringPointer(filepath.FromSlash(tc.path)),
				}
				require.Equal(t, []protocol.DocumentLink{link}, links)
			}
		})
	}
}

func TestDocumentLink_VolumeFileLinks(t *testing.T) {
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, "tempFile.txt")
	f, err := os.Create(tempFile)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, f.Close())
	})
	testsFolder := createFileStructure(t)
	composeStringURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(testsFolder, "compose.yaml")), "/"))

	testCases := []struct {
		name      string
		content   string
		path      string
		linkRange protocol.Range
	}{
		{
			name: "mount local file",
			content: `
services:
  test:
    volumes:
      - ./a.txt:/mount/a.txt`,
			path: filepath.Join(testsFolder, "./a.txt"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 8},
				End:   protocol.Position{Line: 4, Character: 15},
			},
		},
		{
			name: "mount local file (string in single quotes)",
			content: `
services:
  test:
    volumes:
      - './a.txt:/mount/a.txt'`,
			path: filepath.Join(testsFolder, "./a.txt"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 9},
				End:   protocol.Position{Line: 4, Character: 16},
			},
		},
		{
			name: "mount local file (string in double quotes)",
			content: `
services:
  test:
    volumes:
      - "./a.txt:/mount/a.txt"`,
			path: filepath.Join(testsFolder, "./a.txt"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 9},
				End:   protocol.Position{Line: 4, Character: 16},
			},
		},
		{
			name: "mount local file (string is anchored)",
			content: `
services:
  test:
    volumes:
      - &anchor ./a.txt:/mount/a.txt`,
			path: filepath.Join(testsFolder, "./a.txt"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 16},
				End:   protocol.Position{Line: 4, Character: 23},
			},
		},
		{
			name: "mount local file (volumes attribute name is anchored)",
			content: `
services:
  test:
    &anchor volumes:
      - ./a.txt:/mount/a.txt`,
			path: filepath.Join(testsFolder, "./a.txt"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 8},
				End:   protocol.Position{Line: 4, Character: 15},
			},
		},
		{
			name: "mount local file (volumes attribute value is anchored)",
			content: `
services:
  test:
    volumes: &anchor
      - ./a.txt:/mount/a.txt`,
			path: filepath.Join(testsFolder, "./a.txt"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 8},
				End:   protocol.Position{Line: 4, Character: 15},
			},
		},
		{
			name: "mount local folder",
			content: `
services:
  test:
    volumes:
      - ./folder:/mount/folder`,
		},
		{
			name: "absolute path",
			content: fmt.Sprintf(`
services:
  test:
    volumes:
      - %v:/mount/file.txt`, tempFile),
			path: tempFile,
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 8},
				End:   protocol.Position{Line: 4, Character: protocol.UInteger(8 + len(tempFile))},
			},
		},
		{
			name: "mount local file with a complex volume object",
			content: `
services:
  test:
    volumes:
      - type: bind
        source: ./a.txt
        target: /mnt/a.txt`,
			path: filepath.Join(testsFolder, "./a.txt"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 16},
				End:   protocol.Position{Line: 5, Character: 23},
			},
		},
		{
			name: "mount local file with a complex volume object with the type's attribute name anchored",
			content: `
services:
  test:
    volumes:
      - &anchor type: bind
        source: ./a.txt
        target: /mnt/a.txt`,
			path: filepath.Join(testsFolder, "./a.txt"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 16},
				End:   protocol.Position{Line: 5, Character: 23},
			},
		},
		{
			name: "mount local file with a complex volume object with the type's attribute value anchored",
			content: `
services:
  test:
    volumes:
      - type: &anchor bind
        source: ./a.txt
        target: /mnt/a.txt`,
			path: filepath.Join(testsFolder, "./a.txt"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 16},
				End:   protocol.Position{Line: 5, Character: 23},
			},
		},
		{
			name: "mount local file with a complex volume object with the source's attribute name anchored",
			content: `
services:
  test:
    volumes:
      - type: bind
        &anchor source: ./a.txt
        target: /mnt/a.txt`,
			path: filepath.Join(testsFolder, "./a.txt"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 24},
				End:   protocol.Position{Line: 5, Character: 31},
			},
		},
		{
			name: "mount local file with a complex volume object with the source's attribute value anchored",
			content: `
services:
  test:
    volumes:
      - type: bind
        source: &anchor ./a.txt
        target: /mnt/a.txt`,
			path: filepath.Join(testsFolder, "./a.txt"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 24},
				End:   protocol.Position{Line: 5, Character: 31},
			},
		},
		{
			name: "mount local folder with a complex volume object",
			content: `
services:
  test:
    volumes:
      - type: bind
        source: ./folder
        target: /mount/folder`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := document.NewDocumentManager()
			doc := document.NewComposeDocument(mgr, uri.URI(composeStringURI), 1, []byte(tc.content))
			links, err := DocumentLink(context.Background(), composeStringURI, doc)
			require.NoError(t, err)
			if tc.path == "" {
				require.Equal(t, []protocol.DocumentLink{}, links)
			} else {
				link := protocol.DocumentLink{
					Range:   tc.linkRange,
					Target:  types.CreateStringPointer(fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(tc.path), "/"))),
					Tooltip: types.CreateStringPointer(filepath.FromSlash(tc.path)),
				}
				require.Equal(t, []protocol.DocumentLink{link}, links)
			}
		})
	}
}

func TestDocumentLink_ConfigFileLinks(t *testing.T) {
	testsFolder := filepath.Join(os.TempDir(), t.Name())
	composeStringURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(testsFolder, "compose.yaml")), "/"))

	testCases := []struct {
		name      string
		content   string
		path      string
		linkRange protocol.Range
	}{
		{
			name: "./httpd.conf",
			content: `
configs:
  test:
    file: ./httpd.conf`,
			path: filepath.Join(testsFolder, "httpd.conf"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 10},
				End:   protocol.Position{Line: 3, Character: 22},
			},
		},
		{
			name: `"./httpd.conf"`,
			content: `
configs:
  test:
    file: "./httpd.conf"`,
			path: filepath.Join(testsFolder, "httpd.conf"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 11},
				End:   protocol.Position{Line: 3, Character: 23},
			},
		},
		{
			name: `'./httpd.conf'`,
			content: `
configs:
  test:
    file: './httpd.conf'`,
			path: filepath.Join(testsFolder, "httpd.conf"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 11},
				End:   protocol.Position{Line: 3, Character: 23},
			},
		},
		{
			name: "anchors and aliases to nothing",
			content: `
configs:
  test:
    file: &configFile
  test2:
    file: *configFile`,
		},
		{
			name: "anchor has string content",
			content: `
configs:
  test:
    file: &configFile ./httpd.conf
  test2:
    file: *configFile`,
			path: filepath.Join(testsFolder, "httpd.conf"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 22},
				End:   protocol.Position{Line: 3, Character: 34},
			},
		},
		{
			name: "anchor on the configs object itself",
			content: `
&anchor configs:
  test:
    file: ./httpd.conf`,
			path: filepath.Join(testsFolder, "httpd.conf"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 10},
				End:   protocol.Position{Line: 3, Character: 22},
			},
		},
		{
			name: "anchor on the configs object's value",
			content: `
configs: &anchor
  test:
    file: ./httpd.conf`,
			path: filepath.Join(testsFolder, "httpd.conf"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 10},
				End:   protocol.Position{Line: 3, Character: 22},
			},
		},
		{
			name: "anchor on the config object",
			content: `
configs:
  test: &anchor
    file: ./httpd.conf`,
			path: filepath.Join(testsFolder, "httpd.conf"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 10},
				End:   protocol.Position{Line: 3, Character: 22},
			},
		},
		{
			name: "anchor on the file attribute",
			content: `
configs:
  test:
    &anchor file: ./httpd.conf`,
			path: filepath.Join(testsFolder, "httpd.conf"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 18},
				End:   protocol.Position{Line: 3, Character: 30},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := document.NewDocumentManager()
			doc := document.NewComposeDocument(mgr, uri.URI(composeStringURI), 1, []byte(tc.content))
			links, err := DocumentLink(context.Background(), composeStringURI, doc)
			require.NoError(t, err)
			if tc.path == "" {
				require.Equal(t, []protocol.DocumentLink{}, links)
			} else {
				link := protocol.DocumentLink{
					Range:   tc.linkRange,
					Target:  types.CreateStringPointer(fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(tc.path), "/"))),
					Tooltip: types.CreateStringPointer(filepath.FromSlash(tc.path)),
				}
				require.Equal(t, []protocol.DocumentLink{link}, links)
			}
		})
	}
}

func TestDocumentLink_SecretFileLinks(t *testing.T) {
	testsFolder := filepath.Join(os.TempDir(), t.Name())
	composeStringURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(testsFolder, "compose.yaml")), "/"))

	testCases := []struct {
		name      string
		content   string
		path      string
		linkRange protocol.Range
	}{
		{
			name: "./server.cert",
			content: `
secrets:
  test:
    file: ./server.cert`,
			path: filepath.Join(testsFolder, "server.cert"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 10},
				End:   protocol.Position{Line: 3, Character: 23},
			},
		},
		{
			name: `"./server.cert"`,
			content: `
secrets:
  test:
    file: "./server.cert"`,
			path: filepath.Join(testsFolder, "server.cert"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 11},
				End:   protocol.Position{Line: 3, Character: 24},
			},
		},
		{
			name: `'./server.cert'`,
			content: `
secrets:
  test:
    file: './server.cert'`,
			path: filepath.Join(testsFolder, "server.cert"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 11},
				End:   protocol.Position{Line: 3, Character: 24},
			},
		},
		{
			name: "anchors and aliases to nothing",
			content: `
secrets:
  test:
    file: &configFile
  test2:
    file: *configFile`,
		},
		{
			name: "anchor has string content",
			content: `
secrets:
  test:
    file: &configFile ./server.cert
  test2:
    file: *configFile`,
			path: filepath.Join(testsFolder, "server.cert"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 22},
				End:   protocol.Position{Line: 3, Character: 35},
			},
		},
		{
			name: "anchor on the secrets object itself",
			content: `
&anchor secrets:
  test:
    file: ./server.cert`,
			path: filepath.Join(testsFolder, "server.cert"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 10},
				End:   protocol.Position{Line: 3, Character: 23},
			},
		},
		{
			name: "anchor on the secrets object's value",
			content: `
secrets: &anchor
  test:
    file: ./server.cert`,
			path: filepath.Join(testsFolder, "server.cert"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 10},
				End:   protocol.Position{Line: 3, Character: 23},
			},
		},
		{
			name: "anchor on the secret object",
			content: `
secrets:
  test: &anchor
    file: ./server.cert`,
			path: filepath.Join(testsFolder, "server.cert"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 10},
				End:   protocol.Position{Line: 3, Character: 23},
			},
		},
		{
			name: "anchor on the file attribute",
			content: `
secrets:
  test:
    &anchor file: ./server.cert`,
			path: filepath.Join(testsFolder, "server.cert"),
			linkRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 18},
				End:   protocol.Position{Line: 3, Character: 31},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := document.NewDocumentManager()
			doc := document.NewComposeDocument(mgr, uri.URI(composeStringURI), 1, []byte(tc.content))
			links, err := DocumentLink(context.Background(), composeStringURI, doc)
			require.NoError(t, err)
			if tc.path == "" {
				require.Equal(t, []protocol.DocumentLink{}, links)
			} else {
				link := protocol.DocumentLink{
					Range:   tc.linkRange,
					Target:  types.CreateStringPointer(fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(tc.path), "/"))),
					Tooltip: types.CreateStringPointer(filepath.FromSlash(tc.path)),
				}
				require.Equal(t, []protocol.DocumentLink{link}, links)
			}
		})
	}
}

func TestDocumentLink_ModelsModelLinks(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		links   []protocol.DocumentLink
	}{
		{
			name: "ai/llama3.3",
			content: `
models:
  modelA:
    model: ai/llama3.3`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 22},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/r/ai/llama3.3"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/r/ai/llama3.3"),
				},
			},
		},
		{
			name: "ai/llama3.3:latest",
			content: `
models:
  modelA:
    model: ai/llama3.3:latest`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 22},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/r/ai/llama3.3"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/r/ai/llama3.3"),
				},
			},
		},
		{
			name: "\"ai/llama3.3\"",
			content: `
models:
  modelA:
    model: "ai/llama3.3"`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 12},
						End:   protocol.Position{Line: 3, Character: 23},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/r/ai/llama3.3"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/r/ai/llama3.3"),
				},
			},
		},
		{
			name: "'ai/llama3.3'",
			content: `
models:
  modelA:
    model: 'ai/llama3.3'`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 12},
						End:   protocol.Position{Line: 3, Character: 23},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/r/ai/llama3.3"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/r/ai/llama3.3"),
				},
			},
		},
		{
			name: "\"ai/llama3.3:\"",
			content: `
models:
  modelA:
    model: "ai/llama3.3:"`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 12},
						End:   protocol.Position{Line: 3, Character: 23},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/r/ai/llama3.3"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/r/ai/llama3.3"),
				},
			},
		},
		{
			name: "'ai/llama3.3:'",
			content: `
models:
  modelA:
    model: 'ai/llama3.3:'`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 12},
						End:   protocol.Position{Line: 3, Character: 23},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/r/ai/llama3.3"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/r/ai/llama3.3"),
				},
			},
		},
		{
			name: "hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF",
			content: `
models:
  modelA:
    model: hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 53},
					},
					Target:  types.CreateStringPointer("https://hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF"),
					Tooltip: types.CreateStringPointer("https://hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF"),
				},
			},
		},
		{
			name: "hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF:latest",
			content: `
models:
  modelA:
    model: hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF:latest`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 53},
					},
					Target:  types.CreateStringPointer("https://hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF"),
					Tooltip: types.CreateStringPointer("https://hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF"),
				},
			},
		},
		{
			name: "\"hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF\"",
			content: `
models:
  modelA:
    model: "hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF"`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 12},
						End:   protocol.Position{Line: 3, Character: 54},
					},
					Target:  types.CreateStringPointer("https://hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF"),
					Tooltip: types.CreateStringPointer("https://hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF"),
				},
			},
		},
		{
			name: "'hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF'",
			content: `
models:
  modelA:
    model: 'hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF'`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 12},
						End:   protocol.Position{Line: 3, Character: 54},
					},
					Target:  types.CreateStringPointer("https://hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF"),
					Tooltip: types.CreateStringPointer("https://hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF"),
				},
			},
		},
		{
			name: "\"hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF:\"",
			content: `
models:
  modelA:
    model: "hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF:"`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 12},
						End:   protocol.Position{Line: 3, Character: 54},
					},
					Target:  types.CreateStringPointer("https://hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF"),
					Tooltip: types.CreateStringPointer("https://hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF"),
				},
			},
		},
		{
			name: "'hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF:'",
			content: `
models:
  modelA:
    model: 'hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF:'`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 12},
						End:   protocol.Position{Line: 3, Character: 54},
					},
					Target:  types.CreateStringPointer("https://hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF"),
					Tooltip: types.CreateStringPointer("https://hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF"),
				},
			},
		},
		{
			name: "hf.co",
			content: `
models:
  modelA:
    model: hf.co`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "\"hf.co:\"",
			content: `
models:
  modelA:
    model: "hf.co:"`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "'hf.co:'",
			content: `
models:
  modelA:
    model: 'hf.co:'`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "anchors and aliases to nothing",
			content: `
models:
  model1:
    model: &aiModelHello
  model2:
    model: *aiModelHello`,
			links: []protocol.DocumentLink{},
		},
		{
			name: "anchor on the models object itself",
			content: `
&anchor models:
  model1:
    model: ai/qwen3`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 19},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/r/ai/qwen3"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/r/ai/qwen3"),
				},
			},
		},
		{
			name: "anchor on the models object's value",
			content: `
models: &anchor
  model1:
    model: ai/qwen3`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 19},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/r/ai/qwen3"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/r/ai/qwen3"),
				},
			},
		},
		{
			name: "anchor on the model object itself",
			content: `
models:
  &anchor model1:
    model: ai/qwen3`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 19},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/r/ai/qwen3"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/r/ai/qwen3"),
				},
			},
		},
		{
			name: "anchor on the model object's value",
			content: `
models:
  model1: &anchor
    model: ai/qwen3`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 11},
						End:   protocol.Position{Line: 3, Character: 19},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/r/ai/qwen3"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/r/ai/qwen3"),
				},
			},
		},
		{
			name: "anchor on the model object's value as JSON",
			content: `
models:
  model1: &anchor { model: ai/qwen3 }`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 27},
						End:   protocol.Position{Line: 2, Character: 35},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/r/ai/qwen3"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/r/ai/qwen3"),
				},
			},
		},
		{
			name: "anchor on the model attribute's value",
			content: `
models:
  model1:
    &anchor model: ai/qwen3`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 19},
						End:   protocol.Position{Line: 3, Character: 27},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/r/ai/qwen3"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/r/ai/qwen3"),
				},
			},
		},
		{
			name: "anchor on the model attribute's value",
			content: `
models:
  model1:
    model: &anchor ai/qwen3`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 19},
						End:   protocol.Position{Line: 3, Character: 27},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/r/ai/qwen3"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/r/ai/qwen3"),
				},
			},
		},
		{
			name: "invalid models",
			content: `
models:
  - `,
			links: []protocol.DocumentLink{},
		},
		{
			name: "two documents",
			content: `
---
models:
  model1:
    model: ai/smollm2
---
models:
  model2:
    model: ai/smollm3`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 4, Character: 11},
						End:   protocol.Position{Line: 4, Character: 21},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/r/ai/smollm2"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/r/ai/smollm2"),
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 8, Character: 11},
						End:   protocol.Position{Line: 8, Character: 21},
					},
					Target:  types.CreateStringPointer("https://hub.docker.com/r/ai/smollm3"),
					Tooltip: types.CreateStringPointer("https://hub.docker.com/r/ai/smollm3"),
				},
			},
		},
	}

	composeStringURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(os.TempDir(), "compose.yaml")), "/"))
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := document.NewDocumentManager()
			doc := document.NewComposeDocument(mgr, "docker-compose.yml", 1, []byte(tc.content))
			links, err := DocumentLink(context.Background(), composeStringURI, doc)
			require.NoError(t, err)
			require.Equal(t, tc.links, links)
		})
	}
}
