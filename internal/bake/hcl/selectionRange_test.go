package hcl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/uri"
)

func TestSelectionRange(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		positions []protocol.Position
		expected  []protocol.SelectionRange
	}{
		{
			name:    "simple target block - block type",
			content: "target \"webapp\" {\n  context = \".\"\n}",
			positions: []protocol.Position{
				{Line: 0, Character: 2}, // Inside "target"
			},
			expected: []protocol.SelectionRange{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 0},
						End:   protocol.Position{Line: 0, Character: 6},
					},
					Parent: &protocol.SelectionRange{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 17},
							End:   protocol.Position{Line: 2, Character: 1},
						},
						Parent: &protocol.SelectionRange{
							Range: protocol.Range{
								Start: protocol.Position{Line: 0, Character: 0},
								End:   protocol.Position{Line: 2, Character: 1},
							},
						},
					},
				},
			},
		},
		{
			name:    "simple target block - block label",
			content: "target \"webapp\" {\n  context = \".\"\n}",
			positions: []protocol.Position{
				{Line: 0, Character: 10}, // Inside "webapp"
			},
			expected: []protocol.SelectionRange{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 7},
						End:   protocol.Position{Line: 0, Character: 15},
					},
					Parent: &protocol.SelectionRange{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 17},
							End:   protocol.Position{Line: 2, Character: 1},
						},
						Parent: &protocol.SelectionRange{
							Range: protocol.Range{
								Start: protocol.Position{Line: 0, Character: 0},
								End:   protocol.Position{Line: 2, Character: 1},
							},
						},
					},
				},
			},
		},
		{
			name:    "attribute name",
			content: "target \"webapp\" {\n  context = \".\"\n}",
			positions: []protocol.Position{
				{Line: 1, Character: 4}, // Inside "context"
			},
			expected: []protocol.SelectionRange{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 2},
						End:   protocol.Position{Line: 1, Character: 9},
					},
					Parent: &protocol.SelectionRange{
						Range: protocol.Range{
							Start: protocol.Position{Line: 1, Character: 2},
							End:   protocol.Position{Line: 1, Character: 13},
						},
						Parent: &protocol.SelectionRange{
							Range: protocol.Range{
								Start: protocol.Position{Line: 0, Character: 17},
								End:   protocol.Position{Line: 2, Character: 1},
							},
							Parent: &protocol.SelectionRange{
								Range: protocol.Range{
									Start: protocol.Position{Line: 0, Character: 0},
									End:   protocol.Position{Line: 2, Character: 1},
								},
							},
						},
					},
				},
			},
		},
		{
			name:    "attribute value",
			content: "target \"webapp\" {\n  context = \".\"\n}",
			positions: []protocol.Position{
				{Line: 1, Character: 12}, // Inside "."
			},
			expected: []protocol.SelectionRange{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 12},
						End:   protocol.Position{Line: 1, Character: 13},
					},
					Parent: &protocol.SelectionRange{
						Range: protocol.Range{
							Start: protocol.Position{Line: 1, Character: 12},
							End:   protocol.Position{Line: 1, Character: 13},
						},
						Parent: &protocol.SelectionRange{
							Range: protocol.Range{
								Start: protocol.Position{Line: 1, Character: 2},
								End:   protocol.Position{Line: 1, Character: 13},
							},
							Parent: &protocol.SelectionRange{
								Range: protocol.Range{
									Start: protocol.Position{Line: 0, Character: 17},
									End:   protocol.Position{Line: 2, Character: 1},
								},
								Parent: &protocol.SelectionRange{
									Range: protocol.Range{
										Start: protocol.Position{Line: 0, Character: 0},
										End:   protocol.Position{Line: 2, Character: 1},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:    "array attribute",
			content: "target \"webapp\" {\n  tags = [\"latest\", \"v1.0\"]\n}",
			positions: []protocol.Position{
				{Line: 1, Character: 13}, // Inside "latest"
			},
			expected: []protocol.SelectionRange{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 10},
						End:   protocol.Position{Line: 1, Character: 18},
					},
					Parent: &protocol.SelectionRange{
						Range: protocol.Range{
							Start: protocol.Position{Line: 1, Character: 9},
							End:   protocol.Position{Line: 1, Character: 25},
						},
						Parent: &protocol.SelectionRange{
							Range: protocol.Range{
								Start: protocol.Position{Line: 1, Character: 2},
								End:   protocol.Position{Line: 1, Character: 25},
							},
							Parent: &protocol.SelectionRange{
								Range: protocol.Range{
									Start: protocol.Position{Line: 0, Character: 17},
									End:   protocol.Position{Line: 2, Character: 1},
								},
								Parent: &protocol.SelectionRange{
									Range: protocol.Range{
										Start: protocol.Position{Line: 0, Character: 0},
										End:   protocol.Position{Line: 2, Character: 1},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:    "object attribute key",
			content: "target \"webapp\" {\n  args = {\n    VAR = \"value\"\n  }\n}",
			positions: []protocol.Position{
				{Line: 2, Character: 6}, // Inside "VAR"
			},
			expected: []protocol.SelectionRange{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 4},
						End:   protocol.Position{Line: 2, Character: 7},
					},
					Parent: &protocol.SelectionRange{
						Range: protocol.Range{
							Start: protocol.Position{Line: 1, Character: 9},
							End:   protocol.Position{Line: 3, Character: 3},
						},
						Parent: &protocol.SelectionRange{
							Range: protocol.Range{
								Start: protocol.Position{Line: 1, Character: 2},
								End:   protocol.Position{Line: 3, Character: 3},
							},
							Parent: &protocol.SelectionRange{
								Range: protocol.Range{
									Start: protocol.Position{Line: 0, Character: 17},
									End:   protocol.Position{Line: 4, Character: 1},
								},
								Parent: &protocol.SelectionRange{
									Range: protocol.Range{
										Start: protocol.Position{Line: 0, Character: 0},
										End:   protocol.Position{Line: 4, Character: 1},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:    "function call",
			content: "target \"webapp\" {\n  tags = [upper(\"tag\")]\n}",
			positions: []protocol.Position{
				{Line: 1, Character: 11}, // Inside "upper"
			},
			expected: []protocol.SelectionRange{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 10},
						End:   protocol.Position{Line: 1, Character: 15},
					},
					Parent: &protocol.SelectionRange{
						Range: protocol.Range{
							Start: protocol.Position{Line: 1, Character: 10},
							End:   protocol.Position{Line: 1, Character: 21},
						},
						Parent: &protocol.SelectionRange{
							Range: protocol.Range{
								Start: protocol.Position{Line: 1, Character: 9},
								End:   protocol.Position{Line: 1, Character: 22},
							},
							Parent: &protocol.SelectionRange{
								Range: protocol.Range{
									Start: protocol.Position{Line: 1, Character: 2},
									End:   protocol.Position{Line: 1, Character: 22},
								},
								Parent: &protocol.SelectionRange{
									Range: protocol.Range{
										Start: protocol.Position{Line: 0, Character: 17},
										End:   protocol.Position{Line: 2, Character: 1},
									},
									Parent: &protocol.SelectionRange{
										Range: protocol.Range{
											Start: protocol.Position{Line: 0, Character: 0},
											End:   protocol.Position{Line: 2, Character: 1},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	temporaryBakeFile := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(os.TempDir(), "docker-bake.hcl")), "/"))
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := document.NewBakeHCLDocument(uri.URI(temporaryBakeFile), 1, []byte(tc.content))
			result, err := SelectionRange(doc, tc.positions)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}
