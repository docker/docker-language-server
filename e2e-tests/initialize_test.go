package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker-language-server/internal/bake/hcl"
	"github.com/docker/docker-language-server/internal/pkg/cli/metadata"
	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/pkg/server"
	"github.com/docker/docker-language-server/internal/tliron/glsp"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("DOCKER_LANGUAGE_SERVER_TELEMETRY", "false")
}

type TestStream struct {
	incoming *bytes.Buffer
	outgoing *bytes.Buffer
	closed   bool
}

func (ts *TestStream) Read(b []byte) (int, error) {
	for {
		if ts.closed {
			return 0, io.EOF
		}

		r, err := ts.incoming.Read(b)
		if r > 0 {
			return r, err
		} else if err != io.EOF {
			return r, err
		}
		time.Sleep(1 * time.Millisecond)
	}
}

func (ts *TestStream) Write(n []byte) (int, error) {
	return ts.outgoing.Write(n)
}

func (ts *TestStream) Close() error {
	ts.closed = true
	return nil
}

func startServer() *server.Server {
	docManager := document.NewDocumentManager()
	s := server.NewServer(docManager)
	return s
}

func createDidOpenTextDocumentParams(homedir, testName, text string, languageID protocol.LanguageIdentifier) protocol.DidOpenTextDocumentParams {
	return protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        protocol.URI(fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(homedir, testName)), "/"))),
			Text:       text,
			LanguageID: languageID,
			Version:    1,
		},
	}
}

func createDidChangeTextDocumentParams(homedir, testName, text string, version int32) protocol.DidChangeTextDocumentParams {
	return protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{
				URI: protocol.URI(fmt.Sprintf("file:///%v", strings.TrimPrefix(filepath.ToSlash(filepath.Join(homedir, testName)), "/"))),
			},
			Version: version,
		},
		ContentChanges: []any{
			protocol.TextDocumentContentChangeEvent{
				Text: text,
			},
		},
	}
}

func createGuaranteedInitializeResult() protocol.InitializeResult {
	syncKind := protocol.TextDocumentSyncKindFull
	return protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			CodeActionProvider: protocol.CodeActionOptions{},
			CompletionProvider: &protocol.CompletionOptions{
				TriggerCharacters: []string{"/"},
			},
			DefinitionProvider:        protocol.DefinitionOptions{},
			DocumentHighlightProvider: &protocol.DocumentHighlightOptions{},
			DocumentLinkProvider:      &protocol.DocumentLinkOptions{},
			DocumentSymbolProvider:    protocol.DocumentSymbolOptions{},
			ExecuteCommandProvider: &protocol.ExecuteCommandOptions{
				Commands: []string{types.TelemetryCallbackCommandId},
			},
			HoverProvider:            protocol.HoverOptions{},
			InlayHintProvider:        protocol.InlayHintOptions{},
			InlineCompletionProvider: protocol.InlineCompletionOptions{},
			SemanticTokensProvider: protocol.SemanticTokensOptions{
				Legend: protocol.SemanticTokensLegend{
					TokenModifiers: []string{},
					TokenTypes:     hcl.SemanticTokenTypes,
				},
				Full:  true,
				Range: false,
			},
			TextDocumentSync: protocol.TextDocumentSyncOptions{
				OpenClose: &protocol.True,
				Change:    &syncKind,
			},
		},
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    "docker-language-server",
			Version: &metadata.Version,
		},
	}
}

func initialize(t *testing.T, conn *jsonrpc2.Conn, initializeParams protocol.InitializeParams) {
	expected := createGuaranteedInitializeResult()
	expected.Capabilities.DocumentFormattingProvider = protocol.DocumentFormattingOptions{}
	expected.Capabilities.RenameProvider = protocol.RenameOptions{PrepareProvider: types.CreateBoolPointer(true)}
	initializeCheck(t, conn, initializeParams, expected)
}

