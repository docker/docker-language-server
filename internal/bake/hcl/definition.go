package hcl

import (
	"context"
	"errors"
	"strings"

	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"go.lsp.dev/uri"
)

func Definition(ctx context.Context, definitionLinkSupport bool, manager *document.Manager, documentURI uri.URI, doc document.BakeHCLDocument, position protocol.Position) (any, error) {
	body, ok := doc.File().Body.(*hclsyntax.Body)
	if !ok {
		return nil, errors.New("unrecognized body in HCL document")
	}

	for _, b := range body.Blocks {
		if isInsideBodyRangeLines(b.Body, int(position.Line+1)) {
			for _, attribute := range b.Body.Attributes {
				if isInsideRange(attribute.Expr.Range(), position) {
					return ResolveAttributeValue(ctx, definitionLinkSupport, manager, doc, body, documentURI, position, b, attribute), nil
				}
			}
		}
	}

	for _, attribute := range body.Attributes {
		if isInsideRange(attribute.NameRange, position) {
			return types.CreateDefinitionResult(
				definitionLinkSupport,
				protocol.Range{
					Start: protocol.Position{
						Line:      uint32(attribute.NameRange.Start.Line) - 1,
						Character: uint32(attribute.NameRange.Start.Column) - 1,
					},
					End: protocol.Position{
						Line:      uint32(attribute.NameRange.End.Line) - 1,
						Character: uint32(attribute.NameRange.End.Column) - 1,
					},
				},
				&protocol.Range{
					Start: protocol.Position{
						Line:      uint32(attribute.NameRange.Start.Line) - 1,
						Character: uint32(attribute.NameRange.Start.Column) - 1,
					},
					End: protocol.Position{
						Line:      uint32(attribute.NameRange.End.Line) - 1,
						Character: uint32(attribute.NameRange.End.Column) - 1,
					},
				},
				string(documentURI),
			), nil
		}

		if isInsideRange(attribute.SrcRange, position) {
			return ResolveAttributeValue(ctx, definitionLinkSupport, manager, doc, body, documentURI, position, nil, attribute), nil
		}
	}
	return nil, nil
}

func ResolveAttributeValue(ctx context.Context, definitionLinkSupport bool, manager *document.Manager, doc document.BakeHCLDocument, body *hclsyntax.Body, documentURI uri.URI, position protocol.Position, sourceBlock *hclsyntax.Block, attribute *hclsyntax.Attribute) any {
	if tupleConsExpr, ok := attribute.Expr.(*hclsyntax.TupleConsExpr); ok {
		for _, e := range tupleConsExpr.Exprs {
			if isInsideRange(e.Range(), position) {
				if templateExpr, ok := e.(*hclsyntax.TemplateExpr); ok && sourceBlock != nil {
					if templateExpr.IsStringLiteral() {
						// look up a target reference if it's inside a
						// target block's inherits attribute, or a
						// group block's targets attribute
						if (sourceBlock.Type == "target" && attribute.Name == "inherits") ||
							(sourceBlock.Type == "group" && attribute.Name == "targets") {
							value, _ := templateExpr.Value(&hcl.EvalContext{})
							target := value.AsString()
							templateExprRange := templateExpr.Range()
							sourceRange := hcl.Range{
								Start: hcl.Pos{
									Line:   templateExprRange.Start.Line,
									Column: templateExprRange.Start.Column + 1,
								},
								End: hcl.Pos{
									Line:   templateExprRange.End.Line,
									Column: templateExprRange.End.Column - 1,
								},
							}
							return CalculateBlockLocation(definitionLinkSupport, doc.Input(), body, documentURI, sourceRange, "target", target, false)
						}
					}
				}

				return ResolveExpression(ctx, definitionLinkSupport, manager, doc, body, documentURI, position, sourceBlock, attribute.Name, e)
			}
		}
	}

	return ResolveExpression(ctx, definitionLinkSupport, manager, doc, body, documentURI, position, sourceBlock, attribute.Name, attribute.Expr)
}

