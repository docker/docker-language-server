package parser

import (
	"context"
	"strings"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

var BakeSchema = &schema.BodySchema{
	Blocks: map[string]*schema.BlockSchema{
		"group": {
			Labels: []*schema.LabelSchema{
				{
					Name: "groupName",
				},
			},
			Body: &schema.BodySchema{
				Attributes: map[string]*schema.AttributeSchema{
					"name": {
						IsOptional: true,
						Constraint: schema.LiteralType{Type: cty.String},
					},
					"description": {
						IsOptional: true,
						Constraint: schema.LiteralType{Type: cty.String},
					},
					"targets": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
					},
				},
			},
		},
		"variable": {
			Labels: []*schema.LabelSchema{
				{
					Name: "variableName",
				},
			},
			Body: &schema.BodySchema{
				Attributes: map[string]*schema.AttributeSchema{
					"default": {
						IsOptional: true,
						Constraint: schema.OneOf{
							schema.AnyExpression{OfType: cty.Bool},
							schema.AnyExpression{OfType: cty.Number},
							schema.AnyExpression{OfType: cty.String},
							schema.AnyExpression{OfType: cty.List(cty.Bool)},
							schema.AnyExpression{OfType: cty.List(cty.Number)},
							schema.AnyExpression{OfType: cty.List(cty.String)},
						},
					},
				},
				Blocks: map[string]*schema.BlockSchema{
					"validation": {
						Body: &schema.BodySchema{
							Attributes: map[string]*schema.AttributeSchema{
								"condition": {
									IsOptional: false,
									Constraint: schema.AnyExpression{OfType: cty.Bool},
								},
								"error_message": {
									IsOptional: true,
									Constraint: schema.AnyExpression{OfType: cty.String},
								},
							},
						},
					},
				},
			},
		},
		"function": {
			Labels: []*schema.LabelSchema{
				{
					Name: "functionName",
				},
			},
			Body: &schema.BodySchema{
				Attributes: map[string]*schema.AttributeSchema{
					"params": {
						IsOptional: false,
						Constraint: schema.Map{Elem: schema.LiteralType{Type: cty.String}},
					},
					"variadic_param": {
						IsOptional: true,
						Constraint: schema.LiteralType{Type: cty.String},
					},
					"result": {
						IsOptional: false,
						Constraint: schema.LiteralType{Type: cty.String},
					},
				},
			},
		},
		"target": {
			Description: lang.MarkupContent{
				Value: "A target reflects a single `docker build` invocation.",
				Kind:  lang.MarkdownKind,
			},
			Labels: []*schema.LabelSchema{
				{
					Name: "targetName",
				},
			},
			Body: &schema.BodySchema{
				Attributes: map[string]*schema.AttributeSchema{
					"args": {
						IsOptional: true,
						Constraint: schema.Map{Elem: schema.LiteralType{Type: cty.String}},
						Description: lang.MarkupContent{
							Value: "Use the `args` attribute to define build arguments for the target. This has the same effect as passing a [`--build-arg`](https://docs.docker.com/reference/cli/docker/buildx/build/#build-arg) flag to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"annotations": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
					},
					"attest": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
					},
					"cache-from": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
					},
					"cache-to": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
					},
					"call": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.String},
					},
					"context": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.String},
					},
					"contexts": {
						IsOptional: true,
						Constraint: schema.Map{Elem: schema.LiteralType{Type: cty.String}},
					},
					"description": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.String},
					},
					"dockerfile-inline": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.String},
					},
					"dockerfile": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.String},
					},
					"entitlements": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.OneOf{
							schema.LiteralValue{Value: cty.StringVal("network.host")},
							schema.LiteralValue{Value: cty.StringVal("security.insecure")},
						}},
					},
					"inherits": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
					},
					"labels": {
						IsOptional: true,
						Constraint: schema.Map{Elem: schema.LiteralType{Type: cty.String}},
					},
					"matrix": {
						IsOptional: true,
						Constraint: schema.Map{Elem: schema.List{Elem: schema.LiteralType{Type: cty.String}}},
					},
					"name": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.String},
					},
					"network": {
						IsOptional: true,
						Constraint: schema.OneOf{
							schema.LiteralValue{Value: cty.StringVal("default")},
							schema.LiteralValue{Value: cty.StringVal("host")},
							schema.LiteralValue{Value: cty.StringVal("none")},
						},
					},
					"no-cache-filter": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
					},
					"no-cache": {
						IsOptional: true,
						Constraint: schema.LiteralType{Type: cty.Number},
					},
					"output": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.OneOf{
							schema.AnyExpression{OfType: cty.String},
							schema.Object{
								Attributes: map[string]*schema.AttributeSchema{
									"type": {
										IsOptional: true,
										Constraint: schema.AnyExpression{OfType: cty.String},
									},
									"dest": {
										IsOptional: true,
										Constraint: schema.AnyExpression{OfType: cty.String},
									},
								},
							},
						}},
					},
					"platforms": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
					},
					"pull": {
						IsOptional: true,
						Constraint: schema.LiteralType{Type: cty.Bool},
					},
					"secret": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
					},
					"shm-size": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.String},
					},
					"ssh": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
					},
					"tags": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
					},
					"target": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.String},
						Description: lang.MarkupContent{
							Value: "Set the target build stage to build. This is the same as the [`--target flag`](https://docs.docker.com/reference/cli/docker/image/build/#target).",
							Kind:  lang.MarkdownKind,
						},
					},
					"ulimits": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
					},
				},
			},
		},
	},
}

type PathReaderImpl struct {
	File     *hcl.File
	Filename string
}

func (r *PathReaderImpl) Paths(ctx context.Context) []lang.Path {
	return nil
}

func (r *PathReaderImpl) PathContext(path lang.Path) (*decoder.PathContext, error) {
	return &decoder.PathContext{
		Files:  map[string]*hcl.File{r.Filename: r.File},
		Schema: BakeSchema,
	}, nil
}

func ConvertToHCLPosition(content string, line, column int) hcl.Pos {
	lines := strings.Split(content, "\n")
	offset := 0
	for i := range line {
		offset += len(lines[i]) + 1
	}
	offset += column
	return hcl.Pos{Line: line + 1, Column: column + 1, Byte: offset}
}