func initializeCheck(t *testing.T, conn *jsonrpc2.Conn, initializeParams protocol.InitializeParams, expected protocol.InitializeResult) {
	if options, ok := initializeParams.InitializationOptions.(map[string]any); ok {
		options["telemetry"] = "off"
	} else {
		initializeParams.InitializationOptions = map[string]string{"telemetry": "off"}
	}
	var initializeResult *protocol.InitializeResult
	err := conn.Call(context.Background(), protocol.MethodInitialize, initializeParams, &initializeResult)
	require.NoError(t, err)
	requireJsonEqual(t, expected, initializeResult)
}

func TestInitializeFunctionIsolatedCall(t *testing.T) {
	testCases := []struct {
		name             string
		params           protocol.InitializeParams
		workspaceFolders []string
		err              error
	}{
		{
			name: "one workspace folder",
			params: protocol.InitializeParams{
				WorkspaceFolders: []protocol.WorkspaceFolder{{Name: "abc", URI: "uri:///a/b/c"}},
			},
			workspaceFolders: []string{"/a/b/c"},
		},
		{
			name: "two workspace folders",
			params: protocol.InitializeParams{
				WorkspaceFolders: []protocol.WorkspaceFolder{
					{Name: "abc", URI: "uri:///a/b/c"},
					{Name: "def", URI: "uri:///d/e/f"},
				},
			},
			workspaceFolders: []string{"/a/b/c", "/d/e/f"},
		},
		{
			name:             "rootUri",
			params:           protocol.InitializeParams{RootURI: types.CreateStringPointer("uri:///a/b/c")},
			workspaceFolders: []string{"/a/b/c"},
		},
		{
			name:             "rootPath",
			params:           protocol.InitializeParams{RootPath: types.CreateStringPointer("/a/b/c")},
			workspaceFolders: []string{"/a/b/c"},
		},
		{
			name:             "wsl$ rootUri",
			params:           protocol.InitializeParams{RootURI: types.CreateStringPointer("file://wsl%24/docker-desktop/tmp")},
			workspaceFolders: []string{"\\\\wsl$\\docker-desktop\\tmp"},
		},
		{
			name: "wsl$ workspace folder",
			params: protocol.InitializeParams{
				WorkspaceFolders: []protocol.WorkspaceFolder{{Name: "tmp", URI: "file://wsl%24/docker-desktop/tmp"}},
			},
			workspaceFolders: []string{"\\\\wsl$\\docker-desktop\\tmp"},
		},
		{
			name:             "invalid rootUri",
			params:           protocol.InitializeParams{RootURI: types.CreateStringPointer(":/no/scheme")},
			workspaceFolders: nil,
			err:              &jsonrpc2.Error{Code: -32602, Message: "invalid rootUri specified in the initialize request (:/no/scheme): parse \":/no/scheme\": missing protocol scheme"},
		},
		{
			name:             "unrecognized dollar sign",
			params:           protocol.InitializeParams{RootURI: types.CreateStringPointer("file://abc%24/hello/tmp")},
			workspaceFolders: nil,
			err:              &jsonrpc2.Error{Code: -32602, Message: "invalid rootUri specified in the initialize request (file://abc%24/hello/tmp): parse \"file://abc%24/hello/tmp\": invalid URL escape \"%24\""},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := testInitialize(t, tc.params, tc.err)
			require.Equal(t, tc.workspaceFolders, s.WorkspaceFolders())
		})
	}
}

func testInitialize(t *testing.T, initializeParams protocol.InitializeParams, expectedErr error) *server.Server {
	s := startServer()
	_, err := s.Initialize(&glsp.Context{}, &initializeParams)
	require.Equal(t, expectedErr, err)
	return s
}

