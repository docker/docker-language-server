package hcl

import (
	"errors"
	"sort"

	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func SelectionRange(doc document.BakeHCLDocument, positions []protocol.Position) ([]protocol.SelectionRange, error) {
	body, ok := doc.File().Body.(*hclsyntax.Body)
	if !ok {
		return nil, errors.New("unrecognized body in HCL document")
	}

	result := make([]protocol.SelectionRange, len(positions))
	for i, position := range positions {
		result[i] = findSelectionRangeForPosition(body, position)
	}

	return result, nil
}

func findSelectionRangeForPosition(body *hclsyntax.Body, position protocol.Position) protocol.SelectionRange {
	// Start with the most specific range and work outward
	var ranges []protocol.Range

	// Check if position is inside any block
	for _, block := range body.Blocks {
		if isInsideRange(block.Range(), position) {
			// Add ranges for the block content
			blockRanges := getBlockSelectionRanges(block, position)
			ranges = append(ranges, blockRanges...)
		}
	}

	// Check if position is inside any top-level attribute
	for _, attr := range body.Attributes {
		if isInsideRange(attr.SrcRange, position) {
			attrRanges := getAttributeSelectionRanges(attr, position)
			ranges = append(ranges, attrRanges...)
		}
	}

	// Build the nested selection range structure
	if len(ranges) == 0 {
		// No specific range found, return the whole document
		return protocol.SelectionRange{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 1000, Character: 0}, // Large enough end position
			},
		}
	}

	// Sort ranges by specificity (smallest first)
	sortRangesBySpecificity(ranges)

	// Build nested structure
	var current *protocol.SelectionRange
	for i := len(ranges) - 1; i >= 0; i-- {
		newRange := &protocol.SelectionRange{
			Range:  ranges[i],
			Parent: current,
		}
		current = newRange
	}

	return *current
}

func getBlockSelectionRanges(block *hclsyntax.Block, position protocol.Position) []protocol.Range {
	var ranges []protocol.Range

	// Check if inside block label
	for _, labelRange := range block.LabelRanges {
		if isInsideRange(labelRange, position) {
			ranges = append(ranges, hclRangeToProtocolRange(labelRange))
			break
		}
	}

	// Check if inside block type
	if isInsideRange(block.TypeRange, position) {
		ranges = append(ranges, hclRangeToProtocolRange(block.TypeRange))
	}

	// Check block body attributes
	for _, attr := range block.Body.Attributes {
		if isInsideRange(attr.SrcRange, position) {
			attrRanges := getAttributeSelectionRanges(attr, position)
			ranges = append(ranges, attrRanges...)
		}
	}

	// Check nested blocks
	for _, nestedBlock := range block.Body.Blocks {
		if isInsideRange(nestedBlock.Range(), position) {
			nestedRanges := getBlockSelectionRanges(nestedBlock, position)
			ranges = append(ranges, nestedRanges...)
		}
	}

	// Add the block body range
	ranges = append(ranges, hclRangeToProtocolRange(block.Body.SrcRange))

	// Add the entire block range
	ranges = append(ranges, hclRangeToProtocolRange(block.Range()))

	return ranges
}

func getAttributeSelectionRanges(attr *hclsyntax.Attribute, position protocol.Position) []protocol.Range {
	var ranges []protocol.Range

	// Check if inside attribute name
	if isInsideRange(attr.NameRange, position) {
		ranges = append(ranges, hclRangeToProtocolRange(attr.NameRange))
	}

	// Check if inside attribute value expression
	if isInsideRange(attr.Expr.Range(), position) {
		exprRanges := getExpressionSelectionRanges(attr.Expr, position)
		ranges = append(ranges, exprRanges...)
	}

	// Add the entire attribute range
	ranges = append(ranges, hclRangeToProtocolRange(attr.SrcRange))

	return ranges
}

