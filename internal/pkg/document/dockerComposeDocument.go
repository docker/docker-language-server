package document

import (
	"sync"

	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"go.lsp.dev/uri"
)

type ComposeDocument interface {
	Document
	File() *ast.File
	ParsingError() error
}

type composeDocument struct {
	document
	mutex        sync.Mutex
	file         *ast.File
	parsingError error
}

func NewComposeDocument(u uri.URI, version int32, input []byte) ComposeDocument {
	doc := &composeDocument{
		document: document{
			uri:        u,
			identifier: protocol.DockerComposeLanguage,
			version:    version,
			input:      input,
		},
	}
	doc.document.copyFn = doc.copy
	doc.document.parseFn = doc.parse
	doc.document.parseFn(true)
	return doc
}

func (d *composeDocument) parse(_ bool) bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.file, d.parsingError = parser.ParseBytes(d.input, parser.ParseComments)
	return true
}

func (d *composeDocument) copy() Document {
	return NewComposeDocument(d.uri, d.version, d.input)
}

func (d *composeDocument) File() *ast.File {
	return d.file
}

func (d *composeDocument) ParsingError() error {
	return d.parsingError
}
