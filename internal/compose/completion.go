package compose

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/docker/docker-language-server/internal/hub"
	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/token"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type completionItemText struct {
	label         string
	newText       string
	documentation string
}

type textEditModifier struct {
	isInterested func(attributeName string, path []*ast.MappingValueNode) bool
	modify       func(file *ast.File, manager *document.Manager, documentPath document.DocumentPath, edit protocol.TextEdit, attributeName, spacing string, path []*ast.MappingValueNode) protocol.TextEdit
}

func samePath(uriPath, path string) bool {
	return filepath.ToSlash(uriPath) == filepath.ToSlash(path)
}

// extendingCurrentFile checks if the extends object's file attribute is
// pointing to the current file.
func extendingCurrentFile(documentPath document.DocumentPath, extendsNode *ast.MappingValueNode) bool {
	if extends, ok := extendsNode.Value.(*ast.MappingNode); ok {
		for _, extendsAttribute := range extends.Values {
			if extendsAttribute.Key.GetToken().Value == "file" {
				_, path := types.Concatenate(documentPath.Folder, extendsAttribute.Value.GetToken().Value, documentPath.WSLDollarSignHost)
				_, originalPath := types.Concatenate(documentPath.Folder, documentPath.FileName, documentPath.WSLDollarSignHost)
				if !samePath(originalPath, path) {
					return false
				}
				break
			}
		}
	}
	return true
}

var buildTargetModifier = textEditModifier{
	isInterested: func(attributeName string, path []*ast.MappingValueNode) bool {
		return attributeName == "target" && len(path) == 3 && path[2].Key.GetToken().Value == "build"
	},
	modify: func(file *ast.File, manager *document.Manager, documentPath document.DocumentPath, edit protocol.TextEdit, attributeName, spacing string, path []*ast.MappingValueNode) protocol.TextEdit {
		if _, ok := path[2].Value.(*ast.NullNode); ok {
			dockerfileURI, dockerfilePath := types.Concatenate(documentPath.Folder, "Dockerfile", documentPath.WSLDollarSignHost)
			stages := findBuildStages(manager, dockerfileURI, dockerfilePath, "")
			if len(stages) > 0 {
				edit.NewText = fmt.Sprintf("%v%v", edit.NewText, createChoiceSnippetText(stages))
				return edit
			}
		} else if mappingNode, ok := path[2].Value.(*ast.MappingNode); ok {
			dockerfileAttributePath := "Dockerfile"
			for _, buildAttribute := range mappingNode.Values {
				switch buildAttribute.Key.GetToken().Value {
				case "dockerfile_inline":
					return edit
				case "dockerfile":
					dockerfileAttributePath = buildAttribute.Value.GetToken().Value
				}
			}

			dockerfileURI, dockerfilePath := types.Concatenate(documentPath.Folder, dockerfileAttributePath, documentPath.WSLDollarSignHost)
			stages := findBuildStages(manager, dockerfileURI, dockerfilePath, "")
			if len(stages) > 0 {
				edit.NewText = fmt.Sprintf("%v%v", edit.NewText, createChoiceSnippetText(stages))
				return edit
			}
		}
		return edit
	},
}

var serviceSuggestionModifier = textEditModifier{
	isInterested: func(attributeName string, path []*ast.MappingValueNode) bool {
		return attributeName == "service" && len(path) == 3 && path[0].Key.GetToken().Value == "services" && path[2].Key.GetToken().Value == "extends"
	},
	modify: func(file *ast.File, manager *document.Manager, documentPath document.DocumentPath, edit protocol.TextEdit, attributeName, spacing string, path []*ast.MappingValueNode) protocol.TextEdit {
		if extendingCurrentFile(documentPath, path[2]) {
			services := []completionItemText{}
			for _, service := range findDependencies(file, "services") {
				if service != path[1].Key.GetToken().Value {
					services = append(services, completionItemText{newText: service})
				}
			}
			if len(services) > 0 {
				edit.NewText = fmt.Sprintf("%v%v", edit.NewText, createChoiceSnippetText(services))
			}
		}
		return edit
	},
}

