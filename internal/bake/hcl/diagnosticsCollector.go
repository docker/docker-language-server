package hcl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/docker/buildx/bake"
	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/pkg/lsp/textdocument"
	"github.com/docker/docker-language-server/internal/scout"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/moby/buildkit/solver/errdefs"
)

type BakePrintOutput struct {
	Group  map[string]bake.Group  `json:"group,omitempty"`
	Target map[string]bake.Target `json:"target"`
}

var builtinArgs = []string{
	"HTTP_PROXY",
	"HTTPS_PROXY",
	"FTP_PROXY",
	"ALL_PROXY",
	"NO_PROXY",
	"BUILDKIT_CACHE_MOUNT_NS",
	"BUILDKIT_MULTI_PLATFORM",
	"BUILDKIT_SANDBOX_HOSTNAME",
	"BUILDKIT_DOCKERFILE_CHECK",
	"BUILDKIT_CONTEXT_KEEP_GIT_DIR",
	"SOURCE_DATE_EPOCH",
}

type BakeHCLDiagnosticsCollector struct {
	docs  *document.Manager
	scout scout.Service
}

func NewBakeHCLDiagnosticsCollector(docs *document.Manager, scout scout.Service) textdocument.DiagnosticsCollector {
	return &BakeHCLDiagnosticsCollector{docs: docs, scout: scout}
}

func UnwrapToHCL(err error) hcl.Diagnostics {
	if diag, ok := err.(hcl.Diagnostics); ok {
		return diag
	}
	err = errors.Unwrap(err)
	if diag, ok := err.(hcl.Diagnostics); ok {
		return diag
	}
	return nil
}

func (c *BakeHCLDiagnosticsCollector) SupportsLanguageIdentifier(languageIdentifier protocol.LanguageIdentifier) bool {
	return languageIdentifier == protocol.DockerBakeLanguage
}

