package hcl

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
	"github.com/dromara/carbon/v2"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/uri"
)

func TestCompletion(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(wd)))
	completionTestFolderPath := path.Join(projectRoot, "testdata", "completion")

	testCases := []struct {
		name              string
		content           string
		dockerfileContent string
		line              uint32
		character         uint32
		items             []protocol.CompletionItem
	}{
		{
			name:              "inside a comment with no content",
			content:           "# comment",
			line:              0,
			dockerfileContent: "",
			character:         2,
			items:             []protocol.CompletionItem{},
		},
		{
			name:              "empty file",
			content:           "",
			line:              0,
			dockerfileContent: "",
			character:         0,
			items: []protocol.CompletionItem{
				{
					Detail:           types.CreateStringPointer("Block"),
					Kind:             types.CreateCompletionItemKindPointer(protocol.CompletionItemKindClass),
					Label:            "function",
					InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: 0},
						},
						NewText: "function \"${1:functionName}\" {\n  ${2}\n}",
					},
				},
				{
					Detail:           types.CreateStringPointer("Block"),
					Kind:             types.CreateCompletionItemKindPointer(protocol.CompletionItemKindClass),
					Label:            "group",
					InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: 0},
						},
						NewText: "group \"${1:groupName}\" {\n  ${2}\n}",
					},
				},
				{
					Detail:           types.CreateStringPointer("Block"),
					Kind:             types.CreateCompletionItemKindPointer(protocol.CompletionItemKindClass),
					Label:            "target",
					InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: 0},
						},
						NewText: "target \"${1:targetName}\" {\n  ${2}\n}",
					},
				},
				{
					Detail:           types.CreateStringPointer("Block"),
					Kind:             types.CreateCompletionItemKindPointer(protocol.CompletionItemKindClass),
					Label:            "variable",
					InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: 0},
						},
						NewText: "variable \"${1:variableName}\" {\n  ${2}\n}",
					},
				},
			},
		},
		{
			name:              "group block's attributes",
			content:           "group \"default\" {\n  \n}",
			line:              1,
			dockerfileContent: "",
			character:         2,
			items: []protocol.CompletionItem{
				{
					Detail:           types.CreateStringPointer("optional, string"),
					Kind:             types.CreateCompletionItemKindPointer(protocol.CompletionItemKindProperty),
					Label:            "description",
					InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 1, Character: 2},
							End:   protocol.Position{Line: 1, Character: 2},
						},
						NewText: "description = \"${1:value}\"",
					},
				},
				{
					Detail:           types.CreateStringPointer("optional, string"),
					Kind:             types.CreateCompletionItemKindPointer(protocol.CompletionItemKindProperty),
					Label:            "name",
					InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 1, Character: 2},
							End:   protocol.Position{Line: 1, Character: 2},
						},
						NewText: "name = \"${1:value}\"",
					},
				},
				{
					Detail:           types.CreateStringPointer("optional, list of string"),
					Kind:             types.CreateCompletionItemKindPointer(protocol.CompletionItemKindProperty),
					Label:            "targets",
					InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 1, Character: 2},
							End:   protocol.Position{Line: 1, Character: 2},
						},
						NewText: "targets = [ \"${1:value}\" ]",
					},
				},
			},
		},
		{
			name:              "target attribute in target block",
			content:           "target \"api\" {\n  target = \"\"\n}",
			line:              1,
			dockerfileContent: "",
			character:         12,
			items: []protocol.CompletionItem{
				{
					Label: "base",
				},
				{
					Label: "tests",
				},
				{
					Label: "release",
				},
			},
		},
		{
			name:              "target attribute in target block returns nothing for empty dockerfile-inline",
			content:           "target \"api\" {\n  target = \"\"\n  dockerfile-inline = \"FROM scratch\"\n}",
			line:              1,
			dockerfileContent: "",
			character:         12,
			items:             []protocol.CompletionItem{},
		},
		{
			name:              "args keys in target block",
			content:           "target \"api\" {\n  args = {\n    \"blah\" = \"\"\n  }\n}",
			line:              2,
			dockerfileContent: "",
			character:         5,
			items: []protocol.CompletionItem{
				{
					Label: "TARGETOS",
				},
				{
					Label: "TARGETARCH",
				},
				{
					Label: "argOne",
				},
				{
					Label: "argTwo",
				},
				{
					Label: "argOnePredefined",
				},
			},
		},
		{
			name:              "target attribute in target block returns content from manager (instead of what is on disk)",
			content:           "target \"api\" {\n  target = \"\"\n}",
			line:              1,
			dockerfileContent: "FROM scratch AS override",
			character:         12,
			items: []protocol.CompletionItem{
				{
					Label: "override",
				},
			},
		},
		{
			name:      "inherits attribute inside an empty array",
			content:   "target \"source\" {}\ntarget \"default\" {\n  inherits = [  ]\n}",
			line:      2,
			character: 15,
			items: []protocol.CompletionItem{
				{
					Label:      "source",
					InsertText: types.CreateStringPointer("\"source\""),
				},
				{
					Label:      "default",
					InsertText: types.CreateStringPointer("\"default\""),
				},
			},
		},
		{
			name:      "inherits attribute inside a quoted string within an array",
			content:   "target \"source\" {}\ntarget \"default\" {\n  inherits = [ \"\" ]\n}",
			line:      2,
			character: 16,
			items: []protocol.CompletionItem{
				{
					Label: "source",
				},
				{
					Label: "default",
				},
			},
		},
		{
			name:      "inherits attribute references a non-existent Dockerfile",
			content:   "target \"source\" {}\ntarget \"default\" {\n  inherits = [  ]\n  dockerfile = \"./Dockerfile-does-not-exist\"\n}",
			line:      2,
			character: 15,
			items: []protocol.CompletionItem{
				{
					Label:      "source",
					InsertText: types.CreateStringPointer("\"source\""),
				},
				{
					Label:      "default",
					InsertText: types.CreateStringPointer("\"default\""),
				},
			},
		},
		{
			name:      "inherits attribute within a non-target block",
			content:   "target2 \"default\" {\n  inherits = [ \"\" ]\n}",
			line:      1,
			character: 16,
			items:     []protocol.CompletionItem{},
		},
		{
			name:      "network attribute suggests default/host/none when there is no value",
			content:   "target \"t\" {\n  network = \n}",
			line:      1,
			character: 12,
			items: []protocol.CompletionItem{
				{
					Detail:           types.CreateStringPointer("string"),
					Label:            "default",
					InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 1, Character: 12},
							End:   protocol.Position{Line: 1, Character: 12},
						},
						NewText: "\"default\"",
					},
				},
				{
					Detail:           types.CreateStringPointer("string"),
					Label:            "host",
					InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 1, Character: 12},
							End:   protocol.Position{Line: 1, Character: 12},
						},
						NewText: "\"host\"",
					},
				},
				{
					Detail:           types.CreateStringPointer("string"),
					Label:            "none",
					InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 1, Character: 12},
							End:   protocol.Position{Line: 1, Character: 12},
						},
						NewText: "\"none\"",
					},
				},
			},
		},
		{
			name:      "network attribute suggests default/host/none when value is the empty string",
			content:   "target \"t\" {\n  network = \"\"\n}",
			line:      1,
			character: 13,
			items: []protocol.CompletionItem{
				{
					Label: "default",
				},
				{
					Label: "host",
				},
				{
					Label: "none",
				},
			},
		},
		{
			name:      "entitlements attribute suggests network.host and security.insecure",
			content:   "target \"t\" {\n  entitlements = [  ]\n}",
			line:      1,
			character: 19,
			items: []protocol.CompletionItem{
				{
					Detail:           types.CreateStringPointer("string"),
					Label:            "network.host",
					InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 1, Character: 19},
							End:   protocol.Position{Line: 1, Character: 19},
						},
						NewText: "\"network.host\"",
					},
				},
				{
					Detail:           types.CreateStringPointer("string"),
					Label:            "security.insecure",
					InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 1, Character: 19},
							End:   protocol.Position{Line: 1, Character: 19},
						},
						NewText: "\"security.insecure\"",
					},
				},
			},
		},
		{
			name:      "resolve build stage targets from a Dockerfile inside a context folder",
			content:   "target \"backend\" {\n  context = \"./backend\"\n  target=\"\"\n}",
			line:      2,
			character: 10,
			items: []protocol.CompletionItem{
				{
					Label: "nested",
				},
			},
		},
		{
			name:      "inherits attribute should not suggest anything in a variable block",
			content:   "variable \"var\" {\n  inherits = [\"\"]\n}\ntarget \"base\" {}",
			line:      1,
			character: 15,
			items:     []protocol.CompletionItem{},
		},
		{
			name:      "network attribute should not suggest anything in a variable block",
			content:   "variable \"var\" {\n  network = \"\"\n}\ntarget \"base\" {}",
			line:      1,
			character: 13,
			items:     []protocol.CompletionItem{},
		},
	}

	bakeFilePath := filepath.Join(completionTestFolderPath, "docker-bake.hcl")
	bakeFilePath = filepath.ToSlash(bakeFilePath)

	dockerfilePath := filepath.Join(completionTestFolderPath, "Dockerfile")
	dockerfilePath = filepath.ToSlash(dockerfilePath)

	bakeFileURI := uri.URI(fmt.Sprintf("file:///%v", strings.TrimPrefix(bakeFilePath, "/")))
	dockerfileURI := uri.URI(fmt.Sprintf("file:///%v", strings.TrimPrefix(dockerfilePath, "/")))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := document.NewDocumentManager()
			if tc.dockerfileContent != "" {
				changed, err := manager.Write(context.Background(), dockerfileURI, protocol.DockerfileLanguage, 1, []byte(tc.dockerfileContent))
				require.NoError(t, err)
				require.True(t, changed)
			}
			doc := document.NewBakeHCLDocument(bakeFileURI, 1, []byte(tc.content))
			list, err := Completion(context.Background(), &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: string(bakeFileURI)},
					Position:     protocol.Position{Line: tc.line, Character: tc.character},
				},
			}, manager, doc)
			require.NoError(t, err)
			require.False(t, list.IsIncomplete)
			require.Equal(t, tc.items, list.Items)
		})
	}
}