var serviceProviderModifier = textEditModifier{
	isInterested: func(attributeName string, path []*ast.MappingValueNode) bool {
		return attributeName == "provider" && len(path) == 2 && path[0].Key.GetToken().Value == "services"
	},
	modify: func(file *ast.File, manager *document.Manager, documentPath document.DocumentPath, edit protocol.TextEdit, attributeName, spacing string, path []*ast.MappingValueNode) protocol.TextEdit {
		edit.NewText = fmt.Sprintf("provider:\n%vtype: ${1:model}\n%voptions:\n%v  ${2:model}: ${3:ai/example-model}", spacing, spacing, spacing)
		return edit
	},
}

var serviceProviderTypeModifier = textEditModifier{
	isInterested: func(attributeName string, path []*ast.MappingValueNode) bool {
		return attributeName == "type" && len(path) == 3 && path[0].Key.GetToken().Value == "services" && path[2].Key.GetToken().Value == "provider"
	},
	modify: func(file *ast.File, manager *document.Manager, documentPath document.DocumentPath, edit protocol.TextEdit, attributeName, spacing string, path []*ast.MappingValueNode) protocol.TextEdit {
		edit.NewText = "type: ${1:model}"
		return edit
	},
}

var textEditModifiers = []textEditModifier{buildTargetModifier, serviceSuggestionModifier, serviceProviderModifier, serviceProviderTypeModifier}

func prefix(line string, character int) string {
	sb := strings.Builder{}
	sb.Grow(character)
	for i := range character {
		if unicode.IsSpace(rune(line[i])) {
			sb.Reset()
		} else {
			sb.WriteByte(line[i])
		}
	}
	return sb.String()
}

func createSpacing(line string, character int, arrayAttributes bool) string {
	if arrayAttributes {
		// 2 more for the attribute, then 2 more for the array offset = 4 total
		return strings.Repeat(" ", character+4)
	}
	sb := strings.Builder{}
	sb.Grow(character + 2)
	for i := range character {
		if unicode.IsSpace(rune(line[i])) || line[i] == '-' {
			sb.WriteString(" ")
		}
	}
	sb.WriteString("  ")
	return sb.String()
}

func createTopLevelItems() []protocol.CompletionItem {
	items := []protocol.CompletionItem{}
	for attributeName, schema := range schemaProperties() {
		item := protocol.CompletionItem{Label: attributeName}
		if schema.Description != "" {
			item.Documentation = schema.Description
		}
		items = append(items, item)
	}
	slices.SortFunc(items, func(a, b protocol.CompletionItem) int {
		return strings.Compare(a.Label, b.Label)
	})
	return items
}

func calculateTopLevelNodeOffset(file *ast.File) int {
	if len(file.Docs) == 1 {
		if m, ok := file.Docs[0].Body.(*ast.MappingNode); ok {
			return m.Values[0].Key.GetToken().Position.Column - 1
		}
	}
	return -1
}

