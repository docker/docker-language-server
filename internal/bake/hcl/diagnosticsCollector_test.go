package hcl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/scout"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/uri"
)

func TestCollectDiagnostics(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		diagnostics []protocol.Diagnostic
	}{
		{
			name:    "target block with no name",
			content: "target {\n}",
			diagnostics: []protocol.Diagnostic{
				{
					Message:  "Missing name for target (All target blocks must have 1 labels (name).)",
					Source:   types.CreateStringPointer("docker-language-server"),
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 7},
						End:   protocol.Position{Line: 0, Character: 8},
					},
				},
			},
		},
		{
			name:    "target block with missing attribute value",
			content: "target {\n  dockerfile = \n}",
			diagnostics: []protocol.Diagnostic{
				{
					Message:  "Invalid expression (Expected the start of an expression, but found an invalid expression token.)",
					Source:   types.CreateStringPointer("docker-language-server"),
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 15},
						End:   protocol.Position{Line: 2, Character: 0},
					},
				},
			},
		},
		{
			name:    "target block with alpine:3.17.0 is flagged with vulnerabilities",
			content: "target \"t1\" {\n  tags = [ \"alpine:3.17.0\" ]\n}",
			diagnostics: []protocol.Diagnostic{
				{
					Message:  "The image contains 1 critical and 7 high vulnerabilities",
					Source:   types.CreateStringPointer("docker-language-server"),
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityWarning),
					Code:     &protocol.IntegerOrString{Value: "critical_high_vulnerabilities"},
					CodeDescription: &protocol.CodeDescription{
						HRef: "https://hub.docker.com/layers/library/alpine/3.17.0/images/sha256-c0d488a800e4127c334ad20d61d7bc21b4097540327217dfab52262adc02380c",
					},
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 12},
						End:   protocol.Position{Line: 1, Character: 25},
					},
				},
			},
		},
		{
			name:    "target block with network attribute empty string",
			content: "target \"t1\" {\n  network = \"\"\n}",
			diagnostics: []protocol.Diagnostic{
				{
					Message:  "network attribute must be either: default, host, or none",
					Source:   types.CreateStringPointer("docker-language-server"),
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 12},
						End:   protocol.Position{Line: 1, Character: 14},
					},
				},
			},
		},
		{
			name:    "target block with entitlements attribute empty string",
			content: "target \"t1\" {\n  entitlements = [ \"\" ]\n}",
			diagnostics: []protocol.Diagnostic{
				{
					Message:  "entitlements attribute must be either: network.host or security.insecure",
					Source:   types.CreateStringPointer("docker-language-server"),
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 19},
						End:   protocol.Position{Line: 1, Character: 21},
					},
				},
			},
		},
		{
			name:        "args can be found in Dockerfile (unquoted)",
			content:     "target \"t1\" {\n  args = {\n    valid = \"value\"\n  }\n}",
			diagnostics: []protocol.Diagnostic{},
		},
		{
			name:        "args can be found in Dockerfile (quoted)",
			content:     "target \"t1\" {\n  args = {\n    \"valid\" = \"value\"\n  }\n}",
			diagnostics: []protocol.Diagnostic{},
		},
		{
			name:    "args cannot be found in Dockerfile",
			content: "target \"t1\" {\n  args = {\n    missing = \"value\"\n  }\n}",
			diagnostics: []protocol.Diagnostic{
				{
					Message:  "'missing' not defined as an ARG in your Dockerfile",
					Source:   types.CreateStringPointer("docker-language-server"),
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 4},
						End:   protocol.Position{Line: 2, Character: 11},
					},
				},
			},
		},
		{
			name:        "args resolution to a dockerfile that points to a valid variable",
			content:     "variable var { default = \"./backend/Dockerfile\" }\ntarget \"t1\" {\n  dockerfile = var\nargs = {\n    BACKEND_VAR = \"newValue\"\n  }\n}",
			diagnostics: []protocol.Diagnostic{},
		},
		{
			name:    "args resolution to a dockerfile that points to a valid variable",
			content: "variable var { default = \"./backend/Dockerfile\" }\ntarget \"t1\" {\n  dockerfile = var\nargs = {\n    missing = \"newValue\"\n  }\n}",

			diagnostics: []protocol.Diagnostic{
				{
					Message:  "'missing' not defined as an ARG in your Dockerfile",
					Source:   types.CreateStringPointer("docker-language-server"),
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
					Range: protocol.Range{
						Start: protocol.Position{Line: 4, Character: 4},
						End:   protocol.Position{Line: 4, Character: 11},
					},
				},
			},
		},
		{
			name:        "args resolution when inheriting a parent that points to a var",
			content:     "variable var { }\ntarget \"t1\" {\n  inherits = [var]\nargs = {\n    missing = \"value\"\n  }\n}",
			diagnostics: []protocol.Diagnostic{},
		},
		{
			name:        "args resolution when inheriting a parent that points to a ${var}",
			content:     "variable var { }\ntarget \"t1\" {\n  inherits = [\"${var}\"]\nargs = {\n    missing = \"value\"\n  }\n}",
			diagnostics: []protocol.Diagnostic{},
		},
		{
			name:        "args references built-in args",
			content:     "target \"t1\" {\n  args = {\n    HTTP_PROXY = \"\"\n    HTTPS_PROXY = \"\"\n    FTP_PROXY = \"\"\n    NO_PROXY = \"\"\n    ALL_PROXY = \"\"\n    BUILDKIT_SYNTAX = \"\"\n  }\n}",
			diagnostics: []protocol.Diagnostic{},
		},
		{
			name:    "target cannot be found in Dockerfile",
			content: "target \"t1\" {\n  target = \"nonexistent\"\n}",
			diagnostics: []protocol.Diagnostic{
				{
					Message:  "target could not be found in your Dockerfile",
					Source:   types.CreateStringPointer("docker-language-server"),
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 11},
						End:   protocol.Position{Line: 1, Character: 24},
					},
				},
			},
		},
		{
			name:        "ignore args if dockerfile-inline is used",
			content:     "target \"t1\" {\n  dockerfile-inline = \"\"\n  args = {\n    missing = \"value\"\n  }\n}",
			diagnostics: []protocol.Diagnostic{},
		},
		{
			name:    "dockerfile attribute flagged as unnecessary if dockerfile-inline exists",
			content: "target \"t1\" {\n  dockerfile = \"./Dockerfile\"\n  dockerfile-inline = \"FROM scratch\"\n}",
			diagnostics: []protocol.Diagnostic{
				{
					Message:  "dockerfile attribute is ignored if dockerfile-inline is defined",
					Source:   types.CreateStringPointer("docker-language-server"),
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityWarning),
					Tags:     []protocol.DiagnosticTag{protocol.DiagnosticTagUnnecessary},
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 2},
						End:   protocol.Position{Line: 1, Character: 29},
					},
					Data: []types.NamedEdit{
						{
							Title: "Remove unnecessary dockerfile attribute",
							Edit:  "",
							Range: &protocol.Range{
								Start: protocol.Position{Line: 1},
								End:   protocol.Position{Line: 2},
							},
						},
					},
				},
			},
		},
		{
			name: "target inheritance checked when looking at the target attribute",
			content: `target "lint" {
  target = "build"
  dockerfile = "./Dockerfile2"
}

target "lint2" {
  inherits = ["lint"]
  target = "build"
}`,
			diagnostics: []protocol.Diagnostic{},
		},
		{
			name:        "target inheritance references non-existing parent target",
			content:     "target \"child\" {\n  inherits = [\"parent\"]\n  target = \"build\"\n}",
			diagnostics: []protocol.Diagnostic{},
		},
		{
			name:        "context folder used for looking for the Dockerfile",
			content:     "target \"backend\" {\n  context = \"./backend\"\n  args = {\n    BACKEND_VAR=\"changed\"\n  }\n}",
			diagnostics: []protocol.Diagnostic{},
		},
		{
			name: "context has a variable and the referenced ARG is valid",
			content: `
variable "VAR" {
  default = "./backend"
}

target "build" {
  context = "${VAR}"
  dockerfile = "Dockerfile"
  args = {
    BACKEND_VAR = "newValue"
  }
}`,
			diagnostics: []protocol.Diagnostic{},
		},
		{
			name: "context has a variable and the referenced ARG is invalid",
			content: `
variable "VAR" {
  default = "./backend"
}

target "build" {
  context = "${VAR}"
  dockerfile = "Dockerfile"
  args = {
    NON_EXISTENT_VAR = "newValue"
  }
}`,
			diagnostics: []protocol.Diagnostic{
				{
					Message:  "'NON_EXISTENT_VAR' not defined as an ARG in your Dockerfile",
					Source:   types.CreateStringPointer("docker-language-server"),
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
					Range: protocol.Range{
						Start: protocol.Position{Line: 9, Character: 4},
						End:   protocol.Position{Line: 9, Character: 20},
					},
				},
			},
		},
		{
			name: "context has a variable and the target is inherited",
			content: `
variable "VAR" {
  default = "."
}

target "common-base" {
  context = "${VAR}/folder/subfolder"
  dockerfile = "Dockerfile"
}

target "build" {
  inherits = ["common-base"]
  args = {
    VAR = "value"
  }
}`,
			diagnostics: []protocol.Diagnostic{},
		},
		{
			name: "parent target cannot be resolved but local target is resolvable",
			content: `
variable "VAR" {
  default = "."
}

target "common-base" {
  dockerfile = "${VAR}/folder/subfolder/Dockerfile"
}

target "build" {
  inherits = ["common-base"]
  dockerfile = "Dockerfile.non-existent-file"
  args = {
    VAR = "value"
  }
}`,
			diagnostics: []protocol.Diagnostic{
				{
					Message:  "'VAR' not defined as an ARG in your Dockerfile",
					Source:   types.CreateStringPointer("docker-language-server"),
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
					Range: protocol.Range{
						Start: protocol.Position{Line: 13, Character: 4},
						End:   protocol.Position{Line: 13, Character: 7},
					},
				}},
		},
		{
			name: "malformed variable interpolation has the right line numbers",
			content: `target x {
  tags = ["${var"]
  dockerfile = "./Dockerfile"
}`,
			diagnostics: []protocol.Diagnostic{
				{
					Message:  "Invalid multi-line string (Quoted strings may not be split over multiple lines. To produce a multi-line string, either use the \\n escape to represent a newline character or use the \"heredoc\" multi-line template syntax.)",
					Source:   types.CreateStringPointer("docker-language-server"),
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 18},
						End:   protocol.Position{Line: 2, Character: 0},
					},
				},
				{
					Message:  "Invalid multi-line string (Quoted strings may not be split over multiple lines. To produce a multi-line string, either use the \\n escape to represent a newline character or use the \"heredoc\" multi-line template syntax.)",
					Source:   types.CreateStringPointer("docker-language-server"),
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 29},
						End:   protocol.Position{Line: 3, Character: 0},
					},
				},
				{
					Message:  "Unclosed template interpolation sequence (There is no closing brace for this interpolation sequence before the end of the quoted template. This might be caused by incorrect nesting inside the given expression.)",
					Source:   types.CreateStringPointer("docker-language-server"),
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 11},
						End:   protocol.Position{Line: 1, Character: 13},
					},
				},
			},
		},
		{
			name: "annotations attribute has no equals sign",
			content: `
target "t1" {
  annotations = [ "abc" ]
}`,
			diagnostics: []protocol.Diagnostic{
				{
					Message:  `invalid annotation "abc", expected key=value`,
					Source:   types.CreateStringPointer("docker-language-server"),
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 18},
						End:   protocol.Position{Line: 2, Character: 23},
					},
				},
			},
		},
	}

	wd, err := os.Getwd()
	require.NoError(t, err)
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(wd)))
	diagnosticsTestFolderPath := filepath.Join(projectRoot, "testdata", "diagnostics")
	bakeFilePath := filepath.Join(diagnosticsTestFolderPath, "docker-bake.hcl")
	bakeFileURI := uri.URI(fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(bakeFilePath), "/")))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := document.NewDocumentManager()
			bytes := []byte(tc.content)
			collector := &BakeHCLDiagnosticsCollector{docs: manager, scout: scout.NewService()}
			doc := document.NewBakeHCLDocument(bakeFileURI, 1, bytes)
			diagnostics := collector.CollectDiagnostics("docker-language-server", "", doc, "")
			require.Equal(t, tc.diagnostics, diagnostics)
		})
	}
}