func getExpressionSelectionRanges(expr hclsyntax.Expression, position protocol.Position) []protocol.Range {
	var ranges []protocol.Range

	switch e := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		ranges = append(ranges, hclRangeToProtocolRange(e.Range()))

	case *hclsyntax.TemplateExpr:
		for _, part := range e.Parts {
			if isInsideRange(part.Range(), position) {
				partRanges := getExpressionSelectionRanges(part, position)
				ranges = append(ranges, partRanges...)
			}
		}
		ranges = append(ranges, hclRangeToProtocolRange(e.Range()))

	case *hclsyntax.TupleConsExpr:
		for _, item := range e.Exprs {
			if isInsideRange(item.Range(), position) {
				itemRanges := getExpressionSelectionRanges(item, position)
				ranges = append(ranges, itemRanges...)
			}
		}
		ranges = append(ranges, hclRangeToProtocolRange(e.Range()))

	case *hclsyntax.ObjectConsExpr:
		for _, item := range e.Items {
			if isInsideRange(item.KeyExpr.Range(), position) {
				keyRanges := getExpressionSelectionRanges(item.KeyExpr, position)
				ranges = append(ranges, keyRanges...)
			}
			if isInsideRange(item.ValueExpr.Range(), position) {
				valueRanges := getExpressionSelectionRanges(item.ValueExpr, position)
				ranges = append(ranges, valueRanges...)
			}
		}
		ranges = append(ranges, hclRangeToProtocolRange(e.Range()))

	case *hclsyntax.ScopeTraversalExpr:
		ranges = append(ranges, hclRangeToProtocolRange(e.Range()))

	case *hclsyntax.FunctionCallExpr:
		if isInsideRange(e.NameRange, position) {
			ranges = append(ranges, hclRangeToProtocolRange(e.NameRange))
		}
		for _, arg := range e.Args {
			if isInsideRange(arg.Range(), position) {
				argRanges := getExpressionSelectionRanges(arg, position)
				ranges = append(ranges, argRanges...)
			}
		}
		ranges = append(ranges, hclRangeToProtocolRange(e.Range()))

	case *hclsyntax.ConditionalExpr:
		if isInsideRange(e.Condition.Range(), position) {
			condRanges := getExpressionSelectionRanges(e.Condition, position)
			ranges = append(ranges, condRanges...)
		}
		if isInsideRange(e.TrueResult.Range(), position) {
			trueRanges := getExpressionSelectionRanges(e.TrueResult, position)
			ranges = append(ranges, trueRanges...)
		}
		if isInsideRange(e.FalseResult.Range(), position) {
			falseRanges := getExpressionSelectionRanges(e.FalseResult, position)
			ranges = append(ranges, falseRanges...)
		}
		ranges = append(ranges, hclRangeToProtocolRange(e.Range()))

	case *hclsyntax.BinaryOpExpr:
		if isInsideRange(e.LHS.Range(), position) {
			lhsRanges := getExpressionSelectionRanges(e.LHS, position)
			ranges = append(ranges, lhsRanges...)
		}
		if isInsideRange(e.RHS.Range(), position) {
			rhsRanges := getExpressionSelectionRanges(e.RHS, position)
			ranges = append(ranges, rhsRanges...)
		}
		ranges = append(ranges, hclRangeToProtocolRange(e.Range()))

	default:
		ranges = append(ranges, hclRangeToProtocolRange(expr.Range()))
	}

	return ranges
}

func hclRangeToProtocolRange(hclRange hcl.Range) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{
			Line:      uint32(hclRange.Start.Line - 1),
			Character: uint32(hclRange.Start.Column - 1),
		},
		End: protocol.Position{
			Line:      uint32(hclRange.End.Line - 1),
			Character: uint32(hclRange.End.Column - 1),
		},
	}
}

func sortRangesBySpecificity(ranges []protocol.Range) {
	// Sort ranges by size (smaller ranges first)
	sort.Slice(ranges, func(i, j int) bool {
		return rangeSize(ranges[i]) < rangeSize(ranges[j])
	})
}

func rangeSize(r protocol.Range) int {
	lineSpan := int(r.End.Line - r.Start.Line)
	charSpan := int(r.End.Character - r.Start.Character)

	// If it's a single line, just return character span
	if lineSpan == 0 {
		return charSpan
	}

	// For multi-line ranges, weight lines more heavily
	return lineSpan*10000 + charSpan
}

func isInsideRange(hclRange hcl.Range, position protocol.Position) bool {
	// Convert protocol position to HCL position (1-based)
	line := int(position.Line) + 1
	column := int(position.Character) + 1

	// Check if position is within the range
	if line < hclRange.Start.Line || line > hclRange.End.Line {
		return false
	}

	// If on the start line, check column is after start
	if line == hclRange.Start.Line && column < hclRange.Start.Column {
		return false
	}

	// If on the end line, check column is before end
	if line == hclRange.End.Line && column > hclRange.End.Column {
		return false
	}

	return true
}