func Completion(ctx context.Context, params *protocol.CompletionParams, manager *document.Manager, hub *hub.Service, doc document.ComposeDocument) (*protocol.CompletionList, error) {
	documentPath, err := doc.DocumentPath()
	if err != nil {
		return nil, fmt.Errorf("LSP client sent invalid URI: %v", params.TextDocument.URI)
	}

	file := doc.File()
	if file == nil || len(file.Docs) == 0 {
		return nil, nil
	}
	if m, ok := file.Docs[0].Body.(*ast.MappingNode); ok && len(m.Values) == 0 {
		return nil, nil
	}

	lspLine := int(params.Position.Line)
	topLevelNodeOffset := calculateTopLevelNodeOffset(file)
	if topLevelNodeOffset != -1 && params.Position.Character == uint32(topLevelNodeOffset) {
		return &protocol.CompletionList{Items: createTopLevelItems()}, nil
	}

	lines := strings.Split(string(doc.Input()), "\n")
	if len(lines) <= lspLine {
		return nil, nil
	}
	currentLineTrimmed := strings.TrimSpace(lines[lspLine])
	if strings.HasPrefix(currentLineTrimmed, "#") {
		return nil, nil
	}

	character := int(params.Position.Character) + 1
	if len(lines[lspLine]) < character-1 {
		return nil, nil
	}
	whitespaceLine := currentLineTrimmed == ""
	line := int(lspLine) + 1
	path := constructCompletionNodePath(file, line)
	prefixContent := prefix(lines[lspLine], character-1)
	prefixLength := protocol.UInteger(len(prefixContent))
	if len(path) == 0 {
		if topLevelNodeOffset != -1 && params.Position.Character != uint32(topLevelNodeOffset) {
			return nil, nil
		}
		return &protocol.CompletionList{Items: createTopLevelItems()}, nil
	} else if len(path) == 1 {
		if path[0].Key.GetToken().Value == "include" {
			schema := schemaProperties()["include"].Items.(*jsonschema.Schema)
			items := createSchemaItems(params, schema.Ref.OneOf[1].Properties, lines, lspLine, whitespaceLine, prefixLength, file, manager, documentPath, path)
			items = append(items, folderStructureCompletionItems(documentPath, path, removeQuote(prefixContent))...)
			return processItems(items, whitespaceLine), nil
		}
		return nil, nil
	} else if path[1].Key.GetToken().Position.Column >= character {
		return nil, nil
	}

	path, nodeProps, arrayAttributes := nodeProperties(path, line, character)
	dependencies := dependencyCompletionItems(file, documentPath, path, params, prefixLength)
	if len(dependencies) > 0 {
		return &protocol.CompletionList{Items: dependencies}, nil
	}
	items, stop := buildTargetCompletionItems(params, manager, path, documentPath, prefixLength)
	if stop {
		return &protocol.CompletionList{Items: items}, nil
	}
	items, stop = serviceImageCompletionItems(*hub, path, prefixContent)
	if stop {
		return &protocol.CompletionList{Items: sortItems(items)}, nil
	}
	folderStructureItems := folderStructureCompletionItems(documentPath, path, removeQuote(prefixContent))
	if len(folderStructureItems) > 0 {
		return processItems(folderStructureItems, whitespaceLine && arrayAttributes), nil
	}

	items = namedDependencyCompletionItems(file, path, "configs", "configs", params, prefixLength)
	if len(items) == 0 {
		items = namedDependencyCompletionItems(file, path, "secrets", "secrets", params, prefixLength)
	}
	if len(items) == 0 {
		items = volumeDependencyCompletionItems(file, path, params, prefixLength)
	}
	schemaItems := createSchemaItems(params, nodeProps, lines, lspLine, whitespaceLine && arrayAttributes, prefixLength, file, manager, documentPath, path)
	items = append(items, schemaItems...)
	if len(items) == 0 {
		return nil, nil
	}
	return processItems(items, whitespaceLine && arrayAttributes), nil
}

func removeQuote(prefix string) string {
	if len(prefix) > 0 && (prefix[0] == 34 || prefix[0] == 39) {
		return prefix[1:]
	}
	return prefix
}

func createEnumItems(schema *jsonschema.Schema, params *protocol.CompletionParams, wordPrefixLength protocol.UInteger) []protocol.CompletionItem {
	items := []protocol.CompletionItem{}
	for _, value := range schema.Enum.Values {
		enumValue := value.(string)
		item := protocol.CompletionItem{
			Label:         enumValue,
			Documentation: schema.Description,
			Detail:        extractDetail(schema),
			TextEdit: protocol.TextEdit{
				NewText: enumValue,
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      params.Position.Line,
						Character: params.Position.Character - wordPrefixLength,
					},
					End: params.Position,
				},
			},
		}
		items = append(items, item)
	}
	return items
}

