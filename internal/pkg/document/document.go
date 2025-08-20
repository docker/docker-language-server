package document

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
	"go.lsp.dev/uri"
)

type DocumentPath struct {
	Folder            string
	FileName          string
	WSLDollarSignHost bool
}

type Document interface {
	URI() uri.URI
	DocumentPath() (DocumentPath, error)
	Copy() Document
	Input() []byte
	Version() int32
	Update(version int32, input []byte) bool
	Close()
	LanguageIdentifier() protocol.LanguageIdentifier
}

type NewDocumentFunc func(mgr *Manager, u uri.URI, identifier protocol.LanguageIdentifier, version int32, input []byte) Document

func NewDocument(mgr *Manager, u uri.URI, identifier protocol.LanguageIdentifier, version int32, input []byte) Document {
	switch identifier {
	case protocol.DockerBakeLanguage:
		return NewBakeHCLDocument(u, version, input)
	case protocol.DockerComposeLanguage:
		return NewComposeDocument(mgr, u, version, input)
	}
	return NewDockerfileDocument(u, version, input)
}

// DirectoryForPrefix returns the parent directory to be used given the
// document's path and the prefix string that has been inserted into the
// document thus far.
//
// prefixRequired is true if prefix can just be a name without any
// slashes or backslashes.
func DirectoryForPrefix(documentPath DocumentPath, prefix, defaultValue string, prefixRequired bool) string {
	idx := strings.LastIndex(prefix, "/")
	if idx == -1 {
		if prefixRequired {
			if len(prefix) > 2 && unicode.IsLetter(rune(prefix[0])) && prefix[1] == ':' {
				backslashIdx := strings.LastIndex(prefix, "\\")
				if backslashIdx != -1 {
					return prefix[0 : backslashIdx+1]
				}
			}
			return defaultValue
		}
		return documentPath.Folder
	} else if prefix[0] == '/' {
		return prefix[0 : idx+1]
	}
	_, folder := types.Concatenate(documentPath.Folder, prefix[0:idx], documentPath.WSLDollarSignHost)
	return folder
}

type document struct {
	uri        uri.URI
	identifier protocol.LanguageIdentifier
	version    int32
	// input is the file as it exists in the editor buffer.
	input   []byte
	parseFn func(force bool) bool
	copyFn  func() Document
}

var _ Document = &document{}

func (d *document) Update(version int32, input []byte) bool {
	d.version = version
	d.input = input
	return d.parseFn(true)
}

func (d *document) Version() int32 {
	return d.version
}

func (d *document) Input() []byte {
	return d.input
}

func (d *document) URI() uri.URI {
	return d.uri
}

func (d *document) DocumentPath() (DocumentPath, error) {
	uriString := string(d.uri)
	url, err := url.Parse(uriString)
	if err != nil {
		if strings.HasPrefix(uriString, "file://wsl%24/") {
			path := uriString[len("file://wsl%24"):]
			idx := strings.LastIndex(path, "/")
			return DocumentPath{Folder: path[0:idx], FileName: path[idx+1:], WSLDollarSignHost: true}, nil
		}
		return DocumentPath{}, fmt.Errorf("invalid URI: %v", uriString)
	}
	folder, err := types.AbsoluteFolder(url)
	idx := strings.LastIndex(uriString, "/")
	return DocumentPath{Folder: folder, FileName: uriString[idx+1:]}, err
}

func (d *document) LanguageIdentifier() protocol.LanguageIdentifier {
	return d.identifier
}

func (d *document) Close() {
}

// Copy creates a shallow copy of the Document.
//
// The Contents byte slice is returned as-is.
// A shallow copy of the Tree is made, as Tree-sitter trees are not thread-safe.
func (d *document) Copy() Document {
	return d.copyFn()
}
