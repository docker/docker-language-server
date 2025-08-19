package document

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sync"

	"github.com/docker/buildx/bake"
	"github.com/docker/docker-language-server/internal/bake/hcl/parser"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"go.lsp.dev/uri"
)

type BakeHCLDocument interface {
	Document
	Decoder() *decoder.PathDecoder
	File() *hcl.File
	DockerfileForTarget(block *hclsyntax.Block) (dockerfileURI string, dockerfileAbsolutePath string, err error)
	ParentTargets(target string) ([]string, bool)
}

type BakePrintOutput struct {
	Group  map[string]bake.Group  `json:"group,omitempty"`
	Target map[string]bake.Target `json:"target"`
}

func NewBakeHCLDocument(u uri.URI, version int32, input []byte) BakeHCLDocument {
	doc := &bakeHCLDocument{
		document: document{
			uri:        u,
			identifier: protocol.DockerBakeLanguage,
			version:    version,
			input:      input,
		},
	}
	doc.copyFn = doc.copy
	doc.parseFn = doc.parse
	doc.parseFn(true)
	return doc
}

type bakeHCLDocument struct {
	document
	mutex           sync.Mutex
	decoder         *decoder.PathDecoder
	file            *hcl.File
	bakePrintOutput *BakePrintOutput
}

func (d *bakeHCLDocument) parse(_ bool) bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	file, _ := hclsyntax.ParseConfig(d.input, string(d.uri), hcl.InitialPos)
	decoder := decoder.NewDecoder(&parser.PathReaderImpl{Filename: string(d.uri), File: file})
	pd, _ := decoder.Path(lang.Path{})
	pd.PrefillRequiredFields = true
	d.decoder = pd
	d.file = file
	d.extractBakeOutput()
	return true
}

func (d *bakeHCLDocument) copy() Document {
	return NewBakeHCLDocument(d.uri, d.version, d.input)
}

func (d *dockerfileDocument) copy() Document {
	return NewDockerfileDocument(d.uri, d.version, d.input)
}

func (d *bakeHCLDocument) File() *hcl.File {
	return d.file
}

func (d *bakeHCLDocument) Decoder() *decoder.PathDecoder {
	return d.decoder
}

func (d *bakeHCLDocument) findParentTargets(targets []string, target string) ([]string, bool) {
	if slices.Contains(targets, target) {
		return targets, false
	} else {
		targets = append(targets, target)
	}

	body := d.file.Body.(*hclsyntax.Body)
	found := false
	for _, block := range body.Blocks {
		if block.Type == "target" && len(block.Labels) > 0 && block.Labels[0] == target {
			found = true
			if attr, ok := block.Body.Attributes["inherits"]; ok {
				if tupleConsExpr, ok := attr.Expr.(*hclsyntax.TupleConsExpr); ok {
					if len(tupleConsExpr.Exprs) == 0 {
						return targets, true
					}

					for _, e := range tupleConsExpr.Exprs {
						if templateExpr, ok := e.(*hclsyntax.TemplateExpr); ok {
							if templateExpr.IsStringLiteral() {
								value, _ := templateExpr.Value(&hcl.EvalContext{})
								parent := value.AsString()
								newTargets, resolved := d.findParentTargets(targets, parent)
								if !resolved {
									return nil, false
								}
								targets = newTargets
							} else {
								return nil, false
							}
						} else {
							return nil, false
						}
					}
					return targets, true
				}
				return nil, false
			}
			break
		}
	}
	if !found {
		return nil, false
	}
	return targets, true
}