func createSchemaItems(params *protocol.CompletionParams, nodeProps any, lines []string, lspLine int, whitespacePrefixedArrayAttribute bool, wordPrefixLength protocol.UInteger, file *ast.File, manager *document.Manager, documentPath document.DocumentPath, path []*ast.MappingValueNode) []protocol.CompletionItem {
	items := []protocol.CompletionItem{}
	if schema, ok := nodeProps.(*jsonschema.Schema); ok {
		if schema.Enum != nil {
			return createEnumItems(schema, params, wordPrefixLength)
		}
	} else if properties, ok := nodeProps.(map[string]*jsonschema.Schema); ok {
		spacing := createSpacing(lines[lspLine], int(params.Position.Character), whitespacePrefixedArrayAttribute)
		for attributeName, schema := range properties {
			item := protocol.CompletionItem{
				Detail: extractDetail(schema),
				Label:  attributeName,
				TextEdit: protocol.TextEdit{
					NewText: insertText(spacing, attributeName, schema),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      params.Position.Line,
							Character: params.Position.Character - protocol.UInteger(wordPrefixLength),
						},
						End: params.Position,
					},
				},
				InsertTextMode:   types.CreateInsertTextModePointer(protocol.InsertTextModeAsIs),
				InsertTextFormat: types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet),
			}
			if schema.Description != "" {
				item.Documentation = schema.Description
			} else if schema.Ref != nil && schema.Ref.Description != "" {
				item.Documentation = schema.Ref.Description
			}

			if schema.Enum != nil {
				options := []string{}
				for i := range schema.Enum.Values {
					options = append(options, schema.Enum.Values[i].(string))
				}
				slices.Sort(options)
				sb := strings.Builder{}
				sb.WriteString(attributeName)
				sb.WriteString(": ${1|")
				for i := range options {
					sb.WriteString(options[i])
					if i != len(schema.Enum.Values)-1 {
						sb.WriteString(",")
					}
				}
				sb.WriteString("|}")
				item.TextEdit = protocol.TextEdit{
					NewText: sb.String(),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      params.Position.Line,
							Character: params.Position.Character - protocol.UInteger(wordPrefixLength),
						},
						End: params.Position,
					},
				}
			}
			item.TextEdit = modifyTextEdit(file, manager, documentPath, item.TextEdit.(protocol.TextEdit), attributeName, spacing, path)
			items = append(items, item)
		}
	}
	return items
}

func sortItems(items []protocol.CompletionItem) []protocol.CompletionItem {
	slices.SortFunc(items, func(a, b protocol.CompletionItem) int {
		return strings.Compare(a.Label, b.Label)
	})
	return items
}

func processItems(items []protocol.CompletionItem, arrayPrefix bool) *protocol.CompletionList {
	items = sortItems(items)
	if arrayPrefix {
		for i := range items {
			if edit, ok := items[i].TextEdit.(protocol.TextEdit); ok {
				items[i].TextEdit = protocol.TextEdit{
					NewText: fmt.Sprintf("%v%v", "- ", edit.NewText),
					Range:   edit.Range,
				}
			}
		}
	}
	return &protocol.CompletionList{Items: items}
}

func createChoiceSnippetText(itemTexts []completionItemText) string {
	sb := strings.Builder{}
	sb.WriteString("${1|")
	for i, stage := range itemTexts {
		sb.WriteString(stage.newText)
		if i != len(itemTexts)-1 {
			sb.WriteString(",")
		}
	}
	sb.WriteString("|}")
	return sb.String()
}

func modifyTextEdit(file *ast.File, manager *document.Manager, documentPath document.DocumentPath, edit protocol.TextEdit, attributeName, spacing string, path []*ast.MappingValueNode) protocol.TextEdit {
	for _, modified := range textEditModifiers {
		if modified.isInterested(attributeName, path) {
			return modified.modify(file, manager, documentPath, edit, attributeName, spacing, path)
		}
	}
	return edit
}

func folderStructureCompletionItems(documentPath document.DocumentPath, path []*ast.MappingValueNode, prefix string) []protocol.CompletionItem {
	folder, hideFiles := directoryForNode(documentPath, path, prefix)
	return types.FileStructureCompletionItems(folder, hideFiles)
}