func TestCollectDiagnostics_WSL(t *testing.T) {
	testCases := []struct {
		name              string
		content           string
		dockerfileContent string
		diagnostics       []protocol.Diagnostic
	}{
		{
			name:              "target found in Dockerfile",
			content:           "target \"t1\" {\n  target = \"base\"\n}",
			dockerfileContent: "FROM scratch AS base",
			diagnostics:       []protocol.Diagnostic{},
		},
		{
			name:              "target cannot be found in Dockerfile",
			content:           "target \"t1\" {\n  target = \"nonexistent\"\n}",
			dockerfileContent: "FROM scratch AS base",
			diagnostics: []protocol.Diagnostic{
				{
					Message:  "target could not be found in your Dockerfile",
					Source:   types.CreateStringPointer("docker-language-server"),
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 11},
						End:   protocol.Position{Line: 1, Character: 24},
					},
				},
			},
		},
		{
			name:              "args can be found in Dockerfile",
			content:           "target \"t1\" {\n  args = {\n    VAR = \"newValue\"\n  }\n}",
			dockerfileContent: "ARG VAR=value\nFROM scratch",
			diagnostics:       []protocol.Diagnostic{},
		},
		{
			name:              "args cannot be found in Dockerfile",
			content:           "target \"t1\" {\n  args = {\n    missing = \"newValue\"\n  }\n}",
			dockerfileContent: "ARG VAR=value\nFROM scratch",
			diagnostics: []protocol.Diagnostic{
				{
					Message:  "'missing' not defined as an ARG in your Dockerfile",
					Source:   types.CreateStringPointer("docker-language-server"),
					Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 4},
						End:   protocol.Position{Line: 2, Character: 11},
					},
				},
			},
		},
	}

	dockerfileURI := "file://wsl%24/docker-desktop/tmp/Dockerfile"
	bakeFileURI := uri.URI("file://wsl%24/docker-desktop/tmp/docker-bake.hcl")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := document.NewDocumentManager()
			changed, err := manager.Write(context.Background(), uri.URI(dockerfileURI), protocol.DockerfileLanguage, 1, []byte(tc.dockerfileContent))
			require.NoError(t, err)
			require.True(t, changed)
			collector := &BakeHCLDiagnosticsCollector{docs: manager, scout: scout.NewService()}
			doc := document.NewBakeHCLDocument(bakeFileURI, 1, []byte(tc.content))
			diagnostics := collector.CollectDiagnostics("docker-language-server", "", doc, "")
			require.Equal(t, tc.diagnostics, diagnostics)
		})
	}
}

