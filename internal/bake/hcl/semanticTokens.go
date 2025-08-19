package hcl

import (
	"context"
	"errors"
	"sort"

	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

const TokenType_Type = "type"
const TokenType_Class = "class"
const TokenType_String = "string"
const TokenType_Property = "property"
const TokenType_Keyword = "keyword"
const TokenType_Number = "number"
const TokenType_Operator = "operator"
const TokenType_Variable = "variable"
const TokenType_Comment = "comment"

// SemanticTokenTypes is the list of semantic token types that the
// language server supports and will be included in the capabilities
// response payload to the client's initialize request.
var SemanticTokenTypes = []string{
	TokenType_Type,
	TokenType_Class,
	TokenType_String,
	TokenType_Variable,
	TokenType_Property,
	TokenType_Keyword,
	TokenType_Operator,
	TokenType_Number,
	TokenType_Comment,
}

func SemanticTokenTypeIndex(tokenType string) uint32 {
	for i := range SemanticTokenTypes {
		if tokenType == SemanticTokenTypes[i] {
			return uint32(i)
		}
	}
	return 0
}

func TokenType(hclType lang.SemanticTokenType) uint32 {
	if hclType == TokenType_Comment {
		return SemanticTokenTypeIndex(TokenType_Comment)
	}
	if hclType == lang.TokenBlockType {
		return SemanticTokenTypeIndex(TokenType_Type)
	}
	if hclType == lang.TokenBlockLabel {
		return SemanticTokenTypeIndex(TokenType_Class)
	}
	if hclType == lang.TokenAttrName || hclType == lang.TokenMapKey {
		// lang.TokenMapKey is misleading because it can be quoted or unquoted
		return SemanticTokenTypeIndex(TokenType_Property)
	}
	if hclType == lang.TokenString {
		return SemanticTokenTypeIndex(TokenType_String)
	}
	if hclType == lang.TokenBool || hclType == lang.TokenKeyword {
		return SemanticTokenTypeIndex(TokenType_Keyword)
	}
	if hclType == lang.TokenNumber {
		return SemanticTokenTypeIndex(TokenType_Number)
	}
	if hclType == TokenType_Variable {
		return SemanticTokenTypeIndex(TokenType_Variable)
	}
	if hclType == TokenType_Operator {
		return SemanticTokenTypeIndex(TokenType_Operator)
	}
	return SemanticTokenTypeIndex(TokenType_Type)
}

func SemanticTokensFull(ctx context.Context, doc document.BakeHCLDocument, filename string) (*protocol.SemanticTokens, error) {
	tokens, err := doc.Decoder().SemanticTokensInFile(ctx, filename)
	if err != nil {
		var rangeErr *decoder.PosOutOfRangeError
		if !errors.As(err, &rangeErr) {
			return nil, err
		}
	}

	body, ok := doc.File().Body.(*hclsyntax.Body)
	if !ok {
		return nil, errors.New("unrecognized body in HCL document")
	}

	for _, block := range body.Blocks {
		for _, attribute := range block.Body.Attributes {
			expression := attribute.Expr
			tokens = append(tokens, evaluateToken(expression, true, false)...)
		}
	}

	bytes := doc.Input()
	rawTokens, _ := hclsyntax.LexConfig(bytes, "", hcl.InitialPos)
	for _, rawToken := range rawTokens {
		switch rawToken.Type {
		case hclsyntax.TokenIdent:
			if string(bytes[rawToken.Range.Start.Byte:rawToken.Range.End.Byte]) == "null" {
				tokens = append(tokens, lang.SemanticToken{
					Type:  lang.TokenKeyword,
					Range: rawToken.Range,
				})
			}
		case hclsyntax.TokenComment:
			switch rawToken.Range.Start.Line {
			case rawToken.Range.End.Line:
				tokens = append(tokens, lang.SemanticToken{
					Type:  TokenType_Comment,
					Range: rawToken.Range,
				})
			case rawToken.Range.End.Line - 1:
				switch bytes[rawToken.Range.Start.Byte] {
				case 35:
					// comment with a #
					tokens = append(tokens, convertSingleLineCommentToken(bytes, rawToken))
				case 47:
					// comment with a /
					if bytes[rawToken.Range.Start.Byte+1] == 47 {
						tokens = append(tokens, convertSingleLineCommentToken(bytes, rawToken))
					} else {
						tokens = append(tokens, convertMultilineCommentTokens(bytes, rawToken)...)
					}
				}
			default:
				// more than one line must be a multi-line comment
				tokens = append(tokens, convertMultilineCommentTokens(bytes, rawToken)...)
			}
		}
	}

	sort.Slice(tokens, func(i, j int) bool {
		if tokens[i].Range.Start.Line < tokens[j].Range.Start.Line {
			return true
		} else if tokens[i].Range.Start.Line > tokens[j].Range.Start.Line {
			return false
		}
		return tokens[i].Range.Start.Column < tokens[j].Range.Start.Column
	})

	result := &protocol.SemanticTokens{}
	currentLine := 0
	currentOffset := 0
	for _, token := range tokens {
		line := token.Range.Start.Line - 1
		if line != currentLine {
			currentOffset = 0
		}

		deltaLine := line - currentLine
		startOffset := token.Range.Start.Column - 1
		startCharacter := startOffset - currentOffset
		tokenLength := token.Range.End.Column - token.Range.Start.Column

		result.Data = append(result.Data, uint32(deltaLine))
		result.Data = append(result.Data, uint32(startCharacter))
		result.Data = append(result.Data, uint32(tokenLength))
		result.Data = append(result.Data, TokenType(token.Type))
		result.Data = append(result.Data, uint32(0)) // no modifiers at the moment

		currentLine = line
		currentOffset = startOffset
	}

	return result, nil
}

func evaluateToken(expression hcl.Expression, topLevel, insertLiterals bool) []lang.SemanticToken {
	tokens := []lang.SemanticToken{}
	if _, ok := expression.(*hclsyntax.ScopeTraversalExpr); ok {
		if topLevel {
			tokens = append(tokens, lang.SemanticToken{
				Type:  TokenType_Variable,
				Range: expression.Range(),
			})
		} else {
			tokens = append(tokens, wrapInterpolation(hcl.Range{
				Start: hcl.Pos{
					Line:   expression.Range().Start.Line,
					Column: expression.Range().Start.Column - 2,
				},
				End: hcl.Pos{
					Line:   expression.Range().End.Line,
					Column: expression.Range().End.Column + 1,
				},
			}, expression)...)
		}
	} else if literalValueExpr, ok := expression.(*hclsyntax.LiteralValueExpr); ok {
		if insertLiterals && literalValueExpr.Val.Type() == cty.Bool {
			tokens = append(tokens, lang.SemanticToken{
				Type:  lang.TokenKeyword,
				Range: literalValueExpr.SrcRange,
			})
		}
	} else if tupleConsExpr, ok := expression.(*hclsyntax.TupleConsExpr); ok {
		for _, expressions := range tupleConsExpr.Exprs {
			tokens = append(tokens, evaluateToken(expressions, true, insertLiterals)...)
		}
	} else if templateExpr, ok := expression.(*hclsyntax.TemplateExpr); ok {
		if !templateExpr.IsStringLiteral() {
			rng := expression.Range()
			tokens = append(tokens, lang.SemanticToken{
				Type: lang.TokenString,
				Range: hcl.Range{
					Start: rng.Start,
					End:   hcl.Pos{Line: rng.Start.Line, Column: rng.Start.Column + 1},
				},
			})
			for _, part := range templateExpr.Parts {
				if _, ok := part.(*hclsyntax.LiteralValueExpr); !ok {
					tokens = append(tokens, evaluateToken(part, false, false)...)
				}
			}
			tokens = append(tokens, lang.SemanticToken{
				Type: lang.TokenString,
				Range: hcl.Range{
					Start: hcl.Pos{Line: rng.End.Line, Column: rng.End.Column - 1},
					End:   rng.End,
				},
			})
		} else if insertLiterals {
			rng := expression.Range()
			tokens = append(tokens, lang.SemanticToken{Type: lang.TokenString, Range: rng})
		}
	} else if templateWrapExpr, ok := expression.(*hclsyntax.TemplateWrapExpr); ok {
		// "${var}" expression
		rng := expression.Range()
		tokens = append(tokens, lang.SemanticToken{
			Type: lang.TokenString,
			Range: hcl.Range{
				Start: rng.Start,
				End:   hcl.Pos{Line: rng.Start.Line, Column: rng.Start.Column + 1},
			},
		})
		tokens = append(tokens, wrapInterpolation(hcl.Range{
			Start: hcl.Pos{Line: rng.Start.Line, Column: rng.Start.Column + 1},
			End:   hcl.Pos{Line: rng.End.Line, Column: rng.End.Column - 1},
		}, templateWrapExpr.Wrapped)...)
		tokens = append(tokens, lang.SemanticToken{
			Type: lang.TokenString,
			Range: hcl.Range{
				Start: hcl.Pos{Line: rng.End.Line, Column: rng.End.Column - 1},
				End:   rng.End,
			},
		})
	} else if objectConsExpr, ok := expression.(*hclsyntax.ObjectConsExpr); ok {
		for _, item := range objectConsExpr.Items {
			tokens = append(tokens, evaluateToken(item.ValueExpr, topLevel, false)...)
		}
	} else if conditionalExpr, ok := expression.(*hclsyntax.ConditionalExpr); ok {
		tokens = append(tokens, evaluateToken(conditionalExpr.Condition, topLevel, false)...)
		tokens = append(tokens, evaluateToken(conditionalExpr.TrueResult, topLevel, true)...)
		tokens = append(tokens, evaluateToken(conditionalExpr.FalseResult, topLevel, true)...)
	} else if binaryExpr, ok := expression.(*hclsyntax.BinaryOpExpr); ok {
		tokens = append(tokens, evaluateToken(binaryExpr.LHS, topLevel, true)...)
		tokens = append(tokens, evaluateToken(binaryExpr.RHS, topLevel, true)...)
	}
	return tokens
}

func wrapInterpolation(rng hcl.Range, expression hcl.Expression) []lang.SemanticToken {
	// "${var}" expression
	tokens := []lang.SemanticToken{}
	tokens = append(tokens, lang.SemanticToken{
		Type: TokenType_Operator,
		Range: hcl.Range{
			Start: hcl.Pos{Line: rng.Start.Line, Column: rng.Start.Column},
			End:   hcl.Pos{Line: rng.Start.Line, Column: rng.Start.Column + 2},
		},
	})
	tokens = append(tokens, lang.SemanticToken{
		Type:  TokenType_Variable,
		Range: expression.Range(),
	})
	tokens = append(tokens, lang.SemanticToken{
		Type: TokenType_Operator,
		Range: hcl.Range{
			Start: hcl.Pos{Line: rng.End.Line, Column: rng.End.Column - 1},
			End:   hcl.Pos{Line: rng.End.Line, Column: rng.End.Column},
		},
	})
	return tokens
}

func convertSingleLineCommentToken(bytes []byte, token hclsyntax.Token) lang.SemanticToken {
	// take the start column and add the length of the comment
	endColumn := token.Range.Start.Column + (token.Range.End.Byte - token.Range.Start.Byte)
	// decrement by one for the \n
	endColumn--
	if bytes[endColumn-2] == 13 {
		// decrement by one more for the \r if found (ASCII 13)
		endColumn--
	}

	return createCommentToken(token.Range.Start.Line, token.Range.Start.Column, endColumn-token.Range.Start.Column)
}

func convertMultilineCommentTokens(bytes []byte, token hclsyntax.Token) []lang.SemanticToken {
	tokens := []lang.SemanticToken{}
	column := token.Range.Start.Column
	line := token.Range.Start.Line
	start := token.Range.Start.Byte
	for i := token.Range.Start.Byte; i < token.Range.End.Byte; i++ {
		switch bytes[i] {
		case 13:
			if i == start {
				column = 1
				start = i + 2
				line++
				i++
			} else {
				start++
			}
		case 10:
			if i == start {
				column = 1
				start = i + 1
				line++
			} else {
				tokens = append(tokens, createCommentToken(line, column, i-start))
				start = i + 1
				column = 1
				line++
			}
		}
	}
	tokens = append(tokens, createCommentToken(line, column, token.Range.End.Byte-start))
	return tokens
}

func createCommentToken(line, column, length int) lang.SemanticToken {
	return lang.SemanticToken{
		Type: TokenType_Comment,
		Range: hcl.Range{
			Start: hcl.Pos{Line: line, Column: column},
			End:   hcl.Pos{Line: line, Column: column + length},
		},
	}
}
