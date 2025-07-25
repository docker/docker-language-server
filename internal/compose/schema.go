package compose

import (
	"bytes"
	_ "embed"
	"slices"

	"github.com/goccy/go-yaml/ast"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed compose-spec.json
var schemaData []byte

var composeSchema *jsonschema.Schema

func init() {
	schema, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaData))
	if err != nil {
		return
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", schema); err != nil {
		return
	}
	compiled, err := compiler.Compile("schema.json")
	if err != nil {
		return
	}
	composeSchema = compiled
}

func schemaProperties() map[string]*jsonschema.Schema {
	return composeSchema.Properties
}

func nodeProperties(nodes []*ast.MappingValueNode, line, column int) ([]*ast.MappingValueNode, any, bool) {
	if composeSchema != nil && slices.Contains(composeSchema.Types.ToStrings(), "object") && composeSchema.Properties != nil {
		if prop, ok := composeSchema.Properties[nodes[0].Key.GetToken().Value]; ok {
			for regexp, property := range prop.PatternProperties {
				if regexp.MatchString(nodes[1].Key.GetToken().Value) {
					if property.Ref != nil {
						return recurseNodeProperties(nodes, line, column, 2, property.Ref.Properties, false)
					}
				}
			}
		}
	}
	return nodes, nil, false
}

func recurseNodeProperties(nodes []*ast.MappingValueNode, line, column, nodeOffset int, properties map[string]*jsonschema.Schema, arrayAttributes bool) ([]*ast.MappingValueNode, any, bool) {
	if len(nodes) == nodeOffset {
		if nodes[len(nodes)-1].Key.GetToken().Position.Column >= column {
			return nodes, nil, false
		}
		return nodes, properties, arrayAttributes
	}
	if nodes[nodeOffset].Key.GetToken().Position.Column == column {
		return nodes[0:nodeOffset], properties, false
	}

	value := nodes[nodeOffset].Key.GetToken().Value
	if prop, ok := properties[value]; ok {
		if prop.Ref != nil {
			if len(prop.Ref.Properties) > 0 {
				return recurseNodeProperties(nodes, line, column, nodeOffset+1, prop.Ref.Properties, false)
			}
			// try to match the child node to patternProperties
			if len(nodes) > nodeOffset+1 {
				for regexp, property := range prop.Ref.PatternProperties {
					nextValue := nodes[nodeOffset+1].Key.GetToken().Value
					if regexp.MatchString(nextValue) {
						for _, nested := range property.OneOf {
							if slices.Contains(nested.Types.ToStrings(), "object") {
								return recurseNodeProperties(nodes, line, column, nodeOffset+2, nested.Properties, false)
							}
						}
					}
				}
			}
			if schema, ok := prop.Ref.Items.(*jsonschema.Schema); ok {
				for _, nested := range schema.OneOf {
					if nested.Types != nil && slices.Contains(nested.Types.ToStrings(), "object") {
						if len(nested.Properties) > 0 {
							return recurseNodeProperties(nodes, line, column, nodeOffset+1, nested.Properties, true)
						}
					}
				}
			}
		}

		for _, schema := range prop.OneOf {
			if schema.Types != nil && slices.Contains(schema.Types.ToStrings(), "object") {
				if len(schema.Properties) > 0 {
					return recurseNodeProperties(nodes, line, column, nodeOffset+1, schema.Properties, false)
				}

				for regexp, property := range schema.PatternProperties {
					if len(nodes) == nodeOffset+1 {
						return nodes, nil, false
					}

					nextValue := nodes[nodeOffset+1].Key.GetToken().Value
					if regexp.MatchString(nextValue) {
						for _, nested := range property.OneOf {
							if slices.Contains(nested.Types.ToStrings(), "object") {
								return recurseNodeProperties(nodes, line, column, nodeOffset+2, nested.Properties, false)
							}
						}
						return recurseNodeProperties(nodes, line, column, nodeOffset+1, property.Properties, false)
					}
				}
			}
		}

		if schema, ok := prop.Items.(*jsonschema.Schema); ok {
			for _, nested := range schema.OneOf {
				if nested.Types != nil && slices.Contains(nested.Types.ToStrings(), "object") {
					if len(nested.Properties) > 0 {
						return recurseNodeProperties(nodes, line, column, nodeOffset+1, nested.Properties, true)
					}
				}
			}
			return recurseNodeProperties(nodes, line, column, nodeOffset+1, schema.Properties, true)
		}

		if nodes[nodeOffset].Key.GetToken().Position.Column < column {
			if nodes[nodeOffset].Key.GetToken().Position.Line == line {
				return nodes, prop, false
			}
			return recurseNodeProperties(nodes, line, column, nodeOffset+1, prop.Properties, false)
		}
		return nodes, prop.Properties, false
	}
	return nodes, properties, false
}