func directoryForNode(documentPath document.DocumentPath, path []*ast.MappingValueNode, prefix string) (folder string, hideFiles bool) {
	if len(path) == 1 && path[0].Key.GetToken().Value == "include" {
		return document.DirectoryForPrefix(documentPath, prefix, documentPath.Folder, false), false
	} else if len(path) == 2 {
		// include:
		//   - env_file: ...
		//   - env_file:
		//       - ...
		//   - path: ...
		//   - path:
		//       - ...
		if path[0].Key.GetToken().Value == "include" {
			if path[1].Key.GetToken().Value == "env_file" {
				return document.DirectoryForPrefix(documentPath, prefix, documentPath.Folder, false), false
			}
			if path[1].Key.GetToken().Value == "path" {
				return document.DirectoryForPrefix(documentPath, prefix, documentPath.Folder, false), false
			}
		}
	} else if len(path) == 3 {
		switch path[0].Key.GetToken().Value {
		case "services":
			// services:
			//   serviceA:
			//     env_file: ...
			//     label_file: ...
			//     volumes:
			//       - ...
			switch path[2].Key.GetToken().Value {
			case "env_file":
				return document.DirectoryForPrefix(documentPath, prefix, documentPath.Folder, false), false
			case "label_file":
				return document.DirectoryForPrefix(documentPath, prefix, documentPath.Folder, false), false
			case "volumes":
				return document.DirectoryForPrefix(documentPath, prefix, "", true), false
			}
		case "configs":
			// configs:
			//   configA:
			//     file: ...
			if path[2].Key.GetToken().Value == "file" {
				return document.DirectoryForPrefix(documentPath, prefix, documentPath.Folder, false), false
			}
		case "secrets":
			// secrets:
			//   secretA:
			//     file: ...
			if path[2].Key.GetToken().Value == "file" {
				return document.DirectoryForPrefix(documentPath, prefix, documentPath.Folder, false), false
			}
		}
	} else if len(path) == 4 && path[0].Key.GetToken().Value == "services" {
		// services:
		//   serviceA:
		//     build:
		//       context: ... (folders only)
		//       dockerfile: ...
		//     credential_spec:
		//       file: ...
		//     env_file:
		//       - path: ...
		//     extends:
		//       file: ...
		//     volumes:
		//       - type: bind
		//         source: ...
		if path[2].Key.GetToken().Value == "build" {
			if path[3].Key.GetToken().Value == "context" {
				return document.DirectoryForPrefix(documentPath, prefix, documentPath.Folder, false), true
			} else if path[3].Key.GetToken().Value == "dockerfile" {
				return document.DirectoryForPrefix(documentPath, prefix, documentPath.Folder, false), false
			}
		} else if (path[2].Key.GetToken().Value == "extends" || path[2].Key.GetToken().Value == "credential_spec") && path[3].Key.GetToken().Value == "file" {
			return document.DirectoryForPrefix(documentPath, prefix, documentPath.Folder, false), false
		} else if path[2].Key.GetToken().Value == "env_file" && path[3].Key.GetToken().Value == "path" {
			if _, ok := path[2].Value.(*ast.SequenceNode); ok {
				return document.DirectoryForPrefix(documentPath, prefix, documentPath.Folder, false), false
			}
		} else if path[2].Key.GetToken().Value == "volumes" && path[3].Key.GetToken().Value == "source" {
			if volumes, ok := path[2].Value.(*ast.SequenceNode); ok {
				for _, node := range volumes.Values {
					if volume, ok := node.(*ast.MappingNode); ok {
						if slices.Contains(volume.Values, path[3]) {
							for _, property := range volume.Values {
								if property.Key.GetToken().Value == "type" && property.Value.GetToken().Value == "bind" {
									return document.DirectoryForPrefix(documentPath, prefix, documentPath.Folder, true), false
								}
							}
							return "", false
						}
					}
				}
			}
		}
	}
	return "", false
}

func findDependencies(file *ast.File, dependencyType string) []string {
	services := []string{}
	for _, documentNode := range file.Docs {
		if mappingNode, ok := documentNode.Body.(*ast.MappingNode); ok {
			for _, n := range mappingNode.Values {
				if s, ok := n.Key.(*ast.StringNode); ok {
					if s.Value == dependencyType {
						if mappingNode, ok := n.Value.(*ast.MappingNode); ok {
							for _, service := range mappingNode.Values {
								services = append(services, service.Key.GetToken().Value)
							}
						}
					}
				}
			}
		}
	}
	return services
}

func findBuildStages(manager *document.Manager, dockerfileURI, dockerfilePath, prefix string) []completionItemText {
	_, nodes := document.OpenDockerfile(context.Background(), manager, dockerfileURI, dockerfilePath)
	items := []completionItemText{}
	for _, child := range nodes {
		if strings.EqualFold(child.Value, "FROM") {
			if child.Next != nil && child.Next.Next != nil && strings.EqualFold(child.Next.Next.Value, "AS") && child.Next.Next.Next != nil {
				buildStage := child.Next.Next.Next.Value
				if strings.HasPrefix(buildStage, prefix) {
					items = append(items, completionItemText{
						label:         buildStage,
						documentation: child.Next.Value,
						newText:       buildStage,
					})
				}
			}
		}
	}
	return items
}