func TestInitialize(t *testing.T) {
	testCases := []struct {
		name   string
		params protocol.InitializeParams
		result func() protocol.InitializeResult
	}{
		{
			name: "dynamic formatting support not declared",
			params: protocol.InitializeParams{
				Capabilities: protocol.ClientCapabilities{
					TextDocument: &protocol.TextDocumentClientCapabilities{
						Formatting: &protocol.DocumentFormattingClientCapabilities{},
					},
				},
			},
			result: func() protocol.InitializeResult {
				expected := createGuaranteedInitializeResult()
				expected.Capabilities.DocumentFormattingProvider = protocol.DocumentFormattingOptions{}
				expected.Capabilities.RenameProvider = protocol.RenameOptions{PrepareProvider: types.CreateBoolPointer(true)}
				return expected
			},
		},
		{
			name: "dynamic formatting support false",
			params: protocol.InitializeParams{
				Capabilities: protocol.ClientCapabilities{
					TextDocument: &protocol.TextDocumentClientCapabilities{
						Formatting: &protocol.DocumentFormattingClientCapabilities{DynamicRegistration: types.CreateBoolPointer(false)},
					},
				},
			},
			result: func() protocol.InitializeResult {
				expected := createGuaranteedInitializeResult()
				expected.Capabilities.DocumentFormattingProvider = protocol.DocumentFormattingOptions{}
				expected.Capabilities.RenameProvider = protocol.RenameOptions{PrepareProvider: types.CreateBoolPointer(true)}
				return expected
			},
		},
		{
			name: "dynamic rename support not declared",
			params: protocol.InitializeParams{
				Capabilities: protocol.ClientCapabilities{
					TextDocument: &protocol.TextDocumentClientCapabilities{
						Rename: &protocol.RenameClientCapabilities{},
					},
				},
			},
			result: func() protocol.InitializeResult {
				expected := createGuaranteedInitializeResult()
				expected.Capabilities.DocumentFormattingProvider = protocol.DocumentFormattingOptions{}
				expected.Capabilities.RenameProvider = protocol.RenameOptions{PrepareProvider: types.CreateBoolPointer(true)}
				return expected
			},
		},
		{
			name: "dynamic rename support false",
			params: protocol.InitializeParams{
				Capabilities: protocol.ClientCapabilities{
					TextDocument: &protocol.TextDocumentClientCapabilities{
						Rename: &protocol.RenameClientCapabilities{DynamicRegistration: types.CreateBoolPointer(false)},
					},
				},
			},
			result: func() protocol.InitializeResult {
				expected := createGuaranteedInitializeResult()
				expected.Capabilities.DocumentFormattingProvider = protocol.DocumentFormattingOptions{}
				expected.Capabilities.RenameProvider = protocol.RenameOptions{PrepareProvider: types.CreateBoolPointer(true)}
				return expected
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := startServer()

			client := bytes.NewBuffer(make([]byte, 0, 1024))
			server := bytes.NewBuffer(make([]byte, 0, 1024))
			serverStream := &TestStream{incoming: server, outgoing: client, closed: false}
			defer serverStream.Close()
			go s.ServeStream(serverStream)

			clientStream := jsonrpc2.NewBufferedStream(&TestStream{incoming: client, outgoing: server, closed: false}, jsonrpc2.VSCodeObjectCodec{})
			defer clientStream.Close()
			conn := jsonrpc2.NewConn(context.Background(), clientStream, &ConfigurationHandler{t: t})
			initializeCheck(t, conn, tc.params, tc.result())
		})
	}
}

type registerCapabilityHandler struct {
	t                  *testing.T
	responseChannel    chan error
	registrationParams *protocol.RegistrationParams
}

func (h *registerCapabilityHandler) Handle(_ context.Context, conn *jsonrpc2.Conn, request *jsonrpc2.Request) {
	switch request.Method {
	case protocol.ServerClientRegisterCapability:
		if !request.Notif && request.Params != nil {
			h.registrationParams = &protocol.RegistrationParams{}
			require.NoError(h.t, json.Unmarshal(*request.Params, &h.registrationParams))
			h.responseChannel <- nil
		} else {
			h.responseChannel <- errors.New("malformed client/registerCapability JSON-RPC call")
		}
	}
}