var hierarchyTests = []struct {
	name        string
	content     string
	diagnostics []protocol.Diagnostic
}{
	{
		name: "child target's Dockerfile defines the ARG",
		content: `
target parent {
  args = {
    other = "value2"
  }
}

target foo {
  dockerfile = "Dockerfile2"
  inherits = ["parent"]
}`,
		diagnostics: []protocol.Diagnostic{},
	},
	{
		name: "child target's Dockerfile defines the ARG but not another child",
		content: `
target parent {
  args = {
    other = "value2"
  }
}

target foo {
  dockerfile = "Dockerfile2"
  inherits = ["parent"]
}

target foo2 {
  inherits = ["parent"]
}`,
		diagnostics: []protocol.Diagnostic{},
	},
}

func TestCollectDiagnostics_Hierarchy(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(wd)))
	diagnosticsTestFolderPath := filepath.Join(projectRoot, "testdata", "diagnostics")
	bakeFilePath := filepath.Join(diagnosticsTestFolderPath, "docker-bake.hcl")
	bakeFileURI := uri.URI(fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(bakeFilePath), "/")))

	for _, tc := range hierarchyTests {
		t.Run(tc.name, func(t *testing.T) {
			manager := document.NewDocumentManager()
			bytes := []byte(tc.content)
			collector := &BakeHCLDiagnosticsCollector{docs: manager, scout: scout.NewService()}
			doc := document.NewBakeHCLDocument(bakeFileURI, 1, bytes)
			diagnostics := collector.CollectDiagnostics("docker-language-server", "", doc, "")
			require.Equal(t, tc.diagnostics, diagnostics)
		})
	}
}

func TestCollectDiagnostics_WSLHierarchy(t *testing.T) {
	dockerfileContent := "ARG other=value\nFROM scratch AS build"
	dockerfileURI := uri.URI("file://wsl%24/docker-desktop/tmp/Dockerfile2")
	bakeFileURI := uri.URI("file://wsl%24/docker-desktop/tmp/docker-bake.hcl")

	for _, tc := range hierarchyTests {
		t.Run(tc.name, func(t *testing.T) {
			manager := document.NewDocumentManager()
			changed, err := manager.Write(context.Background(), dockerfileURI, protocol.DockerfileLanguage, 1, []byte(dockerfileContent))
			require.NoError(t, err)
			require.True(t, changed)
			bytes := []byte(tc.content)
			collector := &BakeHCLDiagnosticsCollector{docs: manager, scout: scout.NewService()}
			doc := document.NewBakeHCLDocument(bakeFileURI, 1, bytes)
			diagnostics := collector.CollectDiagnostics("docker-language-server", "", doc, "")
			require.Equal(t, tc.diagnostics, diagnostics)
		})
	}
}