func buildTargetCompletionItems(params *protocol.CompletionParams, manager *document.Manager, path []*ast.MappingValueNode, documentPath document.DocumentPath, prefixLength protocol.UInteger) ([]protocol.CompletionItem, bool) {
	if len(path) == 4 && path[2].Key.GetToken().Value == "build" && path[3].Key.GetToken().Value == "target" {
		if mappingNode, ok := path[2].Value.(*ast.MappingNode); ok {
			dockerfileAttributePath := "Dockerfile"
			for _, buildAttribute := range mappingNode.Values {
				switch buildAttribute.Key.GetToken().Value {
				case "dockerfile_inline":
					return nil, true
				case "dockerfile":
					dockerfileAttributePath = buildAttribute.Value.GetToken().Value
				}
			}

			dockerfileURI, dockerfilePath := types.Concatenate(documentPath.Folder, dockerfileAttributePath, documentPath.WSLDollarSignHost)
			if _, ok := path[3].Value.(*ast.NullNode); ok {
				return createBuildStageItems(params, manager, dockerfileURI, dockerfilePath, "", prefixLength), true
			} else if prefix, ok := path[3].Value.(*ast.StringNode); ok {
				if int(params.Position.Line) == path[3].Value.GetToken().Position.Line-1 {
					offset := int(params.Position.Character) - path[3].Value.GetToken().Position.Column + 1
					// offset can be greater than the length if there's just empty whitespace after the string value,
					// must be non-negative, if negative it suggests the cursor is in the whitespace before the attribute's value
					if offset >= 0 && offset <= len(prefix.Value) {
						return createBuildStageItems(params, manager, dockerfileURI, dockerfilePath, prefix.Value[0:offset], prefixLength), true
					}
				}
			}
		}
	}
	return nil, false
}

func hubRepositoryImage(imageValue string) (repository, image, tagPrefix string) {
	idx := strings.Index(imageValue, ":")
	if idx == -1 {
		return "", "", ""
	}
	slashIndex := strings.Index(imageValue, "/")
	if slashIndex != strings.LastIndex(imageValue, "/") {
		return "", "", ""
	}
	split := strings.Split(imageValue[0:idx], "/")
	if len(split) == 1 {
		return "library", split[0], imageValue[idx+1:]
	}
	return split[0], split[1], imageValue[idx+1:]
}

func serviceImageCompletionItems(hub hub.Service, path []*ast.MappingValueNode, prefix string) ([]protocol.CompletionItem, bool) {
	if len(path) == 3 && path[0].Key.GetToken().Value == "services" && path[2].Key.GetToken().Value == "image" {
		if path[2].Value.GetToken().Type == token.DoubleQuoteType || path[2].Value.GetToken().Type == token.SingleQuoteType {
			prefix = prefix[1:]
		}
		repository, image, tagPrefix := hubRepositoryImage(prefix)
		if repository != "" {
			tags, _ := hub.GetTags(repository, image)
			items := []protocol.CompletionItem{}
			for _, tag := range tags {
				if strings.HasPrefix(tag, tagPrefix) {
					items = append(items, protocol.CompletionItem{
						Label: tag,
						Kind:  types.CreateCompletionItemKindPointer(protocol.CompletionItemKindModule),
					})
				}
			}
			return items, true
		}
	}
	return nil, false
}

func createBuildStageItems(params *protocol.CompletionParams, manager *document.Manager, dockerfileURI, dockerfilePath, prefix string, prefixLength protocol.UInteger) []protocol.CompletionItem {
	items := []protocol.CompletionItem{}
	for _, itemText := range findBuildStages(manager, dockerfileURI, dockerfilePath, prefix) {
		items = append(items, protocol.CompletionItem{
			Label:         itemText.label,
			Documentation: itemText.documentation,
			TextEdit: protocol.TextEdit{
				NewText: itemText.newText,
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      params.Position.Line,
						Character: params.Position.Character - prefixLength,
					},
					End: params.Position,
				},
			},
		})
	}
	return items
}

