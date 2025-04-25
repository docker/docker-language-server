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
)

func TestDocumentLink(t *testing.T) {
	testsFolder := filepath.Join(os.TempDir(), "composeDocumentLinkTests")
	composeFilePath := filepath.Join(testsFolder, "docker-compose.yml")
	referencedFilePath := filepath.Join(testsFolder, "file.yml")
	composeStringURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(composeFilePath), "/"))
	referencedFileStringURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(referencedFilePath), "/"))

	testCases := []struct {
		name    string
		content string
		links   []protocol.DocumentLink
	}{
		{
			name: "included files, short syntax",
			content: `include:
  - file.yml`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 4},
						End:   protocol.Position{Line: 1, Character: 12},
					},
					Target:  types.CreateStringPointer(referencedFileStringURI),
					Tooltip: types.CreateStringPointer(referencedFilePath),
				},
			},
		},
		{
			name: "dockerfile attribute in build section",
			content: `services:
  backend:
    build:
      context: ./backend
      dockerfile: ../backend.Dockerfile`,
			links: []protocol.DocumentLink{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 4, Character: 18},
						End:   protocol.Position{Line: 4, Character: 38},
					},
					Target:  types.CreateStringPointer(fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(testsFolder, "../backend.Dockerfile")), "/"))),
					Tooltip: types.CreateStringPointer(filepath.Join(testsFolder, "../backend.Dockerfile")),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := document.NewComposeDocument("docker-compose.yml", 1, []byte(tc.content))
			links, err := DocumentLink(context.Background(), composeStringURI, doc)
			require.NoError(t, err)
			require.Equal(t, tc.links, links)
		})
	}
}