func TestRegisterCapability(t *testing.T) {
	testCases := []struct {
		name               string
		params             protocol.InitializeParams
		initializeResult   func() protocol.InitializeResult
		registrationParams *protocol.RegistrationParams
	}{
		{
			name: "dynamic formatting supported",
			params: protocol.InitializeParams{
				Capabilities: protocol.ClientCapabilities{
					TextDocument: &protocol.TextDocumentClientCapabilities{
						Formatting: &protocol.DocumentFormattingClientCapabilities{
							DynamicRegistration: types.CreateBoolPointer(true),
						},
					},
				},
			},
			initializeResult: func() protocol.InitializeResult {
				expected := createGuaranteedInitializeResult()
				expected.Capabilities.RenameProvider = protocol.RenameOptions{PrepareProvider: types.CreateBoolPointer(true)}
				return expected
			},
			registrationParams: &protocol.RegistrationParams{
				Registrations: []protocol.Registration{
					{
						ID:     "docker.lsp.dockerbake.textDocument.formatting",
						Method: "textDocument/formatting",
						RegisterOptions: map[string]any{
							"documentSelector": []any{map[string]any{"language": "dockerbake"}},
						},
					},
					{
						ID:     "docker.lsp.dockercompose.textDocument.formatting",
						Method: "textDocument/formatting",
						RegisterOptions: map[string]any{
							"documentSelector": []any{map[string]any{"language": "dockercompose"}},
						},
					},
				},
			},
		},
		{
			name: "dynamic formatting supported but composeSupport is false",
			params: protocol.InitializeParams{
				InitializationOptions: map[string]any{
					"dockercomposeExperimental": map[string]bool{"composeSupport": false},
				},
				Capabilities: protocol.ClientCapabilities{
					TextDocument: &protocol.TextDocumentClientCapabilities{
						Formatting: &protocol.DocumentFormattingClientCapabilities{
							DynamicRegistration: types.CreateBoolPointer(true),
						},
					},
				},
			},
			initializeResult: func() protocol.InitializeResult {
				expected := createGuaranteedInitializeResult()
				expected.Capabilities.RenameProvider = protocol.RenameOptions{PrepareProvider: types.CreateBoolPointer(true)}
				return expected
			},
			registrationParams: &protocol.RegistrationParams{
				Registrations: []protocol.Registration{
					{
						ID:     "docker.lsp.dockerbake.textDocument.formatting",
						Method: "textDocument/formatting",
						RegisterOptions: map[string]any{
							"documentSelector": []any{map[string]any{"language": "dockerbake"}},
						},
					},
				},
			},
		},
		{
			name: "dynamic rename supported",
			params: protocol.InitializeParams{
				Capabilities: protocol.ClientCapabilities{
					TextDocument: &protocol.TextDocumentClientCapabilities{
						Rename: &protocol.RenameClientCapabilities{
							DynamicRegistration: types.CreateBoolPointer(true),
						},
					},
				},
			},
			initializeResult: func() protocol.InitializeResult {
				expected := createGuaranteedInitializeResult()
				expected.Capabilities.DocumentFormattingProvider = protocol.DocumentFormattingOptions{}
				return expected
			},
			registrationParams: &protocol.RegistrationParams{
				Registrations: []protocol.Registration{
					{
						ID:     "docker.lsp.dockercompose.textDocument.rename",
						Method: "textDocument/rename",
						RegisterOptions: map[string]any{
							"documentSelector": []any{map[string]any{"language": "dockercompose"}},
							"prepareProvider":  true,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := startServer()

			client := bytes.NewBuffer(make([]byte, 0, 1024))
			server := bytes.NewBuffer(make([]byte, 0, 1024))
			serverStream := &TestStream{incoming: server, outgoing: client, closed: false}
			defer serverStream.Close()
			go s.ServeStream(serverStream)

			clientStream := jsonrpc2.NewBufferedStream(&TestStream{incoming: client, outgoing: server, closed: false}, jsonrpc2.VSCodeObjectCodec{})
			defer clientStream.Close()
			h := registerCapabilityHandler{t: t, responseChannel: make(chan error)}
			conn := jsonrpc2.NewConn(context.Background(), clientStream, &h)
			initializeCheck(t, conn, tc.params, tc.initializeResult())
			require.NoError(t, <-h.responseChannel)
			require.Equal(t, tc.registrationParams, h.registrationParams)
		})
	}
}