func ResolveExpression(ctx context.Context, definitionLinkSupport bool, manager *document.Manager, doc document.BakeHCLDocument, body *hclsyntax.Body, documentURI uri.URI, position protocol.Position, sourceBlock *hclsyntax.Block, attributeName string, expression hclsyntax.Expression) any {
	if templateExpr, ok := expression.(*hclsyntax.TemplateExpr); ok {
		for _, part := range templateExpr.Parts {
			if isInsideRange(part.Range(), position) {
				return ResolveExpression(ctx, definitionLinkSupport, manager, doc, body, documentURI, position, sourceBlock, attributeName, part)
			}
		}
	}

	if literalValueExpr, ok := expression.(*hclsyntax.LiteralValueExpr); ok && sourceBlock != nil && sourceBlock.Type == "target" {
		if attributeName == "no-cache-filter" || attributeName == "target" {
			dockerfileURI, dockerfilePath, err := doc.DockerfileDocumentPathForTarget(sourceBlock)
			if dockerfilePath == "" || err != nil {
				return nil
			}

			value, _ := literalValueExpr.Value(&hcl.EvalContext{})
			target := value.AsString()

			bytes, nodes := document.OpenDockerfile(ctx, manager, dockerfileURI, dockerfilePath)
			lines := strings.Split(string(bytes), "\n")
			for _, child := range nodes {
				if strings.EqualFold(child.Value, "FROM") {
					if child.Next != nil && child.Next.Next != nil && strings.EqualFold(child.Next.Next.Value, "AS") && child.Next.Next.Next != nil && child.Next.Next.Next.Value == target {
						endLineLength := len(lines[child.EndLine-1])
						// 13 is ASCII for \r
						if lines[child.EndLine-1][endLineLength-1] == 13 {
							endLineLength--
						}
						return types.CreateDefinitionResult(
							definitionLinkSupport,
							protocol.Range{
								Start: protocol.Position{
									Line:      uint32(child.StartLine) - 1,
									Character: 0,
								},
								End: protocol.Position{
									Line:      uint32(child.EndLine) - 1,
									Character: uint32(endLineLength),
								},
							},
							&protocol.Range{
								Start: protocol.Position{
									Line:      uint32(literalValueExpr.Range().Start.Line) - 1,
									Character: uint32(literalValueExpr.Range().Start.Column) - 1,
								},
								End: protocol.Position{
									Line:      uint32(literalValueExpr.Range().End.Line) - 1,
									Character: uint32(uint32(literalValueExpr.Range().End.Column) - 1),
								},
							},
							protocol.URI(dockerfileURI),
						)
					}
				}
			}
		}
	}

	if forExpr, ok := expression.(*hclsyntax.ForExpr); ok {
		if isInsideRange(forExpr.CollExpr.Range(), position) {
			return ResolveExpression(ctx, definitionLinkSupport, manager, doc, body, documentURI, position, sourceBlock, attributeName, forExpr.CollExpr)
		}

		if forExpr.CondExpr != nil && isInsideRange(forExpr.CondExpr.Range(), position) {
			return ResolveExpression(ctx, definitionLinkSupport, manager, doc, body, documentURI, position, sourceBlock, attributeName, forExpr.CondExpr)
		}
	}

	if objectConsExpression, ok := expression.(*hclsyntax.ObjectConsExpr); ok {
		for _, item := range objectConsExpression.Items {
			if isInsideRange(item.KeyExpr.Range(), position) && sourceBlock != nil {
				if attributeName == "args" && sourceBlock.Type == "target" {
					dockerfileURI, dockerfilePath, err := doc.DockerfileDocumentPathForTarget(sourceBlock)
					if dockerfilePath == "" || err != nil {
						return nil
					}

					start := item.KeyExpr.Range().Start.Byte
					end := item.KeyExpr.Range().End.Byte
					if LiteralValue(item.KeyExpr) {
						start++
						end--
					}
					arg := string(doc.Input()[start:end])
					bytes, nodes := document.OpenDockerfile(ctx, manager, dockerfileURI, dockerfilePath)
					lines := strings.Split(string(bytes), "\n")
					for _, child := range nodes {
						if strings.EqualFold(child.Value, "ARG") {
							node := child
							child = child.Next
							for child != nil {
								value := child.Value
								idx := strings.Index(value, "=")
								if idx != -1 {
									value = value[:idx]
								}

								if value == arg {
									originSelectionRange := protocol.Range{
										Start: protocol.Position{
											Line:      uint32(item.KeyExpr.Range().Start.Line) - 1,
											Character: uint32(item.KeyExpr.Range().Start.Column) - 1,
										},
										End: protocol.Position{
											Line:      uint32(item.KeyExpr.Range().End.Line) - 1,
											Character: uint32(item.KeyExpr.Range().End.Column) - 1,
										},
									}
									if LiteralValue(item.KeyExpr) {
										originSelectionRange.Start.Character = originSelectionRange.Start.Character + 1
										originSelectionRange.End.Character = originSelectionRange.End.Character - 1
									}
									endLineLength := len(lines[node.EndLine-1])
									// 13 is ASCII for \r
									if lines[node.EndLine-1][endLineLength-1] == 13 {
										endLineLength--
									}
									return types.CreateDefinitionResult(
										definitionLinkSupport,
										protocol.Range{
											Start: protocol.Position{Line: uint32(node.StartLine) - 1, Character: 0},
											End:   protocol.Position{Line: uint32(node.EndLine) - 1, Character: uint32(endLineLength)},
										},
										&originSelectionRange,
										dockerfileURI,
									)
								}
								child = child.Next
							}
						}
					}
					return nil
				}
			}

			if isInsideRange(item.ValueExpr.Range(), position) {
				return ResolveExpression(ctx, definitionLinkSupport, manager, doc, body, documentURI, position, sourceBlock, attributeName, item.ValueExpr)
			}
		}
	}

	if binaryExpression, ok := expression.(*hclsyntax.BinaryOpExpr); ok {
		if isInsideRange(binaryExpression.LHS.Range(), position) {
			return ResolveExpression(ctx, definitionLinkSupport, manager, doc, body, documentURI, position, sourceBlock, attributeName, binaryExpression.LHS)
		}

		if isInsideRange(binaryExpression.RHS.Range(), position) {
			return ResolveExpression(ctx, definitionLinkSupport, manager, doc, body, documentURI, position, sourceBlock, attributeName, binaryExpression.RHS)
		}

		return nil
	}

	if conditional, ok := expression.(*hclsyntax.ConditionalExpr); ok {
		if isInsideRange(conditional.Condition.Range(), position) {
			return ResolveExpression(ctx, definitionLinkSupport, manager, doc, body, documentURI, position, sourceBlock, attributeName, conditional.Condition)
		}

		if isInsideRange(conditional.TrueResult.Range(), position) {
			return ResolveExpression(ctx, definitionLinkSupport, manager, doc, body, documentURI, position, sourceBlock, attributeName, conditional.TrueResult)
		}

		if isInsideRange(conditional.FalseResult.Range(), position) {
			return ResolveExpression(ctx, definitionLinkSupport, manager, doc, body, documentURI, position, sourceBlock, attributeName, conditional.FalseResult)
		}
		return nil
	}

	if _, ok := expression.(*hclsyntax.ScopeTraversalExpr); ok {
		input := doc.Input()
		name := string(input[expression.Range().Start.Byte:expression.Range().End.Byte])
		if strings.Index(name, "target") == 0 {
			parts := strings.Split(name, ".")
			if len(parts) == 3 {
				offset := expression.Range().Start.Column - 1
				if int(position.Character) < offset+len(parts[0])+1 {
					// cursor inside "target" of target.targetName.attribute
					return nil
				}

				if offset+len(parts[0])+1 <= int(position.Character) && int(position.Character) <= offset+len(parts[0])+1+len(parts[1]) {
					// cursor inside "targetName" of target.targetName.attribute
					return CalculateBlockLocation(
						definitionLinkSupport,
						doc.Input(),
						body,
						documentURI,
						hcl.Range{
							Start: hcl.Pos{
								Line: expression.Range().Start.Line,
								// offset + length + dotSeparator + one-based
								Column: offset + len(parts[0]) + 1 + 1,
							},
							End: hcl.Pos{
								Line: expression.Range().End.Line,
								// offset + length + dotSeparator + length + one-based
								Column: offset + len(parts[0]) + 1 + len(parts[1]) + 1,
							},
						},
						"target",
						parts[1],
						false,
					)
				}

				if offset+len(parts[0])+1+len(parts[1])+1 <= int(position.Character) && int(position.Character) <= offset+len(parts[0])+1+len(parts[1])+1+len(parts[2]) {
					// cursor inside "attribute" of target.targetName.attribute
					return targetAttributeLocation(
						definitionLinkSupport,
						body,
						documentURI,
						hcl.Range{
							Start: hcl.Pos{
								Line:   expression.Range().Start.Line,
								Column: offset + 1 + len(parts[0]) + 1 + len(parts[1]) + 1,
							},
							End: hcl.Pos{
								Line:   expression.Range().End.Line,
								Column: offset + len(parts[0]) + 1 + len(parts[1]) + 1 + len(parts[2]) + 1,
							},
						},
						parts[1],
						parts[2],
					)
				}
			}
		}
		return CalculateBlockLocation(definitionLinkSupport, input, body, documentURI, expression.Range(), "variable", name, true)
	}

	if templateWrapExpr, ok := expression.(*hclsyntax.TemplateWrapExpr); ok {
		return ResolveExpression(ctx, definitionLinkSupport, manager, doc, body, documentURI, position, sourceBlock, attributeName, templateWrapExpr.Wrapped)
	}

	if functionCallExpr, ok := expression.(*hclsyntax.FunctionCallExpr); ok {
		if isInsideRange(functionCallExpr.NameRange, position) {
			return CalculateBlockLocation(definitionLinkSupport, doc.Input(), body, documentURI, functionCallExpr.NameRange, "function", functionCallExpr.Name, true)
		}

		for _, arg := range functionCallExpr.Args {
			if isInsideRange(arg.Range(), position) {
				return ResolveExpression(ctx, definitionLinkSupport, manager, doc, body, documentURI, position, sourceBlock, attributeName, arg)
			}
		}
	}
	return nil
}

