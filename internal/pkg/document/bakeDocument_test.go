package document

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/uri"
)

func TestParentTargets(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		target   string
		targets  []string
		resolved bool
	}{
		{
			name: "empty block",
			content: `
target t1 {}`,
			target:   "t1",
			targets:  nil,
			resolved: true,
		},
		{
			name: "inheriting non-existent target",
			content: `
target t1 { inherits = ["t2"]}`,
			target:   "t1",
			targets:  nil,
			resolved: false,
		},
		{
			name: "inheriting valid quoted target",
			content: `
target t1 { inherits = ["t2"]}
target "t2" {}`,
			target:   "t1",
			targets:  []string{"t2"},
			resolved: true,
		},
		{
			name: "inheriting valid target",
			content: `
target t1 { inherits = ["t2"]}
target t2 {}`,
			target:   "t1",
			targets:  []string{"t2"},
			resolved: true,
		},
		{
			name: "two-way recursion",
			content: `
target t1 { inherits = ["t2"]}
target t2 { inherits = ["t1"]}`,
			target:   "t1",
			targets:  nil,
			resolved: false,
		},
		{
			name: "three-way recursion",
			content: `
target t1 { inherits = ["t2"]}
target t2 { inherits = ["t3"]}
target t3 { inherits = ["t1"]}`,
			target:   "t1",
			targets:  nil,
			resolved: false,
		},
		{
			name: "two-level inheritance",
			content: `
target t1 { inherits = ["t2", "t3"]}
target t2 { inherits = ["t4"]}
target t3 { inherits = ["t5"]}
target t4 { }
target t5 { }`,
			target:   "t1",
			targets:  []string{"t5", "t2", "t4", "t3"},
			resolved: true,
		},
		{
			name: "variables are not resolved",
			content: `
target t1 { inherits = ["t2", var]}
target t2 { }`,
			target:   "t1",
			targets:  nil,
			resolved: false,
		},
		{
			name: "${variables} are not resolved",
			content: `
target t1 { inherits = ["t2", "${var}"]}
target t2 { }`,
			target:   "t1",
			targets:  nil,
			resolved: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := NewBakeHCLDocument("file:///tmp/docker-bake.hcl", 1, []byte(tc.content))
			targets, resolved := doc.ParentTargets(tc.target)
			require.Equal(t, tc.targets, targets)
			require.Equal(t, tc.resolved, resolved)
		})
	}
}

func TestDockerfileForTarget(t *testing.T) {
	root := os.TempDir()
	tmp := filepath.Join(root, "tmp")
	tmp2 := filepath.Join(tmp, "tmp2")
	temporaryBakeFile := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(tmp2, "docker-bake.hcl")), "/"))

	testCases := []struct {
		name        string
		documentURI string
		content     string
		uri         string
		path        string
		err         error
	}{
		{
			name:    "empty block with no label name",
			content: `target {}`,
			err:     errors.New("cannot parse Bake file"),
		},
		{
			name: "group block before target block",
			content: `group g1 { targets = [ "t1" ] }
			target t1 {}`,
			err: errors.New("no target block named g1"),
		},
		{
			name:    "dockerfile set",
			content: `target t1 { dockerfile = "Dockerfile2" }`,
			uri:     fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(tmp2, "Dockerfile2")), "/")),
			path:    filepath.Join(tmp2, "Dockerfile2"),
		},
		{
			name: "dockerfile set to variable",
			content: `
				target "stage1" {
					dockerfile = var
				}
				variable "var" {
					default = "Dockerfile2"
				}`,
			uri:  fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(tmp2, "Dockerfile2")), "/")),
			path: filepath.Join(tmp2, "Dockerfile2"),
		},
		{
			name:    "dockerfile-inline set",
			content: `target t1 { dockerfile-inline = "FROM scratch" }`,
			err:     errors.New("dockerfile-inline defined"),
		},
		{
			name:    "context set to subfolder",
			content: `target t1 { context = "subfolder" }`,
			uri:     fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(tmp2, "subfolder", "Dockerfile")), "/")),
			path:    filepath.Join(tmp2, "subfolder", "Dockerfile"),
		},
		{
			name:    "context set to ../other",
			content: `target t1 { context = "../other" }`,
			uri:     fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(tmp, "other", "Dockerfile")), "/")),
			path:    filepath.Join(tmp, "other", "Dockerfile"),
		},
		{
			name: "context set to subfolder and dockerfile defined",
			content: `target t1 {
				context = "subfolder"
				dockerfile = "Dockerfile2"
			}`,
			uri:  fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(tmp2, "subfolder", "Dockerfile2")), "/")),
			path: filepath.Join(tmp2, "subfolder", "Dockerfile2"),
		},
		{
			name:        "wsl$ with dockerfile set",
			documentURI: "file://wsl%24/docker-desktop/tmp/tmp2/docker-bake.hcl",
			content:     `target t1 { dockerfile = "Dockerfile2" }`,
			uri:         "file://wsl%24/docker-desktop/tmp/tmp2/Dockerfile2",
			path:        "\\\\wsl$\\docker-desktop\\tmp\\tmp2\\Dockerfile2",
		},
		{
			name:        "wsl$ with context set",
			documentURI: "file://wsl%24/docker-desktop/tmp/tmp2/docker-bake.hcl",
			content:     `target t1 { context = "subfolder" }`,
			uri:         "file://wsl%24/docker-desktop/tmp/tmp2/subfolder/Dockerfile",
			path:        "\\\\wsl$\\docker-desktop\\tmp\\tmp2\\subfolder\\Dockerfile",
		},
		{
			name:        "wsl$ with context set to ../other",
			documentURI: "file://wsl%24/docker-desktop/tmp/tmp2/docker-bake.hcl",
			content:     `target t1 { context = "../other" }`,
			uri:         "file://wsl%24/docker-desktop/tmp/other/Dockerfile",
			path:        "\\\\wsl$\\docker-desktop\\tmp\\other\\Dockerfile",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			documentURI := tc.documentURI
			if tc.documentURI == "" {
				documentURI = temporaryBakeFile
			}
			doc := NewBakeHCLDocument(uri.URI(documentURI), 1, []byte(tc.content))
			body, ok := doc.File().Body.(*hclsyntax.Body)
			require.True(t, ok)
			uri, path, err := doc.DockerfileForTarget(body.Blocks[0])
			require.Equal(t, tc.err, err)
			require.Equal(t, tc.uri, uri)
			require.Equal(t, tc.path, path)
		})
	}
}

func TestNewBakeHCLDocument(t *testing.T) {
	temporaryBakeFile := fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(t.TempDir(), "docker-bake.hcl")), "/"))
	temporaryBakeFileURI := uri.URI(temporaryBakeFile)

	testCases := []struct {
		name    string
		content string
	}{
		{
			name: "reference typed variable with no value",
			content: `variable "CI" {
  type = string
}

variable "foo" {
  type = bool
  default = CI
}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := NewBakeHCLDocument(temporaryBakeFileURI, 1, []byte(tc.content))
			_, ok := doc.File().Body.(*hclsyntax.Body)
			require.True(t, ok)
		})
	}
}
