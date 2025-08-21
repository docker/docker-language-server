package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/uri"
)

func documentHighlight(startLine, startCharacter, endLine, endCharacter protocol.UInteger, kind protocol.DocumentHighlightKind) protocol.DocumentHighlight {
	return protocol.DocumentHighlight{
		Kind: &kind,
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      startLine,
				Character: startCharacter,
			},
			End: protocol.Position{
				Line:      endLine,
				Character: endCharacter,
			},
		},
	}
}

var serviceReferenceTestCases = []struct {
	name          string
	content       string
	line          protocol.UInteger
	character     protocol.UInteger
	locations     func(protocol.DocumentUri) any
	links         func(protocol.DocumentUri) any
	ranges        []protocol.DocumentHighlight
	renameEdits   func(protocol.DocumentUri) *protocol.WorkspaceEdit
	prepareRename *protocol.Range
}{
	{
		name: "write highlight on a service",
		content: `
services:
  test:`,
		line:      2,
		character: 4,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 2},
			End:   protocol.Position{Line: 2, Character: 6},
		},
	},
	{
		name: "write highlight on a services node itself anchored",
		content: `
&anchor services:
  test:`,
		line:      2,
		character: 4,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 2},
			End:   protocol.Position{Line: 2, Character: 6},
		},
	},
	{
		name: "write highlight on a services node's value anchored",
		content: `
services: &anchor
  test:`,
		line:      2,
		character: 4,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 2},
			End:   protocol.Position{Line: 2, Character: 6},
		},
	},
	{
		name: "read highlight on an undefined service's depends_on array string",
		content: `
services:
  test:
    depends_on:
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read highlight on an undefined service's depends_on array string with depends_on itself anchored",
		content: `
services:
  test:
    &anchor depends_on:
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read highlight on an undefined service's depends_on array string with depends_on's value anchored",
		content: `
services:
  test:
    depends_on: &anchor
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read highlight on an undefined service's depends_on array string with depends_on itself anchored",
		content: `
services:
  test:
    &anchor depends_on:
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read highlight on an undefined service's depends_on array string with an anchored value",
		content: `
services:
  test:
    depends_on:
      - &anchor test2`,
		line:      4,
		character: 18,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 16, 4, 21, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 16},
								End:   protocol.Position{Line: 4, Character: 21},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 16},
			End:   protocol.Position{Line: 4, Character: 21},
		},
	},
	{
		name: "read highlight on an undefined quoted service's depends_on array string",
		content: `
services:
  test:
    depends_on:
      - "test2"`,
		line:      4,
		character: 12,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 9, 4, 14, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 9},
								End:   protocol.Position{Line: 4, Character: 14},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 9},
			End:   protocol.Position{Line: 4, Character: 14},
		},
	},
	{
		name: "read highlight on a defined quoted service's depends_on array string",
		content: `
services:
  test:
    depends_on:
      - "test2"
  test2:
    image: redis`,
		line:      4,
		character: 12,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 5, Character: 2},
				End:   protocol.Position{Line: 5, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 5, Character: 2},
				End:   protocol.Position{Line: 5, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 9},
				End:   protocol.Position{Line: 4, Character: 14},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 9, 4, 14, protocol.DocumentHighlightKindRead),
			documentHighlight(5, 2, 5, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 9},
								End:   protocol.Position{Line: 4, Character: 14},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 2},
								End:   protocol.Position{Line: 5, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 9},
			End:   protocol.Position{Line: 4, Character: 14},
		},
	},
	{
		name: "read highlight on a depends_on string service with the declaration anchored",
		content: `
services:
  test:
    depends_on:
      - test2
  &anchor test2:
    image: redis`,
		line:      4,
		character: 12,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 5, Character: 10},
				End:   protocol.Position{Line: 5, Character: 15},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 5, Character: 10},
				End:   protocol.Position{Line: 5, Character: 15},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 8},
				End:   protocol.Position{Line: 4, Character: 13},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(5, 10, 5, 15, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 10},
								End:   protocol.Position{Line: 5, Character: 15},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read highlight on an undefined service object with no properties",
		content: `
services:
  test:
    depends_on:
      test2:`,
		line:      4,
		character: 9,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 6, 4, 11, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 6},
								End:   protocol.Position{Line: 4, Character: 11},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 6},
			End:   protocol.Position{Line: 4, Character: 11},
		},
	},
	{
		name: "read highlight on an undefined service object with no properties with the object value anchored",
		content: `
services:
  test:
    depends_on: &anchor
      test2:`,
		line:      4,
		character: 9,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 6, 4, 11, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 6},
								End:   protocol.Position{Line: 4, Character: 11},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 6},
			End:   protocol.Position{Line: 4, Character: 11},
		},
	},
	{
		name: "read highlight on an undefined service object with no properties with the dependency's name anchored",
		content: `
services:
  test:
    depends_on: { &anchor test2: null }`,
		line:      3,
		character: 28,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(3, 26, 3, 31, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 3, Character: 26},
								End:   protocol.Position{Line: 3, Character: 31},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 3, Character: 26},
			End:   protocol.Position{Line: 3, Character: 31},
		},
	},
	{
		name: "read highlight on an undefined service object with properties",
		content: `
services:
  test:
    depends_on:
      test2:
        condition: service_started`,
		line:      4,
		character: 9,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 6, 4, 11, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 6},
								End:   protocol.Position{Line: 4, Character: 11},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 6},
			End:   protocol.Position{Line: 4, Character: 11},
		},
	},
	{
		name: "cursor not on anything meaningful",
		content: `
services:
  test:
    depends_on:
      - test2
      - test2`,
		line:      3,
		character: 9,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges:    nil,
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return nil
		},
		prepareRename: nil,
	},
	{
		name: "read highlight on an undefined service's depends_on array string, duplicated",
		content: `
services:
  test:
    depends_on:
      - test2
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(5, 8, 5, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 8},
								End:   protocol.Position{Line: 5, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read/write highlight on a service's depends_on array string (cursor on read)",
		content: `
services:
  test:
    depends_on:
      - test2
  test2:`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 5, Character: 2},
				End:   protocol.Position{Line: 5, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 5, Character: 2},
				End:   protocol.Position{Line: 5, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 8},
				End:   protocol.Position{Line: 4, Character: 13},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(5, 2, 5, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 2},
								End:   protocol.Position{Line: 5, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read/write highlight on a service's depends_on array string (cursor on write)",
		content: `
services:
  test:
    depends_on:
      - test2
  test2:`,
		line:      5,
		character: 5,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 5, Character: 2},
				End:   protocol.Position{Line: 5, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 5, Character: 2},
				End:   protocol.Position{Line: 5, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 2},
				End:   protocol.Position{Line: 5, Character: 7},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(5, 2, 5, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 2},
								End:   protocol.Position{Line: 5, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 5, Character: 2},
			End:   protocol.Position{Line: 5, Character: 7},
		},
	},
	{
		name: "short syntax form of depends_on in services finding the right match",
		content: `
services:
  web:
    build: .
    depends_on:
      - postgres
      - redis
  postgres:
    image: postgres
  redis:
    image: redis`,
		line:      6,
		character: 11,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 9, Character: 2},
				End:   protocol.Position{Line: 9, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 9, Character: 2},
				End:   protocol.Position{Line: 9, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 6, Character: 8},
				End:   protocol.Position{Line: 6, Character: 13},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(6, 8, 6, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(9, 2, 9, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 8},
								End:   protocol.Position{Line: 6, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 9, Character: 2},
								End:   protocol.Position{Line: 9, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 6, Character: 8},
			End:   protocol.Position{Line: 6, Character: 13},
		},
	},
	{
		name: "long syntax form of depends_on in services",
		content: `
services:
  web:
    build: .
    depends_on:
      db:
        condition: service_healthy
        restart: true
      redis:
        condition: service_started
  db:
    image: postgres
  redis:
    image: redis`,
		line:      8,
		character: 9,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 12, Character: 2},
				End:   protocol.Position{Line: 12, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 12, Character: 2},
				End:   protocol.Position{Line: 12, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 8, Character: 6},
				End:   protocol.Position{Line: 8, Character: 11},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(8, 6, 8, 11, protocol.DocumentHighlightKindRead),
			documentHighlight(12, 2, 12, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 8, Character: 6},
								End:   protocol.Position{Line: 8, Character: 11},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 12, Character: 2},
								End:   protocol.Position{Line: 12, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 8, Character: 6},
			End:   protocol.Position{Line: 8, Character: 11},
		},
	},
	{
		name: "extends as a string attribute",
		content: `
services:
  test:
    image: alpine
  test2:
    extends: test`,
		line:      5,
		character: 15,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 13},
				End:   protocol.Position{Line: 5, Character: 17},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
			documentHighlight(5, 13, 5, 17, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 13},
								End:   protocol.Position{Line: 5, Character: 17},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 5, Character: 13},
			End:   protocol.Position{Line: 5, Character: 17},
		},
	},
	{
		name: "extends (with an anchor) as a string attribute value",
		content: `
services:
  test:
    image: alpine
  test2:
    &anchor extends: test`,
		line:      5,
		character: 23,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 21},
				End:   protocol.Position{Line: 5, Character: 25},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
			documentHighlight(5, 21, 5, 25, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 21},
								End:   protocol.Position{Line: 5, Character: 25},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 5, Character: 21},
			End:   protocol.Position{Line: 5, Character: 25},
		},
	},
	{
		name: "extends as a string attribute value with an anchor",
		content: `
services:
  test:
    image: alpine
  test2:
    extends: &anchor test`,
		line:      5,
		character: 23,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 21},
				End:   protocol.Position{Line: 5, Character: 25},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
			documentHighlight(5, 21, 5, 25, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 21},
								End:   protocol.Position{Line: 5, Character: 25},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 5, Character: 21},
			End:   protocol.Position{Line: 5, Character: 25},
		},
	},
	{
		name: "extends as a quoted string attribute",
		content: `
services:
  test:
    image: alpine
  test2:
    extends: "test"`,
		line:      5,
		character: 15,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 14},
				End:   protocol.Position{Line: 5, Character: 18},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
			documentHighlight(5, 14, 5, 18, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 14},
								End:   protocol.Position{Line: 5, Character: 18},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 5, Character: 14},
			End:   protocol.Position{Line: 5, Character: 18},
		},
	},
	{
		name: "extends as an object without a file attribute",
		content: `
services:
  test:
    image: alpine
  test2:
    extends:
      service: test`,
		line:      6,
		character: 17,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 6, Character: 15},
				End:   protocol.Position{Line: 6, Character: 19},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
			documentHighlight(6, 15, 6, 19, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 15},
								End:   protocol.Position{Line: 6, Character: 19},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 6, Character: 15},
			End:   protocol.Position{Line: 6, Character: 19},
		},
	},
	{
		name: "extends as an object with its value as an anchor",
		content: `
services:
  test:
    image: alpine
  test2:
    extends: &anchor
      service: test`,
		line:      6,
		character: 17,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 6, Character: 15},
				End:   protocol.Position{Line: 6, Character: 19},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
			documentHighlight(6, 15, 6, 19, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 15},
								End:   protocol.Position{Line: 6, Character: 19},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 6, Character: 15},
			End:   protocol.Position{Line: 6, Character: 19},
		},
	},
	{
		name: "extends as an object without a file attribute with an anchor on the service attribute's name",
		content: `
services:
  test:
    image: alpine
  test2:
    extends:
      &anchor service: test`,
		line:      6,
		character: 25,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 6, Character: 23},
				End:   protocol.Position{Line: 6, Character: 27},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
			documentHighlight(6, 23, 6, 27, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 23},
								End:   protocol.Position{Line: 6, Character: 27},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 6, Character: 23},
			End:   protocol.Position{Line: 6, Character: 27},
		},
	},
	{
		name: "extends as an object without a file attribute with an anchor on the service attribute's value",
		content: `
services:
  test:
    image: alpine
  test2:
    extends:
      service: &anchor test`,
		line:      6,
		character: 25,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 6, Character: 23},
				End:   protocol.Position{Line: 6, Character: 27},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
			documentHighlight(6, 23, 6, 27, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 23},
								End:   protocol.Position{Line: 6, Character: 27},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 6, Character: 23},
			End:   protocol.Position{Line: 6, Character: 27},
		},
	},
	{
		name: "extends as an object with a file attribute that points to a non-existent file",
		content: `
services:
  test:
    image: alpine
  test2:
    extends:
      service: test
      file: non-existent.yaml`,
		line:      6,
		character: 17,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges:    nil,
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return nil
		},
		prepareRename: nil,
	},
	{
		name: "extends as an object with a file attribute with an anchor that points to a non-existent file",
		content: `
services:
  test:
    image: alpine
  test2:
    extends:
      service: test
      &anchor file: non-existent.yaml`,
		line:      6,
		character: 17,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges:    nil,
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return nil
		},
		prepareRename: nil,
	},
	{
		name: "anchor name conflicts with a depends_on service (cursor on anchor)",
		content: `
services:
  first: &second
    image: scratch
    depends_on:
      - second
  second:
    image: scratch`,
		line:      2,
		character: 13,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 10, 2, 16, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 10},
								End:   protocol.Position{Line: 2, Character: 16},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 10},
			End:   protocol.Position{Line: 2, Character: 16},
		},
	},
	{
		name: "nested anchor within an anchored MappingNode",
		content: `
services:
  first: &second
    image: &another scratch
    depends_on:
      - second
  second:
    image: scratch`,
		line:      3,
		character: 15,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 3, Character: 12},
				End:   protocol.Position{Line: 3, Character: 19},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 3, Character: 12},
				End:   protocol.Position{Line: 3, Character: 19},
			}, &protocol.Range{
				Start: protocol.Position{Line: 3, Character: 12},
				End:   protocol.Position{Line: 3, Character: 19},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(3, 12, 3, 19, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 3, Character: 12},
								End:   protocol.Position{Line: 3, Character: 19},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 3, Character: 12},
			End:   protocol.Position{Line: 3, Character: 19},
		},
	},
	{
		name: "anchor name conflicts with a depends_on service (cursor on read reference)",
		content: `
services:
  first: &second
    image: scratch
    depends_on:
      - second
  second:
    image: scratch`,
		line:      5,
		character: 10,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 8},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 8},
			}, &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 8},
				End:   protocol.Position{Line: 5, Character: 14},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(5, 8, 5, 14, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 8, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 8},
								End:   protocol.Position{Line: 5, Character: 14},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 8},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 5, Character: 8},
			End:   protocol.Position{Line: 5, Character: 14},
		},
	},
	{
		name: "anchor name conflicts with a depends_on service (cursor on write reference)",
		content: `
services:
  first: &second
    image: scratch
    depends_on:
      - second
  second:
    image: scratch`,
		line:      6,
		character: 6,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 8},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 8},
			}, &protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 8},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(5, 8, 5, 14, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 8, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 8},
								End:   protocol.Position{Line: 5, Character: 14},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 8},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 6, Character: 2},
			End:   protocol.Position{Line: 6, Character: 8},
		},
	},
	{
		name: "anchor name conflicts with an extends service (cursor on anchor)",
		content: `
services:
  first: &second
    image: scratch
    extends:
      service: second
  second:
    image: scratch`,
		line:      2,
		character: 13,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 10, 2, 16, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 10},
								End:   protocol.Position{Line: 2, Character: 16},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 10},
			End:   protocol.Position{Line: 2, Character: 16},
		},
	},
	{
		name: "anchor name conflicts with an extends service (cursor on read reference)",
		content: `
services:
  first: &second
    image: scratch
    extends:
      service: second
  second:
    image: scratch`,
		line:      5,
		character: 18,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 8},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 8},
			}, &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 15},
				End:   protocol.Position{Line: 5, Character: 21},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(5, 15, 5, 21, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 8, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 15},
								End:   protocol.Position{Line: 5, Character: 21},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 8},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 5, Character: 15},
			End:   protocol.Position{Line: 5, Character: 21},
		},
	},
	{
		name: "anchor name conflicts with an extends service (cursor on write reference)",
		content: `
services:
  first: &second
    image: scratch
    extends:
      service: second
  second:
    image: scratch`,
		line:      6,
		character: 6,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 8},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 8},
			}, &protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 8},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(5, 15, 5, 21, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 8, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 15},
								End:   protocol.Position{Line: 5, Character: 21},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 8},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 6, Character: 2},
			End:   protocol.Position{Line: 6, Character: 8},
		},
	},
	{
		name: "invalid services value",
		content: `
services: true`,
		line:          1,
		character:     12,
		locations:     func(u protocol.DocumentUri) any { return nil },
		links:         func(u protocol.DocumentUri) any { return nil },
		ranges:        nil,
		renameEdits:   func(u protocol.DocumentUri) *protocol.WorkspaceEdit { return nil },
		prepareRename: nil,
	},
}

func TestDocumentHighlight_Services(t *testing.T) {
	composeFileURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(os.TempDir(), "compose.yaml")), "/"))
	u := uri.URI(composeFileURI)
	for _, tc := range serviceReferenceTestCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := document.NewComposeDocument(document.NewDocumentManager(), u, 1, []byte(tc.content))
			ranges, err := DocumentHighlight(doc, protocol.Position{Line: tc.line, Character: tc.character})
			slices.SortFunc(ranges, func(a, b protocol.DocumentHighlight) int {
				return int(a.Range.Start.Line) - int(b.Range.Start.Line)
			})
			require.NoError(t, err)
			require.Equal(t, tc.ranges, ranges)
		})
	}
}

var networkReferenceTestCases = []struct {
	name          string
	content       string
	line          protocol.UInteger
	character     protocol.UInteger
	locations     func(protocol.DocumentUri) any
	links         func(protocol.DocumentUri) any
	ranges        []protocol.DocumentHighlight
	renameEdits   func(protocol.DocumentUri) *protocol.WorkspaceEdit
	prepareRename *protocol.Range
}{
	{
		name: "write highlight on a network",
		content: `
networks:
  test:`,
		line:      2,
		character: 4,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 2},
			End:   protocol.Position{Line: 2, Character: 6},
		},
	},
	{
		name: "write highlight on a network with the networks object anchored",
		content: `
networks: &anchor
  test:`,
		line:      2,
		character: 4,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 2},
			End:   protocol.Position{Line: 2, Character: 6},
		},
	},
	{
		name: "read highlight on an undefined network array item",
		content: `
services:
  test:
    networks:
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read highlight on an undefined network object with no properties",
		content: `
services:
  test:
    networks:
      test2:`,
		line:      4,
		character: 9,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 6, 4, 11, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 6},
								End:   protocol.Position{Line: 4, Character: 11},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 6},
			End:   protocol.Position{Line: 4, Character: 11},
		},
	},
	{
		name: "read highlight on an undefined network object with properties",
		content: `
services:
  test:
    networks:
      test2:
        priority: 0`,
		line:      4,
		character: 9,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 6, 4, 11, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 6},
								End:   protocol.Position{Line: 4, Character: 11},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 6},
			End:   protocol.Position{Line: 4, Character: 11},
		},
	},
	{
		name: "read highlight on an undefined networks array item, duplicated",
		content: `
services:
  test:
    networks:
      - test2
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(5, 8, 5, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 8},
								End:   protocol.Position{Line: 5, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read/write highlight on a network array item (cursor on read)",
		content: `
services:
  test:
    networks:
      - test2
networks:
  test2:`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 8},
				End:   protocol.Position{Line: 4, Character: 13},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read/write highlight on a network array item (cursor on write)",
		content: `
services:
  test:
    networks:
      - test2
networks:
  test2:`,
		line:      6,
		character: 5,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 6, Character: 2},
			End:   protocol.Position{Line: 6, Character: 7},
		},
	},
	{
		name: "read/write highlight on a network object (read)",
		content: `
services:
  test:
    networks:
      test2:
        priority: 0
networks:
  test2:`,
		line:      4,
		character: 9,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 6},
				End:   protocol.Position{Line: 4, Character: 11},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 6, 4, 11, protocol.DocumentHighlightKindRead),
			documentHighlight(7, 2, 7, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 6},
								End:   protocol.Position{Line: 4, Character: 11},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 2},
								End:   protocol.Position{Line: 7, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 6},
			End:   protocol.Position{Line: 4, Character: 11},
		},
	},
	{
		name: "read/write highlight on a network object (write)",
		content: `
services:
  test:
    networks:
      test2:
        priority: 0
networks:
  test2:`,
		line:      7,
		character: 5,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 7},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 6, 4, 11, protocol.DocumentHighlightKindRead),
			documentHighlight(7, 2, 7, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 6},
								End:   protocol.Position{Line: 4, Character: 11},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 2},
								End:   protocol.Position{Line: 7, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 7, Character: 2},
			End:   protocol.Position{Line: 7, Character: 7},
		},
	},
	{
		name: "anchor name conflicts with a network (cursor on anchor)",
		content: `
services:
  first: &second
    image: scratch
    networks:
      - second
networks:
  second:`,
		line:      2,
		character: 13,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 10, 2, 16, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 10},
								End:   protocol.Position{Line: 2, Character: 16},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 10},
			End:   protocol.Position{Line: 2, Character: 16},
		},
	},
	{
		name: "anchor name conflicts with a network (cursor on read reference)",
		content: `
services:
  first: &second
    image: scratch
    networks:
      - second
networks:
  second:`,
		line:      5,
		character: 10,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 8},
				End:   protocol.Position{Line: 5, Character: 14},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(5, 8, 5, 14, protocol.DocumentHighlightKindRead),
			documentHighlight(7, 2, 7, 8, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 8},
								End:   protocol.Position{Line: 5, Character: 14},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 2},
								End:   protocol.Position{Line: 7, Character: 8},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 5, Character: 8},
			End:   protocol.Position{Line: 5, Character: 14},
		},
	},
	{
		name: "anchor name conflicts with a network (cursor on write reference)",
		content: `
services:
  first: &second
    image: scratch
    networks:
      - second
networks:
  second:`,
		line:      7,
		character: 6,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, &protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(5, 8, 5, 14, protocol.DocumentHighlightKindRead),
			documentHighlight(7, 2, 7, 8, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 8},
								End:   protocol.Position{Line: 5, Character: 14},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 2},
								End:   protocol.Position{Line: 7, Character: 8},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 7, Character: 2},
			End:   protocol.Position{Line: 7, Character: 8},
		},
	},
}

func TestDocumentHighlight_Networks(t *testing.T) {
	composeFileURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(os.TempDir(), "compose.yaml")), "/"))
	u := uri.URI(composeFileURI)
	for _, tc := range networkReferenceTestCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := document.NewComposeDocument(document.NewDocumentManager(), u, 1, []byte(tc.content))
			ranges, err := DocumentHighlight(doc, protocol.Position{Line: tc.line, Character: tc.character})
			require.NoError(t, err)
			require.Equal(t, tc.ranges, ranges)
		})
	}
}

var volumeReferenceTestCases = []struct {
	name          string
	content       string
	line          protocol.UInteger
	character     protocol.UInteger
	locations     func(protocol.DocumentUri) any
	links         func(protocol.DocumentUri) any
	ranges        []protocol.DocumentHighlight
	renameEdits   func(protocol.DocumentUri) *protocol.WorkspaceEdit
	prepareRename *protocol.Range
}{
	{
		name: "write highlight on a volumes",
		content: `
volumes:
  test:`,
		line:      2,
		character: 4,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 2},
			End:   protocol.Position{Line: 2, Character: 6},
		},
	},
	{
		name: "write highlight on a volumes with the volumes object value anchored",
		content: `
volumes: &anchor
  test:`,
		line:      2,
		character: 4,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 2},
			End:   protocol.Position{Line: 2, Character: 6},
		},
	},
	{
		name: "read highlight on an undefined volume array item",
		content: `
services:
  test:
    volumes:
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read highlight on an undefined volume array item with the volumes array anchored",
		content: `
services:
  test:
    &anchor volumes:
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read highlight on an undefined volume array item with an array value anchor",
		content: `
services:
  test:
    volumes: &anchor
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read highlight on an undefined volume array item with a string anchor",
		content: `
services:
  test:
    volumes:
      - &anchor test2`,
		line:      4,
		character: 18,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 16, 4, 21, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 16},
								End:   protocol.Position{Line: 4, Character: 21},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 16},
			End:   protocol.Position{Line: 4, Character: 21},
		},
	},
	{
		name: "read highlight on an undefined volume array item with a mount path",
		content: `
services:
  test:
    volumes:
      - test2:/mount/path`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read highlight on an undefined volume array item with a mount path that is quoted",
		content: `
services:
  test:
    volumes:
      - "test2:/mount/path"`,
		line:      4,
		character: 11,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 9, 4, 14, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 9},
								End:   protocol.Position{Line: 4, Character: 14},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 9},
			End:   protocol.Position{Line: 4, Character: 14},
		},
	},
	{
		name: "read highlight on a volume's string array item with a bind mount",
		content: `
services:
  test:
    volumes:
      - ./folder:/mount/path`,
		line:      4,
		character: 14,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges:    nil,
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return nil
		},
		prepareRename: nil,
	},
	{
		name: "read highlight on an undefined volume array item's mount path",
		content: `
services:
  test:
    volumes:
      - test2:/mount/path`,
		line:      4,
		character: 18,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges:    nil,
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return nil
		},
		prepareRename: nil,
	},
	{
		name: "read highlight on an invalid volume array item object",
		content: `
services:
  test:
    volumes:
      - source:`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges:    nil,
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return nil
		},
		prepareRename: nil,
	},
	{
		name: "read highlight on an undefined volume array item object's source",
		content: `
services:
  test:
    volumes:
      - source: test2`,
		line:      4,
		character: 18,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 16, 4, 21, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 16},
								End:   protocol.Position{Line: 4, Character: 21},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 16},
			End:   protocol.Position{Line: 4, Character: 21},
		},
	},
	{
		name: "read highlight on an undefined volume array item object's source with an anchor attribute name",
		content: `
services:
  test:
    volumes:
      - &anchor source: test2`,
		line:      4,
		character: 26,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 24, 4, 29, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 24},
								End:   protocol.Position{Line: 4, Character: 29},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 24},
			End:   protocol.Position{Line: 4, Character: 29},
		},
	},
	{
		name: "read highlight on an undefined volume array item object's source with an anchor attribute value",
		content: `
services:
  test:
    volumes:
      - source: &anchor test2`,
		line:      4,
		character: 26,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 24, 4, 29, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 24},
								End:   protocol.Position{Line: 4, Character: 29},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 24},
			End:   protocol.Position{Line: 4, Character: 29},
		},
	},
	{
		name: "read highlight on an undefined volume array item object's source with an anchor on the object itself",
		content: `
services:
  test:
    volumes:
      - &anchor { source: vol, target: /mount, type: volume }
volumes:
  vol:`,
		line:      4,
		character: 28,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 5},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 5},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 26},
				End:   protocol.Position{Line: 4, Character: 29},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 26, 4, 29, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 5, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 26},
								End:   protocol.Position{Line: 4, Character: 29},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 5},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 26},
			End:   protocol.Position{Line: 4, Character: 29},
		},
	},
	{
		name: "read/write highlight on an volume array item object's source (cursor on read)",
		content: `
services:
  test:
    volumes:
      - source: test2
volumes:
  test2:`,
		line:      4,
		character: 18,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 16},
				End:   protocol.Position{Line: 4, Character: 21},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 16, 4, 21, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 16},
								End:   protocol.Position{Line: 4, Character: 21},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 16},
			End:   protocol.Position{Line: 4, Character: 21},
		},
	},
	{
		name: "read highlight on an volume array item object's target which is invalid",
		content: `
services:
  test:
    volumes:
      - target: test2`,
		line:      4,
		character: 18,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges:    nil,
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return nil
		},
		prepareRename: nil,
	},
	{
		name: "read highlight on an invalid volume object",
		content: `
services:
  test:
    volumes:
      test2:`,
		line:      4,
		character: 9,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 6, 4, 11, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 6},
								End:   protocol.Position{Line: 4, Character: 11},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 6},
			End:   protocol.Position{Line: 4, Character: 11},
		},
	},
	{
		name: "read highlight on an invalid volume object with the object itself anchored",
		content: `
services:
  test:
    volumes: &anchor
      test2:`,
		line:      4,
		character: 9,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 6, 4, 11, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 6},
								End:   protocol.Position{Line: 4, Character: 11},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 6},
			End:   protocol.Position{Line: 4, Character: 11},
		},
	},
	{
		name: "read highlight on an invalid volume object with the volume object itself anchored",
		content: `
services:
  test:
    volumes: { &anchor test2: null }`,
		line:      3,
		character: 25,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(3, 23, 3, 28, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 3, Character: 23},
								End:   protocol.Position{Line: 3, Character: 28},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 3, Character: 23},
			End:   protocol.Position{Line: 3, Character: 28},
		},
	},
	{
		name: "read highlight on an undefined volumes array item, duplicated",
		content: `
services:
  test:
    volumes:
      - test2
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(5, 8, 5, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 8},
								End:   protocol.Position{Line: 5, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read/write highlight on a volume array item (cursor on read)",
		content: `
services:
  test:
    volumes:
      - test2
volumes:
  test2:`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 8},
				End:   protocol.Position{Line: 4, Character: 13},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read/write highlight on a volume array item (cursor on write)",
		content: `
services:
  test:
    volumes:
      - test2
volumes:
  test2:`,
		line:      6,
		character: 5,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 6, Character: 2},
			End:   protocol.Position{Line: 6, Character: 7},
		},
	},
	{
		name: "read/write highlight on a volume array item with a mount path (cursor on volume)",
		content: `
services:
  test:
    volumes:
      - test2:/mount/path
volumes:
  test2:`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 8},
				End:   protocol.Position{Line: 4, Character: 13},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "anchor name conflicts with a volume (cursor on anchor)",
		content: `
services:
  first: &second
    image: scratch
    volumes:
      - second
volumes:
  second:`,
		line:      2,
		character: 13,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 10, 2, 16, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 10},
								End:   protocol.Position{Line: 2, Character: 16},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 10},
			End:   protocol.Position{Line: 2, Character: 16},
		},
	},
	{
		name: "anchor name conflicts with a volume (cursor on read reference)",
		content: `
services:
  first: &second
    image: scratch
    volumes:
      - second
volumes:
  second:`,
		line:      5,
		character: 10,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 8},
				End:   protocol.Position{Line: 5, Character: 14},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(5, 8, 5, 14, protocol.DocumentHighlightKindRead),
			documentHighlight(7, 2, 7, 8, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 8},
								End:   protocol.Position{Line: 5, Character: 14},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 2},
								End:   protocol.Position{Line: 7, Character: 8},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 5, Character: 8},
			End:   protocol.Position{Line: 5, Character: 14},
		},
	},
	{
		name: "anchor name conflicts with a volume (cursor on write reference)",
		content: `
services:
  first: &second
    image: scratch
    volumes:
      - second
volumes:
  second:`,
		line:      7,
		character: 6,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, &protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(5, 8, 5, 14, protocol.DocumentHighlightKindRead),
			documentHighlight(7, 2, 7, 8, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 8},
								End:   protocol.Position{Line: 5, Character: 14},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 2},
								End:   protocol.Position{Line: 7, Character: 8},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 7, Character: 2},
			End:   protocol.Position{Line: 7, Character: 8},
		},
	},
}

func TestDocumentHighlight_Volumes(t *testing.T) {
	composeFileURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(os.TempDir(), "compose.yaml")), "/"))
	u := uri.URI(composeFileURI)
	for _, tc := range volumeReferenceTestCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := document.NewComposeDocument(document.NewDocumentManager(), u, 1, []byte(tc.content))
			ranges, err := DocumentHighlight(doc, protocol.Position{Line: tc.line, Character: tc.character})
			require.NoError(t, err)
			require.Equal(t, tc.ranges, ranges)
		})
	}
}

var configReferenceTestCases = []struct {
	name          string
	content       string
	line          protocol.UInteger
	character     protocol.UInteger
	locations     func(protocol.DocumentUri) any
	links         func(protocol.DocumentUri) any
	ranges        []protocol.DocumentHighlight
	renameEdits   func(protocol.DocumentUri) *protocol.WorkspaceEdit
	prepareRename *protocol.Range
}{
	{
		name: "write highlight on a configs",
		content: `
configs:
  test:`,
		line:      2,
		character: 4,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 2},
			End:   protocol.Position{Line: 2, Character: 6},
		},
	},
	{
		name: "write highlight on a configs with the configs object value anchored",
		content: `
configs: &anchor
  test:`,
		line:      2,
		character: 4,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 2},
			End:   protocol.Position{Line: 2, Character: 6},
		},
	},
	{
		name: "read highlight on an undefined config array item",
		content: `
services:
  test:
    configs:
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read highlight on an invalid config object",
		content: `
services:
  test:
    configs:
      test2:`,
		line:      4,
		character: 9,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges:    nil,
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return nil
		},
		prepareRename: nil,
	},
	{
		name: "read highlight on an undefined configs array item, duplicated",
		content: `
services:
  test:
    configs:
      - test2
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(5, 8, 5, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 8},
								End:   protocol.Position{Line: 5, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read/write highlight on a config array item (cursor on read)",
		content: `
services:
  test:
    configs:
      - test2
configs:
  test2:`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 8},
				End:   protocol.Position{Line: 4, Character: 13},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read/write highlight on a config array item (cursor on write)",
		content: `
services:
  test:
    configs:
      - test2
configs:
  test2:`,
		line:      6,
		character: 5,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 6, Character: 2},
			End:   protocol.Position{Line: 6, Character: 7},
		},
	},
	{
		name: "anchor name conflicts with a config (cursor on anchor)",
		content: `
services:
  first: &second
    image: scratch
    configs:
      - second
configs:
  second:`,
		line:      2,
		character: 13,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 10, 2, 16, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 10},
								End:   protocol.Position{Line: 2, Character: 16},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 10},
			End:   protocol.Position{Line: 2, Character: 16},
		},
	},
	{
		name: "anchor name conflicts with a config (cursor on read reference)",
		content: `
services:
  first: &second
    image: scratch
    configs:
      - second
configs:
  second:`,
		line:      5,
		character: 10,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 8},
				End:   protocol.Position{Line: 5, Character: 14},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(5, 8, 5, 14, protocol.DocumentHighlightKindRead),
			documentHighlight(7, 2, 7, 8, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 8},
								End:   protocol.Position{Line: 5, Character: 14},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 2},
								End:   protocol.Position{Line: 7, Character: 8},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 5, Character: 8},
			End:   protocol.Position{Line: 5, Character: 14},
		},
	},
	{
		name: "anchor name conflicts with a config (cursor on write reference)",
		content: `
services:
  first: &second
    image: scratch
    configs:
      - second
configs:
  second:`,
		line:      7,
		character: 6,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, &protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(5, 8, 5, 14, protocol.DocumentHighlightKindRead),
			documentHighlight(7, 2, 7, 8, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 8},
								End:   protocol.Position{Line: 5, Character: 14},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 2},
								End:   protocol.Position{Line: 7, Character: 8},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 7, Character: 2},
			End:   protocol.Position{Line: 7, Character: 8},
		},
	},
}

func TestDocumentHighlight_Configs(t *testing.T) {
	composeFileURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(os.TempDir(), "compose.yaml")), "/"))
	u := uri.URI(composeFileURI)
	for _, tc := range configReferenceTestCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := document.NewComposeDocument(document.NewDocumentManager(), u, 1, []byte(tc.content))
			ranges, err := DocumentHighlight(doc, protocol.Position{Line: tc.line, Character: tc.character})
			require.NoError(t, err)
			require.Equal(t, tc.ranges, ranges)
		})
	}
}

var secretReferenceTestCases = []struct {
	name          string
	content       string
	line          protocol.UInteger
	character     protocol.UInteger
	locations     func(protocol.DocumentUri) any
	links         func(protocol.DocumentUri) any
	ranges        []protocol.DocumentHighlight
	renameEdits   func(protocol.DocumentUri) *protocol.WorkspaceEdit
	prepareRename *protocol.Range
}{
	{
		name: "write highlight on a secrets",
		content: `
secrets:
  test:`,
		line:      2,
		character: 4,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 2},
			End:   protocol.Position{Line: 2, Character: 6},
		},
	},
	{
		name: "write highlight on a secrets with the secrets object value anchored",
		content: `
secrets: &anchor
  test:`,
		line:      2,
		character: 4,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 2},
			End:   protocol.Position{Line: 2, Character: 6},
		},
	},
	{
		name: "read highlight on an undefined secret array item",
		content: `
services:
  test:
    secrets:
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read highlight on an invalid secret object",
		content: `
services:
  test:
    secrets:
      test2:`,
		line:      4,
		character: 9,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges:    nil,
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return nil
		},
		prepareRename: nil,
	},
	{
		name: "read highlight on an undefined secrets array item, duplicated",
		content: `
services:
  test:
    secrets:
      - test2
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(5, 8, 5, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 8},
								End:   protocol.Position{Line: 5, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read/write highlight on a secret array item (cursor on read)",
		content: `
services:
  test:
    secrets:
      - test2
secrets:
  test2:`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 8},
				End:   protocol.Position{Line: 4, Character: 13},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read/write highlight on a secret array item (cursor on write)",
		content: `
services:
  test:
    secrets:
      - test2
secrets:
  test2:`,
		line:      6,
		character: 5,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 6, Character: 2},
			End:   protocol.Position{Line: 6, Character: 7},
		},
	},
	{
		name: "anchor name conflicts with a secret (cursor on anchor)",
		content: `
services:
  first: &second
    image: scratch
    secrets:
      - second
secrets:
  second:`,
		line:      2,
		character: 13,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 10, 2, 16, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 10},
								End:   protocol.Position{Line: 2, Character: 16},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 10},
			End:   protocol.Position{Line: 2, Character: 16},
		},
	},
	{
		name: "anchor name conflicts with a secret (cursor on read reference)",
		content: `
services:
  first: &second
    image: scratch
    secrets:
      - second
secrets:
  second:`,
		line:      5,
		character: 10,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 8},
				End:   protocol.Position{Line: 5, Character: 14},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(5, 8, 5, 14, protocol.DocumentHighlightKindRead),
			documentHighlight(7, 2, 7, 8, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 8},
								End:   protocol.Position{Line: 5, Character: 14},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 2},
								End:   protocol.Position{Line: 7, Character: 8},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 5, Character: 8},
			End:   protocol.Position{Line: 5, Character: 14},
		},
	},
	{
		name: "anchor name conflicts with a secret (cursor on write reference)",
		content: `
services:
  first: &second
    image: scratch
    secrets:
      - second
secrets:
  second:`,
		line:      7,
		character: 6,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, &protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(5, 8, 5, 14, protocol.DocumentHighlightKindRead),
			documentHighlight(7, 2, 7, 8, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 8},
								End:   protocol.Position{Line: 5, Character: 14},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 2},
								End:   protocol.Position{Line: 7, Character: 8},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 7, Character: 2},
			End:   protocol.Position{Line: 7, Character: 8},
		},
	},
}

func TestDocumentHighlight_Secrets(t *testing.T) {
	composeFileURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(os.TempDir(), "compose.yaml")), "/"))
	u := uri.URI(composeFileURI)
	for _, tc := range secretReferenceTestCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := document.NewComposeDocument(document.NewDocumentManager(), u, 1, []byte(tc.content))
			ranges, err := DocumentHighlight(doc, protocol.Position{Line: tc.line, Character: tc.character})
			require.NoError(t, err)
			require.Equal(t, tc.ranges, ranges)
		})
	}
}

var modelReferenceTestCases = []struct {
	name          string
	content       string
	line          protocol.UInteger
	character     protocol.UInteger
	locations     func(protocol.DocumentUri) any
	links         func(protocol.DocumentUri) any
	ranges        []protocol.DocumentHighlight
	renameEdits   func(protocol.DocumentUri) *protocol.WorkspaceEdit
	prepareRename *protocol.Range
}{
	{
		name: "write highlight on a models object",
		content: `
models:
  test:`,
		line:      2,
		character: 4,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 2},
			End:   protocol.Position{Line: 2, Character: 6},
		},
	},
	{
		name: "write highlight on a model with the top level models object value anchored",
		content: `
models: &anchor
  test:`,
		line:      2,
		character: 4,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 2, 2, 6, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 2},
								End:   protocol.Position{Line: 2, Character: 6},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 2},
			End:   protocol.Position{Line: 2, Character: 6},
		},
	},
	{
		name: "read highlight on an undefined models array item",
		content: `
services:
  test:
    models:
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read highlight on an undefined models array item with the models array anchored",
		content: `
services:
  test:
    &anchor models:
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read highlight on an undefined models array item with the models array value anchor",
		content: `
services:
  test:
    models: &anchor
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read highlight on an undefined models array item with a string value anchored",
		content: `
services:
  test:
    models:
      - &anchor test2`,
		line:      4,
		character: 18,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 16, 4, 21, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 16},
								End:   protocol.Position{Line: 4, Character: 21},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 16},
			End:   protocol.Position{Line: 4, Character: 21},
		},
	},
	{
		name: "read highlight on an undefined models object",
		content: `
services:
  test:
    models:
      test2:`,
		line:      4,
		character: 9,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 6, 4, 11, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 6},
								End:   protocol.Position{Line: 4, Character: 11},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 6},
			End:   protocol.Position{Line: 4, Character: 11},
		},
	},
	{
		name: "read highlight on an undefined models object with the object itself anchored",
		content: `
services:
  test:
    models: &anchor
      test2:`,
		line:      4,
		character: 9,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 6, 4, 11, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 6},
								End:   protocol.Position{Line: 4, Character: 11},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 6},
			End:   protocol.Position{Line: 4, Character: 11},
		},
	},
	{
		name: "read highlight on an undefined volumes array item, duplicated",
		content: `
services:
  test:
    models:
      - test2
      - test2`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(5, 8, 5, 13, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 8},
								End:   protocol.Position{Line: 5, Character: 13},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read/write highlight on a models array item (cursor on read)",
		content: `
services:
  test:
    models:
      - test2
models:
  test2:`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 8},
				End:   protocol.Position{Line: 4, Character: 13},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 8},
			End:   protocol.Position{Line: 4, Character: 13},
		},
	},
	{
		name: "read/write highlight on a models array item (cursor on write)",
		content: `
services:
  test:
    models:
      - test2
models:
  test2:`,
		line:      6,
		character: 5,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 8, 4, 13, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 8},
								End:   protocol.Position{Line: 4, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 6, Character: 2},
			End:   protocol.Position{Line: 6, Character: 7},
		},
	},
	{
		name: "read/write highlight on a models object (cursor on read)",
		content: `
services:
  test:
    models:
      test2:
models:
  test2:`,
		line:      4,
		character: 10,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 6},
				End:   protocol.Position{Line: 4, Character: 11},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 6, 4, 11, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 6},
								End:   protocol.Position{Line: 4, Character: 11},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 6},
			End:   protocol.Position{Line: 4, Character: 11},
		},
	},
	{
		name: "read/write highlight on a models object (cursor on write)",
		content: `
services:
  test:
    models:
      test2:
models:
  test2:`,
		line:      6,
		character: 5,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, &protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 7},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 6, 4, 11, protocol.DocumentHighlightKindRead),
			documentHighlight(6, 2, 6, 7, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 6},
								End:   protocol.Position{Line: 4, Character: 11},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 2},
								End:   protocol.Position{Line: 6, Character: 7},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 6, Character: 2},
			End:   protocol.Position{Line: 6, Character: 7},
		},
	},
	{
		name: "anchor name conflicts with a model (cursor on anchor)",
		content: `
services:
  first: &second
    image: scratch
    models:
      - second
models:
  second:`,
		line:      2,
		character: 13,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 10},
				End:   protocol.Position{Line: 2, Character: 16},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 10, 2, 16, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 10},
								End:   protocol.Position{Line: 2, Character: 16},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 10},
			End:   protocol.Position{Line: 2, Character: 16},
		},
	},
	{
		name: "anchor name conflicts with a model (cursor on read reference)",
		content: `
services:
  first: &second
    image: scratch
    models:
      - second
models:
  second:`,
		line:      5,
		character: 10,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 8},
				End:   protocol.Position{Line: 5, Character: 14},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(5, 8, 5, 14, protocol.DocumentHighlightKindRead),
			documentHighlight(7, 2, 7, 8, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 8},
								End:   protocol.Position{Line: 5, Character: 14},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 2},
								End:   protocol.Position{Line: 7, Character: 8},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 5, Character: 8},
			End:   protocol.Position{Line: 5, Character: 14},
		},
	},
	{
		name: "anchor name conflicts with a model (cursor on write reference)",
		content: `
services:
  first: &second
    image: scratch
    models:
      - second
models:
  second:`,
		line:      7,
		character: 6,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, &protocol.Range{
				Start: protocol.Position{Line: 7, Character: 2},
				End:   protocol.Position{Line: 7, Character: 8},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(5, 8, 5, 14, protocol.DocumentHighlightKindRead),
			documentHighlight(7, 2, 7, 8, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 8},
								End:   protocol.Position{Line: 5, Character: 14},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 2},
								End:   protocol.Position{Line: 7, Character: 8},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 7, Character: 2},
			End:   protocol.Position{Line: 7, Character: 8},
		},
	},
}

