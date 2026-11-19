package hcl

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func shouldSuggest(content []byte, body *hclsyntax.Body, position protocol.Position) bool {
	for _, block := range body.Blocks {
		if isInsideRange(block.Range(), position) {
			return false
		}
	}

	for _, attribute := range body.Attributes {
		if isInsideRange(attribute.Range(), position) {
			return false
		}
	}

	rawTokens, _ := hclsyntax.LexConfig(content, "", hcl.InitialPos)
	for _, rawToken := range rawTokens {
		if rawToken.Type == hclsyntax.TokenComment {
			if isInsideRange(rawToken.Range, position) {
				return false
			}
		}
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) <= int(position.Line) {
		return false
	}
	return strings.TrimSpace(lines[position.Line]) != "}"
}

func InlineCompletion(ctx context.Context, params *protocol.InlineCompletionParams, manager *document.Manager, bakeDocument document.BakeHCLDocument) ([]protocol.InlineCompletionItem, error) {
	documentPath, err := bakeDocument.DocumentPath()
	if err != nil {
		return nil, fmt.Errorf("LSP client sent invalid URI: %v", params.TextDocument.URI)
	}

	body, ok := bakeDocument.File().Body.(*hclsyntax.Body)
	if !ok {
		return nil, errors.New("unrecognized body in HCL document")
	}

	documentURI, dockerfilePath := types.Concatenate(documentPath.Folder, "Dockerfile", documentPath.WSLDollarSignHost)
	if !shouldSuggest(bakeDocument.Input(), body, params.Position) {
		return nil, nil
	}

	preexistingTargets := []string{}
	for _, block := range body.Blocks {
		if block.Type == "target" && len(block.Labels) > 0 {
			preexistingTargets = append(preexistingTargets, block.Labels[0])
		}

		if attribute, ok := block.Body.Attributes["target"]; ok {
			if templateExpr, ok := attribute.Expr.(*hclsyntax.TemplateExpr); ok {
				if len(templateExpr.Parts) == 1 {
					if literalValueExpr, ok := templateExpr.Parts[0].(*hclsyntax.LiteralValueExpr); ok {
						value, _ := literalValueExpr.Value(&hcl.EvalContext{})
						if value.Type() == cty.String {
							target := value.AsString()
							preexistingTargets = append(preexistingTargets, target)
						}
					}
				}
			}
		}
	}

	argNames := []string{}
	args := map[string]string{}
	targets := []string{}
	_, nodes := document.OpenDockerfile(ctx, manager, documentURI, dockerfilePath)
	before := true
	for _, child := range nodes {
		if strings.EqualFold(child.Value, "ARG") && before {
			if child.Next != nil {
				arg := child.Next.Value
				idx := strings.Index(arg, "=")
				if idx == -1 {
					args[arg] = ""
					argNames = append(argNames, arg)
				} else {
					args[arg[:idx]] = arg[idx+1:]
					argNames = append(argNames, arg[:idx])
				}
			}
		} else if strings.EqualFold(child.Value, "FROM") {
			before = false
			if child.Next != nil && child.Next.Next != nil && strings.EqualFold(child.Next.Next.Value, "AS") && child.Next.Next.Next != nil {
				if !slices.Contains(preexistingTargets, child.Next.Next.Next.Value) {
					targets = append(targets, child.Next.Next.Next.Value)
				}
			}
		}
	}

	if len(targets) > 0 {
		items := []protocol.InlineCompletionItem{}
		lines := strings.Split(string(bakeDocument.Input()), "\n")
		for _, target := range targets {
			sb := strings.Builder{}
			fmt.Fprintf(&sb, "target \"%v\" {\n", target)
			fmt.Fprintf(&sb, "  target = \"%v\"\n", target)

			if len(args) > 0 {
				sb.WriteString("  args = {\n")
				for _, argName := range argNames {
					fmt.Fprintf(&sb, "    %v = \"%v\"\n", argName, args[argName])
				}
				sb.WriteString("  }\n")
			}

			sb.WriteString("}\n")

			content := strings.TrimLeftFunc(lines[params.Position.Line], unicode.IsSpace)
			if strings.HasPrefix(sb.String(), content) {
				items = append(items, protocol.InlineCompletionItem{
					Range: &protocol.Range{
						Start: protocol.Position{
							Line:      params.Position.Line,
							Character: params.Position.Character - uint32(len(content)),
						},
						End: params.Position,
					},
					InsertText: sb.String(),
				})
			}

		}

		return items, nil
	}
	return nil, nil
}
