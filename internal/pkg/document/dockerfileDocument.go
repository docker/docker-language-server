package document

import (
	"bytes"
	"strings"
	"sync"

	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"go.lsp.dev/uri"
)

type DockerfileDocument interface {
	Document
	Instruction(p protocol.Position) *parser.Node
	Nodes() []*parser.Node
}

func NewDockerfileDocument(u uri.URI, version int32, input []byte) DockerfileDocument {
	doc := &dockerfileDocument{
		document: document{
			uri:        u,
			identifier: protocol.DockerfileLanguage,
			version:    version,
			input:      input,
		},
	}
	doc.copyFn = doc.copy
	doc.parseFn = doc.parse
	doc.parseFn(true)
	return doc
}

type dockerfileDocument struct {
	document
	mutex  sync.Mutex
	result *parser.Result
}

func (d *dockerfileDocument) Nodes() []*parser.Node {
	d.parseFn(false)
	if d.result == nil {
		return nil
	}

	return d.result.AST.Children
}

func (d *dockerfileDocument) parse(force bool) bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.result == nil || force {
		result, _ := parser.Parse(bytes.NewReader(d.input))
		if result == nil {
			// we have no AST information to compare to so we assume
			// something has changed
			d.result = nil
			return true
		} else if d.result == nil {
			d.result = result
			return true
		}

		children := d.result.AST.Children
		newChildren := result.AST.Children
		d.result = result

		if len(children) != len(newChildren) {
			return true
		}

		for i := range children {
			node := children[i]
			newNode := newChildren[i]
			if compareNodes(node, newNode) {
				return true
			}
		}

		if children[0].StartLine != 1 {
			lines := strings.Split(string(d.input), "\n")
			for i := range children[0].StartLine {
				if strings.Contains(lines[i], "check=") {
					return true
				}
			}
		}
	}
	return false
}

func compareNodes(n1, n2 *parser.Node) bool {
	if len(n1.Flags) != len(n2.Flags) {
		return true
	}

	for i := range n1.Flags {
		if n1.Flags[i] != n2.Flags[i] {
			return true
		}
	}

	for {
		if n1 == nil {
			return n2 != nil
		} else if n2 == nil {
			return true
		}

		if n1.StartLine != n2.StartLine || n1.EndLine != n2.EndLine {
			return true
		}

		if n1.Value != n2.Value {
			return true
		}

		n1 = n1.Next
		n2 = n2.Next
	}
}

func (d *dockerfileDocument) Instruction(p protocol.Position) *parser.Node {
	d.parseFn(false)
	if d.result != nil {
		for _, instruction := range d.result.AST.Children {
			if instruction.StartLine <= int(p.Line+1) && int(p.Line+1) <= instruction.EndLine {
				return instruction
			}
		}
	}
	return nil
}