func (d *bakeHCLDocument) ParentTargets(target string) ([]string, bool) {
	body, ok := d.file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, true
	}

	for _, block := range body.Blocks {
		if block.Type == "target" && len(block.Labels) > 0 && block.Labels[0] == target {
			if attr, ok := block.Body.Attributes["inherits"]; ok {
				if _, ok := attr.Expr.(*hclsyntax.TupleConsExpr); ok {
					parents, resolved := d.findParentTargets([]string{}, target)
					if !resolved {
						return nil, false
					}
					idx := slices.Index(parents, target)
					parents[idx] = parents[len(parents)-1]
					return parents[:len(parents)-1], true
				}
				return nil, false
			}
			return nil, true
		}
	}
	return nil, true
}

func (d *bakeHCLDocument) extractBakeOutput() {
	body, ok := d.File().Body.(*hclsyntax.Body)
	if !ok {
		d.bakePrintOutput = nil
		return
	}

	targets := []string{}
	for _, b := range body.Blocks {
		if len(b.Labels) == 1 {
			if b.Type == "target" {
				targets = append(targets, b.Labels[0])
			}
		} else {
			// if the block's label count is not 1 Bake will not parse the file
			d.bakePrintOutput = nil
			return
		}
	}

	dd, err := d.DocumentPath()
	if err != nil {
		d.bakePrintOutput = nil
		return
	}
	btargets, groups, err := bake.ReadTargets(
		context.Background(),
		[]bake.File{{Name: dd.FileName, Data: d.Input()}},
		targets,
		nil,
		nil,
		nil,
	)
	if err != nil {
		d.bakePrintOutput = nil
		return
	}

	checkedGroups := map[string]bake.Group{}
	checkedTargets := map[string]bake.Target{}
	for target, value := range btargets {
		if value != nil {
			checkedTargets[target] = *value
		}
	}
	for group, value := range groups {
		if value != nil {
			checkedGroups[group] = *value
		}
	}
	d.bakePrintOutput = &BakePrintOutput{Group: checkedGroups, Target: checkedTargets}
}

type DockerfileDefinition int

const (
	Undefined DockerfileDefinition = iota
	Unresolvable
	Resolvable
)

func resolvableDockerfile(block *hclsyntax.Block) DockerfileDefinition {
	state := Undefined
	// if the context or dockerfile attributes are not simple strings, do not try to resolve
	if contextAttribute, ok := block.Body.Attributes["context"]; ok {
		if expr, ok := contextAttribute.Expr.(*hclsyntax.TemplateExpr); ok {
			if len(expr.Parts) != 1 {
				return Unresolvable
			}
			state = Resolvable
		} else {
			return Unresolvable
		}
	}

	if dockerfileAttribute, ok := block.Body.Attributes["dockerfile"]; ok {
		if expr, ok := dockerfileAttribute.Expr.(*hclsyntax.TemplateExpr); ok {
			if len(expr.Parts) != 1 {
				return Unresolvable
			}
			return Resolvable
		}
		return Unresolvable
	}
	return state
}

func (d *bakeHCLDocument) DockerfileForTarget(block *hclsyntax.Block) (dockerfileURI string, dockerfileAbsolutePath string, err error) {
	if d.bakePrintOutput == nil || len(block.Labels) != 1 {
		return "", "", errors.New("cannot parse Bake file")
	}

	switch resolvableDockerfile(block) {
	case Undefined:
		targets, _ := d.ParentTargets(block.Labels[0])
		for _, target := range targets {
			body := d.file.Body.(*hclsyntax.Body)
			for _, b := range body.Blocks {
				if len(b.Labels) == 1 && b.Labels[0] == target && resolvableDockerfile(b) == Unresolvable {
					return "", "", nil
				}
			}
		}
	}

	path, _ := d.DocumentPath()
	if target, ok := d.bakePrintOutput.Target[block.Labels[0]]; ok {
		if target.DockerfileInline != nil {
			return "", "", errors.New("dockerfile-inline defined")
		}
		uri, file := types.Concatenate(filepath.Join(path.Folder, *target.Context), *target.Dockerfile, path.WSLDollarSignHost)
		return uri, file, nil
	}
	return "", "", fmt.Errorf("no target block named %v", block.Labels[0])
}
