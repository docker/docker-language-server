package parser

import (
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/zclconf/go-cty/cty"
)

var BakeSchema = &schema.BodySchema{
	Blocks: map[string]*schema.BlockSchema{
		"group": {
			Description: lang.MarkupContent{
				Value: "A group is a grouping of targets. Groups allow you to build multiple targets together.",
				Kind:  lang.MarkdownKind,
			},
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
						Description: lang.MarkupContent{
							Value: "Override the name of the group. If not specified, the group name from the label is used.",
							Kind:  lang.MarkdownKind,
						},
					},
					"description": {
						IsOptional: true,
						Constraint: schema.LiteralType{Type: cty.String},
						Description: lang.MarkupContent{
							Value: "A description for the group that will be shown in the help output.",
							Kind:  lang.MarkdownKind,
						},
					},
					"targets": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
						Description: lang.MarkupContent{
							Value: "List of targets that belong to this group. When building the group, all specified targets will be built.",
							Kind:  lang.MarkdownKind,
						},
					},
				},
			},
		},
		"variable": {
			Description: lang.MarkupContent{
				Value: "A variable defines an input parameter that can be used throughout the Bake file.",
				Kind:  lang.MarkdownKind,
			},
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
						Description: lang.MarkupContent{
							Value: "Default value to use for the variable if no value is provided. Can be a string, number, boolean, or list.",
							Kind:  lang.MarkdownKind,
						},
					},
				},
				Blocks: map[string]*schema.BlockSchema{
					"validation": {
						Description: lang.MarkupContent{
							Value: "Validation rules for the variable to ensure the provided value meets certain criteria.",
							Kind:  lang.MarkdownKind,
						},
						Body: &schema.BodySchema{
							Attributes: map[string]*schema.AttributeSchema{
								"condition": {
									IsOptional: false,
									Constraint: schema.AnyExpression{OfType: cty.Bool},
									Description: lang.MarkupContent{
										Value: "A boolean expression that must evaluate to true for the variable value to be valid.",
										Kind:  lang.MarkdownKind,
									},
								},
								"error_message": {
									IsOptional: true,
									Constraint: schema.AnyExpression{OfType: cty.String},
									Description: lang.MarkupContent{
										Value: "Custom error message to display when the validation condition fails.",
										Kind:  lang.MarkdownKind,
									},
								},
							},
						},
					},
				},
			},
		},
		"function": {
			Description: lang.MarkupContent{
				Value: "A function defines a reusable block of HCL that can be called from other parts of the Bake file.",
				Kind:  lang.MarkdownKind,
			},
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
						Description: lang.MarkupContent{
							Value: "Map of parameter names to their types that the function accepts.",
							Kind:  lang.MarkdownKind,
						},
					},
					"variadic_param": {
						IsOptional: true,
						Constraint: schema.LiteralType{Type: cty.String},
						Description: lang.MarkupContent{
							Value: "Name of a variadic parameter that can accept multiple arguments of the specified type.",
							Kind:  lang.MarkdownKind,
						},
					},
					"result": {
						IsOptional: false,
						Constraint: schema.LiteralType{Type: cty.String},
						Description: lang.MarkupContent{
							Value: "HCL expression that defines what the function returns when called.",
							Kind:  lang.MarkdownKind,
						},
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
						Description: lang.MarkupContent{
							Value: "Add annotations to the image. This has the same effect as passing [`--annotation`](https://docs.docker.com/reference/cli/docker/buildx/build/#annotation) flags to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"attest": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
						Description: lang.MarkupContent{
							Value: "Add attestations to the image. This has the same effect as passing [`--attest`](https://docs.docker.com/reference/cli/docker/buildx/build/#attest) flags to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"cache-from": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
						Description: lang.MarkupContent{
							Value: "External cache sources for the build. This has the same effect as passing [`--cache-from`](https://docs.docker.com/reference/cli/docker/buildx/build/#cache-from) flags to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"cache-to": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
						Description: lang.MarkupContent{
							Value: "External cache destinations for the build. This has the same effect as passing [`--cache-to`](https://docs.docker.com/reference/cli/docker/buildx/build/#cache-to) flags to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"call": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.String},
						Description: lang.MarkupContent{
							Value: "Set the call method for the target. Can be `build`, `check`, or a custom method.",
							Kind:  lang.MarkdownKind,
						},
					},
					"context": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.String},
						Description: lang.MarkupContent{
							Value: "Build context path. This has the same effect as passing the context argument to the [`docker buildx build`](https://docs.docker.com/reference/cli/docker/buildx/build/) command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"contexts": {
						IsOptional: true,
						Constraint: schema.Map{Elem: schema.LiteralType{Type: cty.String}},
						Description: lang.MarkupContent{
							Value: "Additional build contexts for the build. This has the same effect as passing [`--build-context`](https://docs.docker.com/reference/cli/docker/buildx/build/#build-context) flags to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"description": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.String},
						Description: lang.MarkupContent{
							Value: "A description for the target that will be shown in the help output.",
							Kind:  lang.MarkdownKind,
						},
					},
					"dockerfile-inline": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.String},
						Description: lang.MarkupContent{
							Value: "Inline Dockerfile content instead of reading from a file. This has the same effect as passing [`--file -`](https://docs.docker.com/reference/cli/docker/buildx/build/#file) to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"dockerfile": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.String},
						Description: lang.MarkupContent{
							Value: "Path to the Dockerfile to use for the build. This has the same effect as passing [`--file`](https://docs.docker.com/reference/cli/docker/buildx/build/#file) flag to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"entitlements": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.OneOf{
							schema.LiteralValue{Value: cty.StringVal("network.host")},
							schema.LiteralValue{Value: cty.StringVal("security.insecure")},
						}},
						Description: lang.MarkupContent{
							Value: "Allow extra privileged entitlements for the build. This has the same effect as passing [`--allow`](https://docs.docker.com/reference/cli/docker/buildx/build/#allow) flags to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"inherits": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
						Description: lang.MarkupContent{
							Value: "List of other targets to inherit attributes from. Attributes from inherited targets will be merged with this target's attributes.",
							Kind:  lang.MarkdownKind,
						},
					},
					"labels": {
						IsOptional: true,
						Constraint: schema.Map{Elem: schema.LiteralType{Type: cty.String}},
						Description: lang.MarkupContent{
							Value: "Add metadata labels to the image. This has the same effect as passing [`--label`](https://docs.docker.com/reference/cli/docker/buildx/build/#label) flags to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"matrix": {
						IsOptional: true,
						Constraint: schema.Map{Elem: schema.List{Elem: schema.AnyExpression{OfType: cty.String}}},
						Description: lang.MarkupContent{
							Value: "Define a matrix of values to create multiple variants of this target. Each combination of matrix values will create a separate build.",
							Kind:  lang.MarkdownKind,
						},
					},
					"name": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.String},
						Description: lang.MarkupContent{
							Value: "Override the name of the target. If not specified, the target name from the label is used.",
							Kind:  lang.MarkdownKind,
						},
					},
					"network": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.String},
						Description: lang.MarkupContent{
							Value: "Set the networking mode for RUN instructions. This has the same effect as passing [`--network`](https://docs.docker.com/reference/cli/docker/buildx/build/#network) flag to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"no-cache": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.Bool},
						Description: lang.MarkupContent{
							Value: "Do not use cache when building the image. This has the same effect as passing [`--no-cache`](https://docs.docker.com/reference/cli/docker/buildx/build/#no-cache) flag to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"no-cache-filter": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
						Description: lang.MarkupContent{
							Value: "Do not use cache for specified stages. This has the same effect as passing [`--no-cache-filter`](https://docs.docker.com/reference/cli/docker/buildx/build/#no-cache-filter) flags to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"output": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
						Description: lang.MarkupContent{
							Value: "Output destinations for the build result. This has the same effect as passing [`--output`](https://docs.docker.com/reference/cli/docker/buildx/build/#output) flags to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"platforms": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
						Description: lang.MarkupContent{
							Value: "Target platforms for the build. This has the same effect as passing [`--platform`](https://docs.docker.com/reference/cli/docker/buildx/build/#platform) flags to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"pull": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.Bool},
						Description: lang.MarkupContent{
							Value: "Always attempt to pull newer versions of base images. This has the same effect as passing [`--pull`](https://docs.docker.com/reference/cli/docker/buildx/build/#pull) flag to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"push": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.Bool},
						Description: lang.MarkupContent{
							Value: "Push the built image to a registry. This has the same effect as passing [`--push`](https://docs.docker.com/reference/cli/docker/buildx/build/#push) flag to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"resource": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.String},
						Description: lang.MarkupContent{
							Value: "Override the configured resource limits for the build.",
							Kind:  lang.MarkdownKind,
						},
					},
					"secret": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
						Description: lang.MarkupContent{
							Value: "Secrets to expose to the build. This has the same effect as passing [`--secret`](https://docs.docker.com/reference/cli/docker/buildx/build/#secret) flags to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"shm-size": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.String},
						Description: lang.MarkupContent{
							Value: "Size of `/dev/shm` for RUN instructions. This has the same effect as passing [`--shm-size`](https://docs.docker.com/reference/cli/docker/buildx/build/#shm-size) flag to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"ssh": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
						Description: lang.MarkupContent{
							Value: "SSH agent sockets or keys to expose to the build. This has the same effect as passing [`--ssh`](https://docs.docker.com/reference/cli/docker/buildx/build/#ssh) flags to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"tags": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
						Description: lang.MarkupContent{
							Value: "Image names and tags for the built image. This has the same effect as passing [`--tag`](https://docs.docker.com/reference/cli/docker/buildx/build/#tag) flags to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"target": {
						IsOptional: true,
						Constraint: schema.AnyExpression{OfType: cty.String},
						Description: lang.MarkupContent{
							Value: "Set the target build stage to build. This has the same effect as passing [`--target`](https://docs.docker.com/reference/cli/docker/buildx/build/#target) flag to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
					"ulimits": {
						IsOptional: true,
						Constraint: schema.List{Elem: schema.AnyExpression{OfType: cty.String}},
						Description: lang.MarkupContent{
							Value: "Ulimit options for the build container. This has the same effect as passing [`--ulimit`](https://docs.docker.com/reference/cli/docker/buildx/build/#ulimit) flags to the build command.",
							Kind:  lang.MarkdownKind,
						},
					},
				},
			},
		},
	},
}