func dependencyCompletionItems(file *ast.File, documentPath document.DocumentPath, path []*ast.MappingValueNode, params *protocol.CompletionParams, prefixLength protocol.UInteger) []protocol.CompletionItem {
	dependency := map[string]string{
		"depends_on": "services",
		"models":     "models",
		"networks":   "networks",
	}
	for serviceAttribute, dependencyType := range dependency {
		items := namedDependencyCompletionItems(file, path, serviceAttribute, dependencyType, params, prefixLength)
		if len(items) > 0 {
			return items
		}
	}
	if len(path) >= 3 && path[2].Key.GetToken().Value == "extends" && path[0].Key.GetToken().Value == "services" {
		if (len(path) == 4 && path[3].Key.GetToken().Value == "service") || params.Position.Line == protocol.UInteger(path[2].Key.GetToken().Position.Line)-1 {
			if !extendingCurrentFile(documentPath, path[2]) {
				return nil
			}

			items := []protocol.CompletionItem{}
			for _, service := range findDependencies(file, "services") {
				if service != path[1].Key.GetToken().Value {
					item := protocol.CompletionItem{
						Label: service,
						TextEdit: protocol.TextEdit{
							NewText: service,
							Range: protocol.Range{
								Start: protocol.Position{
									Line:      params.Position.Line,
									Character: params.Position.Character - prefixLength,
								},
								End: params.Position,
							},
						},
					}
					items = append(items, item)
				}
			}
			return items
		}
	}
	return nil
}

func volumeDependencyCompletionItems(
	file *ast.File,
	path []*ast.MappingValueNode,
	params *protocol.CompletionParams,
	prefixLength protocol.UInteger,
) []protocol.CompletionItem {
	items := namedDependencyCompletionItems(file, path, "volumes", "volumes", params, prefixLength)
	for i := range items {
		edit := items[i].TextEdit.(protocol.TextEdit)
		items[i].TextEdit = protocol.TextEdit{
			NewText: fmt.Sprintf("%v:${1:/container/path}", edit.NewText),
			Range:   edit.Range,
		}
		items[i].InsertTextFormat = types.CreateInsertTextFormatPointer(protocol.InsertTextFormatSnippet)
	}
	return items
}

func namedDependencyCompletionItems(file *ast.File, path []*ast.MappingValueNode, serviceAttribute, dependencyType string, params *protocol.CompletionParams, prefixLength protocol.UInteger) []protocol.CompletionItem {
	if len(path) == 3 && path[2].Key.GetToken().Value == serviceAttribute {
		items := []protocol.CompletionItem{}
		for _, service := range findDependencies(file, dependencyType) {
			if service != path[1].Key.GetToken().Value {
				item := protocol.CompletionItem{
					Label: service,
					TextEdit: protocol.TextEdit{
						NewText: service,
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      params.Position.Line,
								Character: params.Position.Character - prefixLength,
							},
							End: params.Position,
						},
					},
				}
				items = append(items, item)
			}
		}
		return items
	}
	return nil
}

func constructCompletionNodePath(file *ast.File, line int) []*ast.MappingValueNode {
	for i := range len(file.Docs) {
		if i+1 == len(file.Docs) {
			if mappingNode, ok := file.Docs[i].Body.(*ast.MappingNode); ok {
				return NodeStructure(line, mappingNode.Values)
			}
		}

		if m, ok := file.Docs[i].Body.(*ast.MappingNode); ok {
			if n, ok := file.Docs[i+1].Body.(*ast.MappingNode); ok {
				if m.Values[0].Key.GetToken().Position.Line <= line && line <= n.Values[0].Key.GetToken().Position.Line {
					return NodeStructure(line, m.Values)
				}
			}
		}
	}
	return nil
}

func NodeStructure(line int, rootNodes []*ast.MappingValueNode) []*ast.MappingValueNode {
	if len(rootNodes) == 0 {
		return nil
	}

	var candidate *ast.MappingValueNode
	for _, node := range rootNodes {
		if node.GetToken().Position.Line < line {
			candidate = node
		} else if node.GetToken().Position.Line == line {
			return []*ast.MappingValueNode{node}
		} else if candidate == nil {
			return []*ast.MappingValueNode{}
		} else {
			break
		}
	}
	nodes := []*ast.MappingValueNode{candidate}
	candidates := walkNodes(line, candidate)
	nodes = append(nodes, candidates...)
	return nodes
}