func TestCompletion_FileStructure(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		hideFiles bool
		line      uint32
		character uint32
	}{
		{
			name: "dockerfile attribute in a target block",
			content: `
target t1 {
	dockerfile = "%v"
}`,
			hideFiles: false,
			line:      2,
			character: 15,
		},
		{
			name: "context attribute in a target block",
			content: `
target t1 {
	context = "%v"
}`,
			hideFiles: true,
			line:      2,
			character: 12,
		},
		{
			name: "contexts attribute in a target block",
			content: `
target t1 {
	contexts = {
		src = "%v"
	}
}`,
			hideFiles: true,
			line:      3,
			character: 9,
		},
		{
			name: "contexts attribute in a target block with multiple contexts",
			content: `
target t1 {
	contexts = {
		src = "../path/to/source"
		src2 = "%v"
	}
}`,
			hideFiles: true,
			line:      4,
			character: 10,
		},
	}

	setups := []struct {
		description  string
		content      string
		offset       uint32
		result       *protocol.CompletionList
		folderResult *protocol.CompletionList
	}{
		{
			description: "no prefix",
			content:     "",
			offset:      0,
			result: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label: "a.txt",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFile),
					},
					{
						Label: "b",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
					},
					{
						Label: "folder",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
					},
				},
			},
			folderResult: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label: "b",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
					},
					{
						Label: "folder",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
					},
				},
			},
		},
		{
			description: "./ prefix",
			content:     "./",
			offset:      2,
			result: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label: "a.txt",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFile),
					},
					{
						Label: "b",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
					},
					{
						Label: "folder",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
					},
				},
			},
			folderResult: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label: "b",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
					},
					{
						Label: "folder",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder),
					},
				},
			},
		},
		{
			description: "./folder/ prefix",
			content:     "./folder/",
			offset:      9,
			result: &protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label: "subfile.txt",
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindFile),
					},
				},
			},
			folderResult: nil,
		},
	}

	dir := createFileStructure(t)
	bakeFileURI := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(dir, "docker-bake.hcl")), "/"))

	for _, tc := range testCases {
		for _, setup := range setups {
			t.Run(fmt.Sprintf("%v (%v)", tc.name, setup.description), func(t *testing.T) {
				manager := document.NewDocumentManager()
				doc := document.NewBakeHCLDocument(uri.URI(bakeFileURI), 1, []byte(fmt.Sprintf(tc.content, setup.content)))
				list, err := Completion(context.Background(), &protocol.CompletionParams{
					TextDocumentPositionParams: protocol.TextDocumentPositionParams{
						TextDocument: protocol.TextDocumentIdentifier{URI: bakeFileURI},
						Position:     protocol.Position{Line: tc.line, Character: tc.character + setup.offset},
					},
				}, manager, doc)
				require.NoError(t, err)
				if tc.hideFiles {
					require.Equal(t, setup.folderResult, list)
				} else {
					require.Equal(t, setup.result, list)
				}
			})
		}
	}
}