func (c *BakeHCLDiagnosticsCollector) CollectDiagnostics(source, workspaceFolder string, doc document.Document, text string) []protocol.Diagnostic {
	input := doc.Input()
	_, err := bake.ParseFile(input, doc.URI().Filename())
	diagnostics := []protocol.Diagnostic{}
	if err != nil {
		var errorSource *errdefs.ErrorSource
		var sourceRange *protocol.Range
		if errors.As(err, &errorSource) && len(errorSource.Ranges) > 0 {
			lines := strings.Split(string(input), "\n")
			sourceRange = &protocol.Range{
				Start: protocol.Position{
					Line:      uint32(errorSource.Ranges[0].Start.Line) - 1,
					Character: 0,
				},
				End: protocol.Position{
					Line:      uint32(errorSource.Ranges[0].Start.Line) - 1,
					Character: uint32(len(lines[errorSource.Ranges[0].Start.Line-1])),
				},
			}
		}

		hclDiagnostics := UnwrapToHCL(err)
		for _, hclDiagnostic := range hclDiagnostics {
			diagnostic := protocol.Diagnostic{
				Message:  fmt.Sprintf("%v (%v)", hclDiagnostic.Summary, hclDiagnostic.Detail),
				Source:   types.CreateStringPointer(source),
				Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
			}

			if hclDiagnostic.Context == nil {
				if sourceRange != nil {
					diagnostic.Range = *sourceRange
				}
			} else {
				diagnostic.Range = protocol.Range{
					Start: protocol.Position{
						Line:      uint32(hclDiagnostic.Context.Start.Line) - 1,
						Character: uint32(hclDiagnostic.Context.Start.Column) - 1,
					},
					End: protocol.Position{
						Line:      uint32(hclDiagnostic.Context.End.Line) - 1,
						Character: uint32(hclDiagnostic.Context.End.Column) - 1,
					},
				}
			}

			diagnostics = append(diagnostics, diagnostic)
		}
	}

	body, ok := doc.(document.BakeHCLDocument).File().Body.(*hclsyntax.Body)
	if !ok {
		return diagnostics
	}

	for _, block := range body.Blocks {
		if block.Type == "target" {
			if _, ok := block.Body.Attributes["dockerfile-inline"]; ok {
				if attribute, ok := block.Body.Attributes["dockerfile"]; ok {
					diagnostics = append(diagnostics, protocol.Diagnostic{
						Message:  "dockerfile attribute is ignored if dockerfile-inline is defined",
						Source:   types.CreateStringPointer(source),
						Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityWarning),
						Tags:     []protocol.DiagnosticTag{protocol.DiagnosticTagUnnecessary},
						Range:    createProtocolRange(attribute.SrcRange, false),
						Data: []types.NamedEdit{
							{
								Title: "Remove unnecessary dockerfile attribute",
								Edit:  "",
								Range: &protocol.Range{
									Start: protocol.Position{Line: uint32(attribute.SrcRange.Start.Line - 1)},
									End:   protocol.Position{Line: uint32(attribute.SrcRange.Start.Line)},
								},
							},
						},
					})
				}
			}

			if attribute, ok := block.Body.Attributes["tags"]; ok {
				if expr, ok := attribute.Expr.(*hclsyntax.TupleConsExpr); ok {
					for _, e := range expr.Exprs {
						if templateExpr, ok := e.(*hclsyntax.TemplateExpr); ok {
							if templateExpr.IsStringLiteral() {
								value, _ := templateExpr.Value(&hcl.EvalContext{})
								target := value.AsString()
								imageDiagnostics, err := c.scout.Analyze(target)
								if err == nil {
									for _, diagnostic := range imageDiagnostics {
										if diagnostic.Kind == "critical_high_vulnerabilities" || diagnostic.Kind == "vulnerabilities" {
											rng := templateExpr.SrcRange
											diagnostics = append(diagnostics, scout.ConvertDiagnostic(diagnostic, nil, source, createProtocolRange(rng, true), nil))
											break
										}
									}
								}
							}
						}
					}
				}
			}

			if attribute, ok := block.Body.Attributes["entitlements"]; ok {
				if tupleConsExpr, ok := attribute.Expr.(*hclsyntax.TupleConsExpr); ok {
					for _, e := range tupleConsExpr.Exprs {
						if templateExpr, ok := e.(*hclsyntax.TemplateExpr); ok {
							if templateExpr.IsStringLiteral() {
								value, _ := templateExpr.Value(&hcl.EvalContext{})
								diagnostic := checkStringLiteral(
									source,
									value.AsString(),
									"entitlements attribute must be either: network.host or security.insecure",
									[]string{"network.host", "security.insecure"},
									templateExpr.SrcRange,
								)

								if diagnostic != nil {
									diagnostics = append(diagnostics, *diagnostic)
								}
							}
						}
					}
				}
			}

			if attribute, ok := block.Body.Attributes["network"]; ok {
				if templateExpr, ok := attribute.Expr.(*hclsyntax.TemplateExpr); ok {
					if templateExpr.IsStringLiteral() {
						value, _ := templateExpr.Value(&hcl.EvalContext{})
						diagnostic := checkStringLiteral(
							source,
							value.AsString(),
							"network attribute must be either: default, host, or none",
							[]string{"default", "host", "none"},
							templateExpr.SrcRange,
						)

						if diagnostic != nil {
							diagnostics = append(diagnostics, *diagnostic)
						}
					}
				}
			}

			dockerfilePath, err := EvaluateDockerfilePath(block, doc)
			if dockerfilePath == "" || err != nil {
				continue
			}

			if attribute, ok := block.Body.Attributes["target"]; ok {
				if expr, ok := attribute.Expr.(*hclsyntax.TemplateExpr); ok && len(expr.Parts) == 1 {
					if literalValueExpr, ok := expr.Parts[0].(*hclsyntax.LiteralValueExpr); ok {
						diagnostic := c.checkTargetTarget(dockerfilePath, expr, literalValueExpr, source)
						if diagnostic != nil {
							diagnostics = append(diagnostics, *diagnostic)
						}
					}
				}
			}

			if attribute, ok := block.Body.Attributes["args"]; ok {
				if expr, ok := attribute.Expr.(*hclsyntax.ObjectConsExpr); ok {
					argsDiagnostics := c.checkTargetArgs(dockerfilePath, input, expr, source)
					diagnostics = append(diagnostics, argsDiagnostics...)
				}
			}
		}
	}
	return diagnostics
}

// EvaluateDockerfilePath uses the output of `docker buildx bake --print`
// to identify the location of the Dockerfile that block is using.
func EvaluateDockerfilePath(block *hclsyntax.Block, doc document.Document) (string, error) {
	if len(block.Labels) == 0 {
		// if the target block has no label we cannot ask Bake to try and print it
		return "", errors.New("target block has no label")
	}

	if _, ok := block.Body.Attributes["target"]; ok {
		return ParseDockerfileFromBakeOutput(doc, block.Labels[0])
	}

	if _, ok := block.Body.Attributes["args"]; ok {
		return ParseDockerfileFromBakeOutput(doc, block.Labels[0])
	}
	return "", nil
}

// checkTargetArgs examines the args attribute of a target block.
func (c *BakeHCLDiagnosticsCollector) checkTargetArgs(dockerfilePath string, input []byte, expr *hclsyntax.ObjectConsExpr, source string) []protocol.Diagnostic {
	_, nodes := OpenDockerfile(context.Background(), c.docs, dockerfilePath)
	args := []string{}
	for _, child := range nodes {
		if strings.EqualFold(child.Value, "ARG") {
			child = child.Next
			for child != nil {
				value := child.Value
				idx := strings.Index(value, "=")
				if idx != -1 {
					value = value[:idx]
				}
				args = append(args, value)
				child = child.Next
			}
		}
	}

	diagnostics := []protocol.Diagnostic{}
	for _, item := range expr.Items {
		start := item.KeyExpr.Range().Start.Byte
		end := item.KeyExpr.Range().End.Byte
		if LiteralValue(item.KeyExpr) {
			start++
			end--
		}
		arg := string(input[start:end])
		if slices.Contains(builtinArgs, arg) {
			continue
		}

		diagnostic := checkStringLiteral(
			source,
			arg,
			fmt.Sprintf("'%v' not defined as an ARG in your Dockerfile", arg),
			args,
			item.KeyExpr.Range(),
		)

		if diagnostic != nil {
			diagnostics = append(diagnostics, *diagnostic)
		}
	}
	return diagnostics
}

