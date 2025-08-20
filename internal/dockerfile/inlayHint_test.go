package dockerfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker-language-server/internal/hub"
	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/uri"
)

func TestInlayHint(t *testing.T) {
	// set timezone to UTC for this test so the locale is consistent
	origTZ := os.Getenv("TZ")
	require.NoError(t, os.Setenv("TZ", "UTC"))
	t.Cleanup(func() {
		require.NoError(t, os.Setenv("TZ", origTZ))
	})

	testCases := []struct {
		name       string
		content    string
		rng        protocol.Range
		inlayHints []protocol.InlayHint
	}{
		{
			name:    "alpine",
			content: "FROM alpine",
			rng: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 11},
			},
			inlayHints: []protocol.InlayHint{},
		},
		{
			name:    "alpine:3.16",
			content: "FROM alpine:3.16",
			rng: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 16},
			},
			inlayHints: []protocol.InlayHint{
				{
					Label:       "(last pushed 1 year ago)",
					PaddingLeft: types.CreateBoolPointer(true),
					Position:    protocol.Position{Line: 0, Character: 16},
					Tooltip:     types.CreateAnyPointer("2024-01-27 00:47:58 UTC"),
				},
			},
		},
		{
			name:    "alpine@sha256:72af6266bafde8c78d5f20a2a85d0576533ce1ecd6ed8bcf7baf62a743f3b24d",
			content: "FROM alpine@sha256:72af6266bafde8c78d5f20a2a85d0576533ce1ecd6ed8bcf7baf62a743f3b24d",
			rng: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 16},
			},
			inlayHints: []protocol.InlayHint{},
		},
		{
			name:    "alpine:3.16@sha256:72af6266bafde8c78d5f20a2a85d0576533ce1ecd6ed8bcf7baf62a743f3b24d",
			content: "FROM alpine:3.16@sha256:72af6266bafde8c78d5f20a2a85d0576533ce1ecd6ed8bcf7baf62a743f3b24d",
			rng: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 16},
			},
			inlayHints: []protocol.InlayHint{},
		},
		{
			name:    "prom/prometheus",
			content: "FROM prom/prometheus",
			rng: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 16},
			},
			inlayHints: []protocol.InlayHint{},
		},
		{
			name:    "prom/prometheus:v2.6.1",
			content: "FROM prom/prometheus:v2.6.1",
			rng: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 27},
			},
			inlayHints: []protocol.InlayHint{
				{
					Label:       "(last pushed 6 years ago)",
					PaddingLeft: types.CreateBoolPointer(true),
					Position:    protocol.Position{Line: 0, Character: 27},
					Tooltip:     types.CreateAnyPointer("2019-01-15 20:13:35 UTC"),
				},
			},
		},
		{
			name:    "content outside range should not return anything",
			content: "\n\nFROM alpine:3.16",
			rng: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 1, Character: 0},
			},
			inlayHints: []protocol.InlayHint{},
		},
	}

	dockerfileURI := uri.URI(fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(os.TempDir(), "Dockerfile")), "/")))
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hubService := hub.NewService()
			doc := document.NewDockerfileDocument(dockerfileURI, 1, []byte(tc.content))
			inlayHints, err := InlayHint(hubService, doc, tc.rng)
			require.NoError(t, err)
			require.Equal(t, tc.inlayHints, inlayHints)
		})
	}
}