func TestCompletion_WSL(t *testing.T) {
	testCases := []struct {
		name              string
		content           string
		dockerfileContent string
		line              uint32
		character         uint32
		items             []protocol.CompletionItem
	}{
		{
			name:              "target attribute in target block",
			content:           "target \"api\" {\n  target = \"\"\n}",
			line:              1,
			dockerfileContent: "",
			character:         12,
			items: []protocol.CompletionItem{
				{
					Label: "base",
				},
				{
					Label: "tests",
				},
				{
					Label: "release",
				},
			},
		},
		{
			name:              "args keys in target block",
			content:           "target \"api\" {\n  args = {\n    \"blah\" = \"\"\n  }\n}",
			line:              2,
			dockerfileContent: "",
			character:         5,
			items: []protocol.CompletionItem{
				{
					Label: "TARGETOS",
				},
				{
					Label: "TARGETARCH",
				},
				{
					Label: "argOne",
				},
				{
					Label: "argTwo",
				},
				{
					Label: "argOnePredefined",
				},
			},
		},
		{
			name:      "resolve build stage targets from a Dockerfile inside a context folder",
			content:   "target \"backend\" {\n  context = \"./backend\"\n  target=\"\"\n}",
			line:      2,
			character: 10,
			items: []protocol.CompletionItem{
				{
					Label: "nested",
				},
			},
		},
	}

	wd, err := os.Getwd()
	require.NoError(t, err)
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(wd)))
	completionTestFolderPath := path.Join(projectRoot, "testdata", "completion")

	dockerfilePath := filepath.ToSlash(filepath.Join(completionTestFolderPath, "Dockerfile"))
	backendDockerfilePath := filepath.ToSlash(filepath.Join(completionTestFolderPath, "backend", "Dockerfile"))

	dockerfileBytes, err := os.ReadFile(dockerfilePath)
	require.NoError(t, err)
	backendDockerfileBytes, err := os.ReadFile(backendDockerfilePath)
	require.NoError(t, err)

	dockerfileURI := uri.URI("file://wsl%24/docker-desktop/tmp/Dockerfile")
	backendDockerfileURI := uri.URI("file://wsl%24/docker-desktop/tmp/backend/Dockerfile")
	bakeFileURI := uri.URI("file://wsl%24/docker-desktop/tmp/docker-bake.hcl")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := document.NewDocumentManager()
			changed, err := manager.Write(context.Background(), dockerfileURI, protocol.DockerfileLanguage, 1, dockerfileBytes)
			require.NoError(t, err)
			require.True(t, changed)
			changed, err = manager.Write(context.Background(), backendDockerfileURI, protocol.DockerfileLanguage, 1, backendDockerfileBytes)
			require.NoError(t, err)
			require.True(t, changed)

			doc := document.NewBakeHCLDocument(bakeFileURI, 1, []byte(tc.content))
			list, err := Completion(context.Background(), &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: string(bakeFileURI)},
					Position:     protocol.Position{Line: tc.line, Character: tc.character},
				},
			}, manager, doc)
			require.NoError(t, err)
			require.False(t, list.IsIncomplete)
			require.Equal(t, tc.items, list.Items)
		})
	}
}