func (c *BakeHCLDiagnosticsCollector) checkTargetTarget(dockerfilePath string, expr *hclsyntax.TemplateExpr, literalValueExpr *hclsyntax.LiteralValueExpr, source string) *protocol.Diagnostic {
	value, _ := literalValueExpr.Value(&hcl.EvalContext{})
	target := value.AsString()

	_, nodes := OpenDockerfile(context.Background(), c.docs, dockerfilePath)
	found := false
	for _, child := range nodes {
		if strings.EqualFold(child.Value, "FROM") {
			if child.Next != nil && child.Next.Next != nil && child.Next.Next.Next != nil && child.Next.Next.Next.Value == target {
				found = true
				break
			}
		}
	}

	if !found {
		return &protocol.Diagnostic{
			Message:  "target could not be found in your Dockerfile",
			Source:   types.CreateStringPointer(source),
			Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(expr.SrcRange.Start.Line) - 1,
					Character: uint32(expr.SrcRange.Start.Column) - 1,
				},
				End: protocol.Position{
					Line:      uint32(expr.SrcRange.End.Line) - 1,
					Character: uint32(expr.SrcRange.End.Column) - 1,
				},
			},
		}
	}
	return nil
}

func LiteralValue(expr hclsyntax.Expression) bool {
	if objectConsKey, ok := expr.(*hclsyntax.ObjectConsKeyExpr); ok {
		if template, ok := objectConsKey.Wrapped.(*hclsyntax.TemplateExpr); ok && len(template.Parts) == 1 {
			if _, ok := template.Parts[0].(*hclsyntax.LiteralValueExpr); ok {
				return true
			}
		}
	}
	return false
}

func checkStringLiteral(diagnosticSource, attributeValue, message string, expectedValues []string, attributeRange hcl.Range) *protocol.Diagnostic {
	if slices.Contains(expectedValues, attributeValue) {
		return nil
	}

	return &protocol.Diagnostic{
		Message:  message,
		Source:   types.CreateStringPointer(diagnosticSource),
		Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(attributeRange.Start.Line) - 1,
				Character: uint32(attributeRange.Start.Column) - 1,
			},
			End: protocol.Position{
				Line:      uint32(attributeRange.End.Line) - 1,
				Character: uint32(attributeRange.End.Column) - 1,
			},
		},
	}
}

func PrintOutput(directory, target string, bakeFileContent []byte) *BakePrintOutput {
	var buf bytes.Buffer
	cmd := exec.Command("docker", "buildx", "bake", "-f-", "--print", target)
	cmd.Stdin = bytes.NewBuffer(bakeFileContent)
	cmd.Dir = directory
	cmd.Stdout = &buf
	err := cmd.Start()
	if err != nil {
		return nil
	}
	cmdErr := cmd.Wait()
	if cmdErr != nil {
		var ee *exec.ExitError
		if errors.As(cmdErr, &ee) {
			c := ee.ProcessState.ExitCode()
			if c != 0 {
				return nil
			}
		}
	}

	var output *BakePrintOutput
	err = json.Unmarshal(buf.Bytes(), &output)
	if err != nil {
		return nil
	}
	return output
}

func ParseDockerfileFromBakeOutput(doc document.Document, target string) (string, error) {
	documentURI := doc.URI()
	output := PrintOutput(filepath.Dir(documentURI.Filename()), target, doc.Input())
	if output == nil {
		return "", nil
	}

	url, err := url.Parse(string(documentURI))
	if err != nil {
		return "", fmt.Errorf("LSP client sent invalid URI: %v", string(documentURI))
	}
	contextPath, err := types.AbsoluteFolder(url)
	if err != nil {
		return "", fmt.Errorf("LSP client sent invalid URI: %v", string(documentURI))
	}
	if block, ok := output.Target[target]; ok {
		if block.DockerfileInline != nil {
			return "", nil
		} else if block.Context != nil {
			contextPath = *block.Context
			contextPath, err = types.AbsolutePath(url, contextPath)
			if err != nil {
				return "", nil
			}
		}

		if block.Dockerfile != nil {
			return filepath.Join(contextPath, *block.Dockerfile), nil
		}
		return filepath.Join(contextPath, "Dockerfile"), nil
	}
	return "", nil
}
