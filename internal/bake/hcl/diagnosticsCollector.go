package hcl

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/moby/buildkit/frontend/dockerfile/parser"
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
	"BUILDKIT_SYNTAX",
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
	dp, err := doc.DocumentPath()
	if err != nil {
		return nil
	}

	_, err = bake.ParseFile(input, dp.FileName)
	diagnostics := []protocol.Diagnostic{}
	if err != nil {
		var sourceError *errdefs.SourceError
		var sourceRange *protocol.Range
		if errors.As(err, &sourceError) && len(sourceError.Ranges) > 0 {
			lines := strings.Split(string(input), "\n")
			sourceRange = &protocol.Range{
				Start: protocol.Position{
					Line:      uint32(sourceError.Ranges[0].Start.Line) - 1,
					Character: 0,
				},
				End: protocol.Position{
					Line:      uint32(sourceError.Ranges[0].Start.Line) - 1,
					Character: uint32(len(lines[sourceError.Ranges[0].Start.Line-1])),
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

			if hclDiagnostic.Subject == nil {
				if sourceRange != nil {
					diagnostic.Range = *sourceRange
				}
			} else {
				diagnostic.Range = protocol.Range{
					Start: protocol.Position{
						Line:      uint32(hclDiagnostic.Subject.Start.Line) - 1,
						Character: uint32(hclDiagnostic.Subject.Start.Column) - 1,
					},
					End: protocol.Position{
						Line:      uint32(hclDiagnostic.Subject.End.Line) - 1,
						Character: uint32(hclDiagnostic.Subject.End.Column) - 1,
					},
				}
			}

			diagnostics = append(diagnostics, diagnostic)
		}
	}

	bakeDoc := doc.(document.BakeHCLDocument)
	body, ok := bakeDoc.File().Body.(*hclsyntax.Body)
	if !ok {
		return diagnostics
	}

	dockerfileContent := map[string][]*parser.Node{}
	for _, b := range body.Blocks {
		if b.Type == "target" && len(b.Labels) == 1 {
			if _, ok := dockerfileContent[b.Labels[0]]; !ok {
				targetDockerfileURI, targetDockerfilePath, err := bakeDoc.DockerfileForTarget(b)
				if targetDockerfilePath != "" && err == nil {
					_, nodes := document.OpenDockerfile(context.Background(), c.docs, targetDockerfileURI, targetDockerfilePath)
					dockerfileContent[b.Labels[0]] = nodes
				} else {
					dockerfileContent[b.Labels[0]] = nil
				}
			}
		}
	}

	for _, block := range body.Blocks {
		if block.Type == "target" && len(block.Labels) == 1 {
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
								imageDiagnostics, err := c.scout.Analyze(protocol.DocumentUri(doc.URI()), target)
								if err == nil {
									for _, diagnostic := range imageDiagnostics {
										if diagnostic.Kind == "critical_high_vulnerabilities" || diagnostic.Kind == "vulnerabilities" {
											rng := templateExpr.SrcRange
											diagnostics = append(diagnostics, scout.ConvertDiagnostic(diagnostic, source, createProtocolRange(rng, true), nil))
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

			if attribute, ok := block.Body.Attributes["annotations"]; ok {
				if expr, ok := attribute.Expr.(*hclsyntax.TupleConsExpr); ok {
					for _, e := range expr.Exprs {
						if templateExpr, ok := e.(*hclsyntax.TemplateExpr); ok {
							if templateExpr.IsStringLiteral() {
								value, _ := templateExpr.Value(&hcl.EvalContext{})
								if len(strings.Split(value.AsString(), "=")) < 2 {
									diagnostics = append(diagnostics, protocol.Diagnostic{
										Message:  fmt.Sprintf(`invalid annotation "%v", expected key=value`, value.AsString()),
										Source:   types.CreateStringPointer(source),
										Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
										Range:    createProtocolRange(templateExpr.SrcRange, false),
									})
								}
							}
						}
					}
				}
			}

			_, dockerfilePath, err := bakeDoc.DockerfileForTarget(block)
			if dockerfilePath == "" || err != nil {
				continue
			}

			if attribute, ok := block.Body.Attributes["target"]; ok {
				if expr, ok := attribute.Expr.(*hclsyntax.TemplateExpr); ok && len(expr.Parts) == 1 {
					if literalValueExpr, ok := expr.Parts[0].(*hclsyntax.LiteralValueExpr); ok {
						if nodes, ok := dockerfileContent[block.Labels[0]]; ok {
							diagnostic := c.checkTargetTarget(nodes, expr, literalValueExpr, source)
							if diagnostic != nil {
								diagnostics = append(diagnostics, *diagnostic)
							}
						}
					}
				}
			}

			if attribute, ok := block.Body.Attributes["args"]; ok {
				if expr, ok := attribute.Expr.(*hclsyntax.ObjectConsExpr); ok {
					args := make(map[string]struct{})
					for _, b := range body.Blocks {
						if b.Type == "target" && len(b.Labels) == 1 && b.Labels[0] != block.Labels[0] {
							parents, _ := bakeDoc.ParentTargets(b.Labels[0])
							if slices.Contains(parents, block.Labels[0]) {
								if nodes, ok := dockerfileContent[b.Labels[0]]; ok {
									c.collectARGs(nodes, args)
								}
							}
						}
					}

					if nodes, ok := dockerfileContent[block.Labels[0]]; ok {
						argsDiagnostics := c.checkTargetArgs(nodes, input, expr, source, args)
						diagnostics = append(diagnostics, argsDiagnostics...)
					}
				}
			}
		}
	}
	return diagnostics
}

func (c *BakeHCLDiagnosticsCollector) collectARGs(nodes []*parser.Node, args map[string]struct{}) {
	for _, child := range nodes {
		if strings.EqualFold(child.Value, "ARG") {
			child = child.Next
			for child != nil {
				value := child.Value
				idx := strings.Index(value, "=")
				if idx != -1 {
					value = value[:idx]
				}
				args[value] = struct{}{}
				child = child.Next
			}
		}
	}
}

// checkTargetArgs examines the args attribute of a target block.
func (c *BakeHCLDiagnosticsCollector) checkTargetArgs(nodes []*parser.Node, input []byte, expr *hclsyntax.ObjectConsExpr, source string, args map[string]struct{}) []protocol.Diagnostic {
	c.collectARGs(nodes, args)
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
		if _, ok := args[arg]; !ok {
			diagnostic := createDiagnostic(
				source,
				fmt.Sprintf("'%v' not defined as an ARG in your Dockerfile", arg),
				item.KeyExpr.Range(),
			)
			diagnostics = append(diagnostics, *diagnostic)
		}

	}
	return diagnostics
}

func (c *BakeHCLDiagnosticsCollector) checkTargetTarget(nodes []*parser.Node, expr *hclsyntax.TemplateExpr, literalValueExpr *hclsyntax.LiteralValueExpr, source string) *protocol.Diagnostic {
	value, _ := literalValueExpr.Value(&hcl.EvalContext{})
	target := value.AsString()

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
	return createDiagnostic(diagnosticSource, message, attributeRange)
}

func createDiagnostic(diagnosticSource, message string, attributeRange hcl.Range) *protocol.Diagnostic {
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