func targetAttributeLocation(definitionLinkSupport bool, body *hclsyntax.Body, documentURI uri.URI, sourceRange hcl.Range, targetName, attributeName string) any {
	for _, b := range body.Blocks {
		if b.Type == "target" && b.Labels[0] == targetName {
			if attribute, ok := b.Body.Attributes[attributeName]; ok {
				return types.CreateDefinitionResult(
					definitionLinkSupport,
					protocol.Range{
						Start: protocol.Position{
							Line:      uint32(attribute.NameRange.Start.Line) - 1,
							Character: protocol.UInteger(attribute.NameRange.Start.Column) - 1,
						},
						End: protocol.Position{
							Line:      uint32(attribute.NameRange.End.Line) - 1,
							Character: protocol.UInteger(attribute.NameRange.End.Column) - 1,
						},
					},
					&protocol.Range{
						Start: protocol.Position{
							Line:      uint32(sourceRange.Start.Line) - 1,
							Character: uint32(sourceRange.Start.Column) - 1,
						},
						End: protocol.Position{
							Line:      uint32(sourceRange.End.Line) - 1,
							Character: uint32(sourceRange.End.Column) - 1,
						},
					},
					string(documentURI),
				)
			}
		}
	}
	return nil
}

// CalculateBlockLocation finds a block with the specified name and
// returns it. If variable is true then it will also look at the
// top-level attributes of the HCL file and resolve to those if the
// names match.
func CalculateBlockLocation(definitionLinkSupport bool, input []byte, body *hclsyntax.Body, documentURI uri.URI, sourceRange hcl.Range, blockName, name string, variable bool) any {
	for _, b := range body.Blocks {
		if b.Type == blockName && b.Labels[0] == name {
			startCharacter := uint32(b.LabelRanges[0].Start.Column)
			endCharacter := uint32(b.LabelRanges[0].End.Column)
			variableNameDeclaration := string(input[b.LabelRanges[0].Start.Byte:b.LabelRanges[0].End.Byte])
			if Quoted(variableNameDeclaration) {
				endCharacter -= 2
			} else {
				startCharacter--
				endCharacter--
			}
			return types.CreateDefinitionResult(
				definitionLinkSupport,
				protocol.Range{
					Start: protocol.Position{
						Line:      uint32(b.LabelRanges[0].Start.Line) - 1,
						Character: startCharacter,
					},
					End: protocol.Position{
						Line:      uint32(b.LabelRanges[0].End.Line) - 1,
						Character: endCharacter,
					},
				},
				&protocol.Range{
					Start: protocol.Position{
						Line:      uint32(sourceRange.Start.Line) - 1,
						Character: uint32(sourceRange.Start.Column) - 1,
					},
					End: protocol.Position{
						Line:      uint32(sourceRange.End.Line) - 1,
						Character: uint32(sourceRange.End.Column) - 1,
					},
				},
				string(documentURI),
			)
		}
	}

	if attribute, ok := body.Attributes[name]; ok && variable {
		return types.CreateDefinitionResult(
			definitionLinkSupport,
			protocol.Range{
				Start: protocol.Position{
					Line:      uint32(attribute.NameRange.Start.Line) - 1,
					Character: uint32(attribute.NameRange.Start.Column) - 1,
				},
				End: protocol.Position{
					Line:      uint32(attribute.NameRange.End.Line) - 1,
					Character: uint32(attribute.NameRange.End.Column) - 1,
				},
			},
			&protocol.Range{
				Start: protocol.Position{
					Line:      uint32(sourceRange.Start.Line) - 1,
					Character: uint32(sourceRange.Start.Column) - 1,
				},
				End: protocol.Position{
					Line:      uint32(sourceRange.End.Line) - 1,
					Character: uint32(sourceRange.End.Column) - 1,
				},
			},
			string(documentURI),
		)
	}
	return nil
}