func TestDocumentHighlight_Models(t *testing.T) {
	composeFileURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(os.TempDir(), "compose.yaml")), "/"))
	u := uri.URI(composeFileURI)
	for _, tc := range modelReferenceTestCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := document.NewComposeDocument(document.NewDocumentManager(), u, 1, []byte(tc.content))
			ranges, err := DocumentHighlight(doc, protocol.Position{Line: tc.line, Character: tc.character})
			require.NoError(t, err)
			require.Equal(t, tc.ranges, ranges)
		})
	}
}

var fragmentTestCases = []struct {
	name          string
	content       string
	line          protocol.UInteger
	character     protocol.UInteger
	locations     func(protocol.DocumentUri) any
	links         func(protocol.DocumentUri) any
	ranges        []protocol.DocumentHighlight
	renameEdits   func(protocol.DocumentUri) *protocol.WorkspaceEdit
	prepareRename *protocol.Range
}{
	{
		name: "anchor with no alias",
		content: `
volumes:
  db-data: &default-volume
    driver: custom`,
		line:      2,
		character: 17,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 12},
				End:   protocol.Position{Line: 2, Character: 26},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 12},
				End:   protocol.Position{Line: 2, Character: 26},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 12},
				End:   protocol.Position{Line: 2, Character: 26},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 12, 2, 26, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 12},
								End:   protocol.Position{Line: 2, Character: 26},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 12},
			End:   protocol.Position{Line: 2, Character: 26},
		},
	},
	{
		name: "anchor with alias (cursor on anchor)",
		content: `
volumes:
  db-data: &default-volume
    driver: default
  metrics: *default-volume`,
		line:      2,
		character: 17,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 12},
				End:   protocol.Position{Line: 2, Character: 26},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 12},
				End:   protocol.Position{Line: 2, Character: 26},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 12},
				End:   protocol.Position{Line: 2, Character: 26},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 12, 2, 26, protocol.DocumentHighlightKindWrite),
			documentHighlight(4, 12, 4, 26, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 12},
								End:   protocol.Position{Line: 2, Character: 26},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 12},
								End:   protocol.Position{Line: 4, Character: 26},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 12},
			End:   protocol.Position{Line: 2, Character: 26},
		},
	},
	{
		name: "anchor with alias (cursor on alias)",
		content: `
volumes:
  db-data: &default-volume
    driver: default
  metrics: *default-volume`,
		line:      4,
		character: 17,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 12},
				End:   protocol.Position{Line: 2, Character: 26},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 12},
				End:   protocol.Position{Line: 2, Character: 26},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 12},
				End:   protocol.Position{Line: 4, Character: 26},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 12, 2, 26, protocol.DocumentHighlightKindWrite),
			documentHighlight(4, 12, 4, 26, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 12},
								End:   protocol.Position{Line: 2, Character: 26},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 12},
								End:   protocol.Position{Line: 4, Character: 26},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 12},
			End:   protocol.Position{Line: 4, Character: 26},
		},
	},
	{
		name: "anchor with alias pointing at the second alias",
		content: `
volumes:
  db-data: &default-volume
    driver: default
  metrics: *default-volume
  another: *default-volume`,
		line:      5,
		character: 17,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 12},
				End:   protocol.Position{Line: 2, Character: 26},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 12},
				End:   protocol.Position{Line: 2, Character: 26},
			}, &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 12},
				End:   protocol.Position{Line: 5, Character: 26},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 12, 2, 26, protocol.DocumentHighlightKindWrite),
			documentHighlight(4, 12, 4, 26, protocol.DocumentHighlightKindRead),
			documentHighlight(5, 12, 5, 26, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 12},
								End:   protocol.Position{Line: 2, Character: 26},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 12},
								End:   protocol.Position{Line: 4, Character: 26},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 12},
								End:   protocol.Position{Line: 5, Character: 26},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 5, Character: 12},
			End:   protocol.Position{Line: 5, Character: 26},
		},
	},
	{
		name: "cursor is over whitespace",
		content: `
volumes:
  db-data: &default-volume
    driver: default
  metrics: *default-volume`,
		line:      4,
		character: 0,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges:    nil,
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return nil
		},
		prepareRename: nil,
	},
	{
		name: "read reference on the first duplicated alias",
		content: `
services:
  serviceA:
    image: &redis redis:8-alpine
  serviceB:
    image: *redis
  serviceC:
    image: &redis redis:7-alpine
  serviceD:
    image: *redis`,
		line:      5,
		character: 14,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 3, Character: 12},
				End:   protocol.Position{Line: 3, Character: 17},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 3, Character: 12},
				End:   protocol.Position{Line: 3, Character: 17},
			}, &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 12},
				End:   protocol.Position{Line: 5, Character: 17},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(3, 12, 3, 17, protocol.DocumentHighlightKindWrite),
			documentHighlight(5, 12, 5, 17, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 3, Character: 12},
								End:   protocol.Position{Line: 3, Character: 17},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 12},
								End:   protocol.Position{Line: 5, Character: 17},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 5, Character: 12},
			End:   protocol.Position{Line: 5, Character: 17},
		},
	},
	{
		name: "write reference on the first duplicated anchor",
		content: `
services:
  serviceA:
    image: &redis redis:8-alpine
  serviceB:
    image: *redis
  serviceC:
    image: &redis redis:7-alpine
  serviceD:
    image: *redis`,
		line:      3,
		character: 14,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 3, Character: 12},
				End:   protocol.Position{Line: 3, Character: 17},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 3, Character: 12},
				End:   protocol.Position{Line: 3, Character: 17},
			}, &protocol.Range{
				Start: protocol.Position{Line: 3, Character: 12},
				End:   protocol.Position{Line: 3, Character: 17},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(3, 12, 3, 17, protocol.DocumentHighlightKindWrite),
			documentHighlight(5, 12, 5, 17, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 3, Character: 12},
								End:   protocol.Position{Line: 3, Character: 17},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 12},
								End:   protocol.Position{Line: 5, Character: 17},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 3, Character: 12},
			End:   protocol.Position{Line: 3, Character: 17},
		},
	},
	{
		name: "read reference on the second duplicated alias",
		content: `
services:
  serviceA:
    image: &redis redis:8-alpine
  serviceB:
    image: *redis
  serviceC:
    image: &redis redis:7-alpine
  serviceD:
    image: *redis`,
		line:      9,
		character: 14,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 12},
				End:   protocol.Position{Line: 7, Character: 17},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 12},
				End:   protocol.Position{Line: 7, Character: 17},
			}, &protocol.Range{
				Start: protocol.Position{Line: 9, Character: 12},
				End:   protocol.Position{Line: 9, Character: 17},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(7, 12, 7, 17, protocol.DocumentHighlightKindWrite),
			documentHighlight(9, 12, 9, 17, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 12},
								End:   protocol.Position{Line: 7, Character: 17},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 9, Character: 12},
								End:   protocol.Position{Line: 9, Character: 17},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 9, Character: 12},
			End:   protocol.Position{Line: 9, Character: 17},
		},
	},
	{
		name: "write reference on the second duplicated anchor",
		content: `
services:
  serviceA:
    image: &redis redis:8-alpine
  serviceB:
    image: *redis
  serviceC:
    image: &redis redis:7-alpine
  serviceD:
    image: *redis`,
		line:      7,
		character: 14,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 12},
				End:   protocol.Position{Line: 7, Character: 17},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 7, Character: 12},
				End:   protocol.Position{Line: 7, Character: 17},
			}, &protocol.Range{
				Start: protocol.Position{Line: 7, Character: 12},
				End:   protocol.Position{Line: 7, Character: 17},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(7, 12, 7, 17, protocol.DocumentHighlightKindWrite),
			documentHighlight(9, 12, 9, 17, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 12},
								End:   protocol.Position{Line: 7, Character: 17},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 9, Character: 12},
								End:   protocol.Position{Line: 9, Character: 17},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 7, Character: 12},
			End:   protocol.Position{Line: 7, Character: 17},
		},
	},
	{
		name: "multiple anchors",
		content: `
services:
  serviceA:
    image: &redis8 redis:8-alpine
  serviceB:
    image: *redis8
  serviceC:
    image: &redis7 redis:7-alpine
  serviceD:
    image: *redis7`,
		line:      3,
		character: 14,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 3, Character: 12},
				End:   protocol.Position{Line: 3, Character: 18},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 3, Character: 12},
				End:   protocol.Position{Line: 3, Character: 18},
			}, &protocol.Range{
				Start: protocol.Position{Line: 3, Character: 12},
				End:   protocol.Position{Line: 3, Character: 18},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(3, 12, 3, 18, protocol.DocumentHighlightKindWrite),
			documentHighlight(5, 12, 5, 18, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 3, Character: 12},
								End:   protocol.Position{Line: 3, Character: 18},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 12},
								End:   protocol.Position{Line: 5, Character: 18},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 3, Character: 12},
			End:   protocol.Position{Line: 3, Character: 18},
		},
	},
	{
		name: "anchor in an array, write reference",
		content: `
services:
  serviceA:
    labels:
      - &label a.b.c=value
  serviceB:
    labels:
      - *label`,
		line:      4,
		character: 11,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 9},
				End:   protocol.Position{Line: 4, Character: 14},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 9},
				End:   protocol.Position{Line: 4, Character: 14},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 9},
				End:   protocol.Position{Line: 4, Character: 14},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 9, 4, 14, protocol.DocumentHighlightKindWrite),
			documentHighlight(7, 9, 7, 14, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 9},
								End:   protocol.Position{Line: 4, Character: 14},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 9},
								End:   protocol.Position{Line: 7, Character: 14},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 9},
			End:   protocol.Position{Line: 4, Character: 14},
		},
	},
	{
		name: "anchor in an array, read reference",
		content: `
services:
  serviceA:
    labels:
      - &label a.b.c=value
  serviceB:
    labels:
      - *label`,
		line:      7,
		character: 11,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 9},
				End:   protocol.Position{Line: 4, Character: 14},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 9},
				End:   protocol.Position{Line: 4, Character: 14},
			}, &protocol.Range{
				Start: protocol.Position{Line: 7, Character: 9},
				End:   protocol.Position{Line: 7, Character: 14},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 9, 4, 14, protocol.DocumentHighlightKindWrite),
			documentHighlight(7, 9, 7, 14, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 9},
								End:   protocol.Position{Line: 4, Character: 14},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 9},
								End:   protocol.Position{Line: 7, Character: 14},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 7, Character: 9},
			End:   protocol.Position{Line: 7, Character: 14},
		},
	},
	{
		name: "anchor in an object inside an array, read reference",
		content: `
services:
  backend:
    volumes:
      - type: &volumeType bind
        source: vol
        target: /data
      - type: *volumeType
        source: vol
        target: /data2`,
		line:      7,
		character: 20,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 15},
				End:   protocol.Position{Line: 4, Character: 25},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 15},
				End:   protocol.Position{Line: 4, Character: 25},
			}, &protocol.Range{
				Start: protocol.Position{Line: 7, Character: 15},
				End:   protocol.Position{Line: 7, Character: 25},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 15, 4, 25, protocol.DocumentHighlightKindWrite),
			documentHighlight(7, 15, 7, 25, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 15},
								End:   protocol.Position{Line: 4, Character: 25},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 15},
								End:   protocol.Position{Line: 7, Character: 25},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 7, Character: 15},
			End:   protocol.Position{Line: 7, Character: 25},
		},
	},
	{
		name: "anchor/alias references all on the same line",
		content: `
services:
  frontend:
    build:
      tags: [&keys aa, *keys, &keys bb, *keys, &keys cc, *keys]`,
		line:      4,
		character: 43,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 31},
				End:   protocol.Position{Line: 4, Character: 35},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 31},
				End:   protocol.Position{Line: 4, Character: 35},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 41},
				End:   protocol.Position{Line: 4, Character: 45},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 31, 4, 35, protocol.DocumentHighlightKindWrite),
			documentHighlight(4, 41, 4, 45, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 31},
								End:   protocol.Position{Line: 4, Character: 35},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 41},
								End:   protocol.Position{Line: 4, Character: 45},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 41},
			End:   protocol.Position{Line: 4, Character: 45},
		},
	},
	{
		name: "anchor/alias references are staggered",
		content: `
services:
  frontend:
    build:
      tags: [&keys aa]
  backend:
    build:
      tags: [*keys, &keys bb, *keys]`,
		line:      7,
		character: 16,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 14},
				End:   protocol.Position{Line: 4, Character: 18},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 14},
				End:   protocol.Position{Line: 4, Character: 18},
			}, &protocol.Range{
				Start: protocol.Position{Line: 7, Character: 14},
				End:   protocol.Position{Line: 7, Character: 18},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 14, 4, 18, protocol.DocumentHighlightKindWrite),
			documentHighlight(7, 14, 7, 18, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 14},
								End:   protocol.Position{Line: 4, Character: 18},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 14},
								End:   protocol.Position{Line: 7, Character: 18},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 7, Character: 14},
			End:   protocol.Position{Line: 7, Character: 18},
		},
	},
	{
		name: "duplicated anchor/alias references all on the same line (cursor on first anchor)",
		content: `
services:
  frontend:
    build:
      tags: [&keys ab, *keys]
  backend:
    build:
      tags: [&keys ab, *keys]`,
		line:      4,
		character: 16,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 14},
				End:   protocol.Position{Line: 4, Character: 18},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 14},
				End:   protocol.Position{Line: 4, Character: 18},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 14},
				End:   protocol.Position{Line: 4, Character: 18},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 14, 4, 18, protocol.DocumentHighlightKindWrite),
			documentHighlight(4, 24, 4, 28, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 14},
								End:   protocol.Position{Line: 4, Character: 18},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 24},
								End:   protocol.Position{Line: 4, Character: 28},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 14},
			End:   protocol.Position{Line: 4, Character: 18},
		},
	},
	{
		name: "duplicated anchor/alias references all on the same line (cursor on first alias)",
		content: `
services:
  frontend:
    build:
      tags: [&keys ab, *keys]
  backend:
    build:
      tags: [&keys ab, *keys]`,
		line:      4,
		character: 26,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 14},
				End:   protocol.Position{Line: 4, Character: 18},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 14},
				End:   protocol.Position{Line: 4, Character: 18},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 24},
				End:   protocol.Position{Line: 4, Character: 28},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 14, 4, 18, protocol.DocumentHighlightKindWrite),
			documentHighlight(4, 24, 4, 28, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 14},
								End:   protocol.Position{Line: 4, Character: 18},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 24},
								End:   protocol.Position{Line: 4, Character: 28},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 24},
			End:   protocol.Position{Line: 4, Character: 28},
		},
	},
	{
		name: "interweaving fragments on the first anchor",
		content: `
services:
  test: &test
    image: alpine:3.22
  test2: &testAgain
    image: alpine:3.21
  test3: *test
  test4: *testAgain`,
		line:      2,
		character: 11,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 9},
				End:   protocol.Position{Line: 2, Character: 13},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 9},
				End:   protocol.Position{Line: 2, Character: 13},
			}, &protocol.Range{
				Start: protocol.Position{Line: 2, Character: 9},
				End:   protocol.Position{Line: 2, Character: 13},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 9, 2, 13, protocol.DocumentHighlightKindWrite),
			documentHighlight(6, 10, 6, 14, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 9},
								End:   protocol.Position{Line: 2, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 10},
								End:   protocol.Position{Line: 6, Character: 14},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 9},
			End:   protocol.Position{Line: 2, Character: 13},
		},
	},
	{
		name: "interweaving fragments on the first alias",
		content: `
services:
  test: &test
    image: alpine:3.22
  test2: &testAgain
    image: alpine:3.21
  test3: *test
  test4: *testAgain`,
		line:      6,
		character: 12,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 9},
				End:   protocol.Position{Line: 2, Character: 13},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 2, Character: 9},
				End:   protocol.Position{Line: 2, Character: 13},
			}, &protocol.Range{
				Start: protocol.Position{Line: 6, Character: 10},
				End:   protocol.Position{Line: 6, Character: 14},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 9, 2, 13, protocol.DocumentHighlightKindWrite),
			documentHighlight(6, 10, 6, 14, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 9},
								End:   protocol.Position{Line: 2, Character: 13},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 6, Character: 10},
								End:   protocol.Position{Line: 6, Character: 14},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 6, Character: 10},
			End:   protocol.Position{Line: 6, Character: 14},
		},
	},
	{
		name: "interweaving fragments on the second anchor",
		content: `
services:
  test: &test
    image: alpine:3.22
  test2: &testAgain
    image: alpine:3.21
  test3: *test
  test4: *testAgain`,
		line:      4,
		character: 12,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 10},
				End:   protocol.Position{Line: 4, Character: 19},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 10},
				End:   protocol.Position{Line: 4, Character: 19},
			}, &protocol.Range{
				Start: protocol.Position{Line: 4, Character: 10},
				End:   protocol.Position{Line: 4, Character: 19},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 10, 4, 19, protocol.DocumentHighlightKindWrite),
			documentHighlight(7, 10, 7, 19, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 10},
								End:   protocol.Position{Line: 4, Character: 19},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 10},
								End:   protocol.Position{Line: 7, Character: 19},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 10},
			End:   protocol.Position{Line: 4, Character: 19},
		},
	},
	{
		name: "interweaving fragments on the second alias",
		content: `
services:
  test: &test
    image: alpine:3.22
  test2: &testAgain
    image: alpine:3.21
  test3: *test
  test4: *testAgain`,
		line:      7,
		character: 12,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 10},
				End:   protocol.Position{Line: 4, Character: 19},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 10},
				End:   protocol.Position{Line: 4, Character: 19},
			}, &protocol.Range{
				Start: protocol.Position{Line: 7, Character: 10},
				End:   protocol.Position{Line: 7, Character: 19},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 10, 4, 19, protocol.DocumentHighlightKindWrite),
			documentHighlight(7, 10, 7, 19, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 10},
								End:   protocol.Position{Line: 4, Character: 19},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 10},
								End:   protocol.Position{Line: 7, Character: 19},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 7, Character: 10},
			End:   protocol.Position{Line: 7, Character: 19},
		},
	},
	{
		name: "aliases with no matching anchor",
		content: `
services:
  test1: *test
  test2: *test
  test3: *test`,
		line:      2,
		character: 12,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 10, 2, 14, protocol.DocumentHighlightKindRead),
			documentHighlight(3, 10, 3, 14, protocol.DocumentHighlightKindRead),
			documentHighlight(4, 10, 4, 14, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 10},
								End:   protocol.Position{Line: 2, Character: 14},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 3, Character: 10},
								End:   protocol.Position{Line: 3, Character: 14},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 10},
								End:   protocol.Position{Line: 4, Character: 14},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 10},
			End:   protocol.Position{Line: 2, Character: 14},
		},
	},
	{
		name: "alias before the actual anchor",
		content: `
services:
  test1: *test
  test2: &test
  test3: *test`,
		line:      2,
		character: 12,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(2, 10, 2, 14, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 2, Character: 10},
								End:   protocol.Position{Line: 2, Character: 14},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 2, Character: 10},
			End:   protocol.Position{Line: 2, Character: 14},
		},
	},
	{
		name: "alias before the actual anchor on the same line",
		content: `
services:
  test:
    build:
      tags: [*test, &test t1, *test]`,
		line:      4,
		character: 16,
		locations: func(u protocol.DocumentUri) any { return nil },
		links:     func(u protocol.DocumentUri) any { return nil },
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 14, 4, 18, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 14},
								End:   protocol.Position{Line: 4, Character: 18},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 4, Character: 14},
			End:   protocol.Position{Line: 4, Character: 18},
		},
	},
	{
		name: "anchored sequence with an alias anchor in the sequence",
		content: `
services:
  test:
    networks:
      - &anchor test
  test2:
    networks: &anchor2
      - *anchor
networks:
  test:
  test2:`,
		line:      7,
		character: 12,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 9},
				End:   protocol.Position{Line: 4, Character: 15},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 4, Character: 9},
				End:   protocol.Position{Line: 4, Character: 15},
			}, &protocol.Range{
				Start: protocol.Position{Line: 7, Character: 9},
				End:   protocol.Position{Line: 7, Character: 15},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(4, 9, 4, 15, protocol.DocumentHighlightKindWrite),
			documentHighlight(7, 9, 7, 15, protocol.DocumentHighlightKindRead),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 4, Character: 9},
								End:   protocol.Position{Line: 4, Character: 15},
							},
						},
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 7, Character: 9},
								End:   protocol.Position{Line: 7, Character: 15},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 7, Character: 9},
			End:   protocol.Position{Line: 7, Character: 15},
		},
	},
	{
		name: "write reference on a nested anchor with main anchor on a sequence",
		content: `
services:
  test:
    image: alpine:3.21
    volumes: &anchor
      - source: &anchor2 vol
        target: /target
        type: volume
volumes:
  vol:`,
		line:      5,
		character: 20,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 5, Character: 17},
				End:   protocol.Position{Line: 5, Character: 24},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 5, Character: 17},
				End:   protocol.Position{Line: 5, Character: 24},
			}, &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 17},
				End:   protocol.Position{Line: 5, Character: 24},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(5, 17, 5, 24, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 17},
								End:   protocol.Position{Line: 5, Character: 24},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 5, Character: 17},
			End:   protocol.Position{Line: 5, Character: 24},
		},
	},
	{
		name: "write reference on a nested anchor in a sequence with main anchor",
		content: `
services:
  test:
    image: alpine:3.21
    volumes: &anchor
      - &anchor2 vol
volumes:
  vol:`,
		line:      5,
		character: 14,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 5, Character: 9},
				End:   protocol.Position{Line: 5, Character: 16},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 5, Character: 9},
				End:   protocol.Position{Line: 5, Character: 16},
			}, &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 9},
				End:   protocol.Position{Line: 5, Character: 16},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(5, 9, 5, 16, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 9},
								End:   protocol.Position{Line: 5, Character: 16},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 5, Character: 9},
			End:   protocol.Position{Line: 5, Character: 16},
		},
	},
	{
		name: "write reference on a nested anchor with main anchor on an object",
		content: `
services:
  test:
    image: alpine:3.21
    volumes:
      - &anchor { source: &anchor2 vol, target: /target, type: volume }
volumes:
  vol:`,
		line:      5,
		character: 30,
		locations: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(false, protocol.Range{
				Start: protocol.Position{Line: 5, Character: 27},
				End:   protocol.Position{Line: 5, Character: 34},
			}, nil, u)
		},
		links: func(u protocol.DocumentUri) any {
			return types.CreateDefinitionResult(true, protocol.Range{
				Start: protocol.Position{Line: 5, Character: 27},
				End:   protocol.Position{Line: 5, Character: 34},
			}, &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 27},
				End:   protocol.Position{Line: 5, Character: 34},
			}, u)
		},
		ranges: []protocol.DocumentHighlight{
			documentHighlight(5, 27, 5, 34, protocol.DocumentHighlightKindWrite),
		},
		renameEdits: func(u protocol.DocumentUri) *protocol.WorkspaceEdit {
			return &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					u: {
						{
							NewText: "newName",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 27},
								End:   protocol.Position{Line: 5, Character: 34},
							},
						},
					},
				},
			}
		},
		prepareRename: &protocol.Range{
			Start: protocol.Position{Line: 5, Character: 27},
			End:   protocol.Position{Line: 5, Character: 34},
		},
	},
}

func TestDocumentHighlight_Fragments(t *testing.T) {
	composeFileURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(os.TempDir(), "compose.yaml")), "/"))
	u := uri.URI(composeFileURI)
	for _, tc := range fragmentTestCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := document.NewComposeDocument(document.NewDocumentManager(), u, 1, []byte(tc.content))
			ranges, err := DocumentHighlight(doc, protocol.Position{Line: tc.line, Character: tc.character})
			require.NoError(t, err)
			require.Equal(t, tc.ranges, ranges)
		})
	}
}