func TestIsInsideRange(t *testing.T) {
	testCases := []struct {
		name     string
		hclRange hcl.Range
		position protocol.Position
		isInside bool
	}{
		{
			name:     "start.line < line && line < end.line",
			hclRange: hcl.Range{Start: hcl.Pos{Line: 1, Column: 5}, End: hcl.Pos{Line: 3, Column: 5}},
			position: protocol.Position{Line: 1, Character: 14}, // line 2 between lines 1 and 3
			isInside: true,
		},
		{
			name:     "start.line < line && line == end.line && character > end.column",
			hclRange: hcl.Range{Start: hcl.Pos{Line: 1, Column: 5}, End: hcl.Pos{Line: 3, Column: 5}},
			position: protocol.Position{Line: 2, Character: 14}, // character 14 is greater than 5
			isInside: false,
		},
		{
			name:     "start.line < line && line == end.line && character < end.column",
			hclRange: hcl.Range{Start: hcl.Pos{Line: 1, Column: 5}, End: hcl.Pos{Line: 3, Column: 5}},
			position: protocol.Position{Line: 2, Character: 2}, // 2-2 is less than 2-5
			isInside: true,
		},
		{
			name:     "start.line < line && end.line < line",
			hclRange: hcl.Range{Start: hcl.Pos{Line: 1, Column: 5}, End: hcl.Pos{Line: 3, Column: 5}},
			position: protocol.Position{Line: 100, Character: 2}, // line is way over
			isInside: false,
		},
		{
			name:     "start.line == line && line < end.line && start.character < character && character > end.character",
			hclRange: hcl.Range{Start: hcl.Pos{Line: 1, Column: 5}, End: hcl.Pos{Line: 3, Column: 5}},
			position: protocol.Position{Line: 0, Character: 10}, // 1-10 is within 1-5 and 3-5
			isInside: true,
		},
		{
			name:     "start.line == line && line < end.line && start.character > character && character > end.character",
			hclRange: hcl.Range{Start: hcl.Pos{Line: 1, Column: 15}, End: hcl.Pos{Line: 3, Column: 5}},
			position: protocol.Position{Line: 0, Character: 10}, // 1-10 is before 1-10
			isInside: false,
		},
		{
			name:     "start.line > line",
			hclRange: hcl.Range{Start: hcl.Pos{Line: 2, Column: 0}, End: hcl.Pos{Line: 2, Column: 5}},
			position: protocol.Position{Line: 0, Character: 0}, // 1-0 is before 2-0
			isInside: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.isInside, isInsideRange(tc.hclRange, tc.position))
		})
	}
}

func createFileStructure(t *testing.T) string {
	dir, err := os.MkdirTemp(os.TempDir(), fmt.Sprintf("%v-%v", t.Name(), carbon.Now().TimestampMilli()))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(dir))
	})

	fileStructure := []struct {
		name  string
		isDir bool
	}{
		{name: "a.txt", isDir: false},
		{name: "b", isDir: true},
		{name: "folder", isDir: true},
		{name: "folder/subfile.txt", isDir: false},
	}
	for _, entry := range fileStructure {
		if entry.isDir {
			require.NoError(t, os.Mkdir(filepath.Join(dir, entry.name), 0755))
		} else {
			f, err := os.Create(filepath.Join(dir, entry.name))
			require.NoError(t, err)
			require.NoError(t, f.Close())
		}
	}
	return dir
}