func walkNodes(line int, node *ast.MappingValueNode) []*ast.MappingValueNode {
	var candidate ast.Node
	value := node.Value
	if mappingNode, ok := value.(*ast.MappingNode); ok {
		for _, child := range mappingNode.Values {
			if child.GetToken().Position.Line < line {
				candidate = child
			} else if child.GetToken().Position.Line == line {
				candidate = child
				break
			}
		}
	} else if sequenceNode, ok := value.(*ast.SequenceNode); ok {
		for _, child := range sequenceNode.Values {
			if child.GetToken().Position.Line < line {
				if _, ok := child.(*ast.NullNode); ok {
					continue
				}
				candidate = child
			} else if child.GetToken().Position.Line == line {
				if _, ok := child.(*ast.NullNode); ok {
					break
				}
				candidate = child
				break
			}
		}
	}

	if mappingNode, ok := candidate.(*ast.MappingNode); ok {
		for _, child := range mappingNode.Values {
			if child.GetToken().Position.Line < line {
				candidate = child
			} else if child.GetToken().Position.Line == line {
				candidate = child
				break
			}
		}
	}

	if candidate == nil {
		return []*ast.MappingValueNode{}
	}

	if next, ok := candidate.(*ast.MappingValueNode); ok {
		nodes := []*ast.MappingValueNode{next}
		candidates := walkNodes(line, next)
		nodes = append(nodes, candidates...)
		return nodes
	}
	return []*ast.MappingValueNode{}
}

func referencedTypes(schema *jsonschema.Schema) []string {
	if schema.Types != nil {
		return schema.Types.ToStrings()
	} else if schema.Ref != nil {
		if schema.Ref.Types != nil {
			return schema.Ref.Types.ToStrings()
		}
		schema = schema.Ref
	}
	schemaTypes := []string{}
	for _, referenced := range schema.OneOf {
		if referenced.Types != nil {
			schemaTypes = append(schemaTypes, referenced.Types.ToStrings()[0])
		} else if referenced.Ref != nil {
			schemaTypes = append(schemaTypes, referenced.Ref.Types.ToStrings()[0])
		}
	}
	return schemaTypes
}

func extractDetail(schema *jsonschema.Schema) *string {
	schemaTypes := referencedTypes(schema)
	slices.Sort(schemaTypes)
	return types.CreateStringPointer(strings.Join(schemaTypes, " or "))
}

func requiredFieldsText(spacing string, schema *jsonschema.Schema, schemaTypes []string) []string {
	if len(schemaTypes) == 1 {
		if slices.Contains(schemaTypes, "array") {
			if schema.Ref != nil {
				schema = schema.Ref
			}
			if itemSchema, ok := schema.Items.(*jsonschema.Schema); ok {
				if itemSchema.Ref != nil {
					itemSchema = itemSchema.Ref
				}
				if itemSchema.Types != nil {
					if slices.Contains(itemSchema.Types.ToStrings(), "object") {
						requiredTexts := []string{}
						for _, r := range itemSchema.Required {
							requiredTexts = append(requiredTexts, insertText(fmt.Sprintf("%v  ", spacing), r, itemSchema.Properties[r]))
						}
						return requiredTexts
					}
				}
			}
		}
	}
	return nil
}

func insertText(spacing, attributeName string, schema *jsonschema.Schema) string {
	schemaTypes := referencedTypes(schema)
	if slices.Contains(schemaTypes, "array") {
		if len(schemaTypes) == 1 {
			required := requiredFieldsText(spacing, schema, schemaTypes)
			if len(required) > 0 {
				slices.Sort(required)
				sb := strings.Builder{}
				sb.WriteString(attributeName)
				sb.WriteString(":")
				for i, requiredAttribute := range required {
					sb.WriteString("\n")
					sb.WriteString(spacing)
					if i == 0 {
						sb.WriteString("- ")
					} else {
						sb.WriteString("  ")
					}
					sb.WriteString(requiredAttribute)
					if len(required) != 1 {
						sb.WriteString(fmt.Sprintf("${%v}", i+1))
					}
				}
				return sb.String()
			}
			return fmt.Sprintf("%v:\n%v- ", attributeName, spacing)
		} else if len(schemaTypes) == 2 && slices.Contains(schemaTypes, "object") {
			return fmt.Sprintf("%v:\n%v", attributeName, spacing)
		}
		return fmt.Sprintf("%v:", attributeName)
	}
	if slices.Contains(schemaTypes, "object") {
		if len(schemaTypes) == 1 {
			return fmt.Sprintf("%v:\n%v", attributeName, spacing)
		}
		return fmt.Sprintf("%v:", attributeName)
	}
	if slices.Contains(schemaTypes, "boolean") {
		return fmt.Sprintf("%v: ${1|true,false|}", attributeName)
	}
	return fmt.Sprintf("%v: ", attributeName)
}
