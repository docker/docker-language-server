package server

import (
	"context"
	"io"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/bugsnag/bugsnag-go"
	"github.com/docker/docker-language-server/internal/bake/hcl"
	"github.com/docker/docker-language-server/internal/compose"
	"github.com/docker/docker-language-server/internal/configuration"
	"github.com/docker/docker-language-server/internal/hub"
	"github.com/docker/docker-language-server/internal/pkg/buildkit"
	"github.com/docker/docker-language-server/internal/pkg/cli/metadata"
	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/pkg/lsp/textdocument"
	"github.com/docker/docker-language-server/internal/scout"
	"github.com/docker/docker-language-server/internal/telemetry"
	"github.com/docker/docker-language-server/internal/tliron/glsp"
	"github.com/docker/docker-language-server/internal/types"
	"github.com/moby/buildkit/identity"

	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/tliron/glsp/server"

	"github.com/sourcegraph/jsonrpc2"
)

const registerCapabilityDelay = 2 * time.Second

type Server struct {
	gs   *server.Server
	docs *document.Manager

	hubService   *hub.Service
	scoutService scout.Service

	// sessionTelemetryProperties contains a map of values that should
	// be included in every telemetry event.
	//
	// server_version: the version of the server
	// lsp_client_info_name: the value of the clientInfo.name parameter in the initialize request
	// lsp_client_info_version: the value of the clientInfo.version parameter in the initialize request
	// client_name: the name of the client that may be set by the language client
	// client_session: the session client that may be set by the language client
	sessionTelemetryProperties map[string]string

	client      *LanguageClient
	telemetry   telemetry.TelemetryClient
	initialized bool

	workspaceFolders []string

	diagnosticsCollectors []textdocument.DiagnosticsCollector

	capabilities *ExperimentalCapabilities

	definitionLinkSupport bool

	showDocumentSupport bool

	showMessageRequestSupport bool

	gitRemotes map[string]string

	// analyzedFiles maps a Git remote with the array of analyzed files
	// within that Git folder.
	analyzedFiles map[string]map[string]bool

	composeSupport    bool
	composeCompletion bool

	mutex sync.RWMutex
}

func NewServer(docManager *document.Manager) *Server {
	hubService := hub.NewService()
	scoutService := scout.NewService()
	handler := protocol.Handler{}
	sessionTelemetryProperties := make(map[string]string)
	sessionTelemetryProperties["server_session"] = identity.NewID()
	sessionTelemetryProperties["server_version"] = metadata.Version
	s := &Server{
		docs:                       docManager,
		definitionLinkSupport:      false,
		analyzedFiles:              make(map[string]map[string]bool),
		gitRemotes:                 make(map[string]string),
		gs:                         server.NewServer(&handler, "", false),
		initialized:                false,
		telemetry:                  telemetry.NewClient(),
		hubService:                 &hubService,
		scoutService:               scoutService,
		sessionTelemetryProperties: sessionTelemetryProperties,
		composeSupport:             true,
		composeCompletion:          true,
		diagnosticsCollectors: []textdocument.DiagnosticsCollector{
			buildkit.NewBuildKitDiagnosticsCollector(),
			scoutService,
			compose.NewComposeDiagnosticsCollector(),
			hcl.NewBakeHCLDiagnosticsCollector(docManager, scoutService),
		},
	}

	handler.Initialize = s.Initialize
	handler.Initialized = s.Initialized
	handler.Shutdown = s.shutdown
	handler.SetTrace = s.setTrace

	handler.TextDocumentCodeAction = s.TextDocumentCodeAction
	handler.TextDocumentCodeLens = s.TextDocumentCodeLens
	handler.TextDocumentCompletion = s.TextDocumentCompletion
	handler.TextDocumentDefinition = s.TextDocumentDefinition
	handler.TextDocumentFormatting = s.TextDocumentFormatting
	handler.TextDocumentDocumentHighlight = s.TextDocumentDocumentHighlight
	handler.TextDocumentDocumentLink = s.TextDocumentDocumentLink
	handler.TextDocumentDocumentSymbol = s.TextDocumentDocumentSymbol
	handler.TextDocumentHover = s.TextDocumentHover
	handler.TextDocumentInlayHint = s.TextDocumentInlayHint
	handler.TextDocumentInlineCompletion = s.TextDocumentInlineCompletion
	handler.TextDocumentPrepareRename = s.TextDocumentPrepareRename
	handler.TextDocumentRename = s.TextDocumentRename
	handler.TextDocumentSemanticTokensFull = s.TextDocumentSemanticTokensFull

	handler.TextDocumentDidOpen = s.TextDocumentDidOpen
	handler.TextDocumentDidChange = s.TextDocumentDidChange
	handler.TextDocumentDidClose = s.TextDocumentDidClose

	handler.WorkspaceDidChangeConfiguration = s.WorkspaceDidChangeConfiguration
	handler.WorkspaceExecuteCommand = s.WorkspaceExecuteCommand

	handler.Recover = func(method string, recovered interface{}) error {
		if s.handleRecovered(method, recovered) {
			return &jsonrpc2.Error{Code: -32803, Message: "Internal server error"}
		}
		return nil
	}
	return s
}

func (s *Server) RunStdio() error {
	return s.gs.RunStdio()
}

func (s *Server) RunTCP(address string) error {
	return s.gs.RunTCP(address)
}

// ServeStream is only intended to be used for the tests.
func (s *Server) ServeStream(stream io.ReadWriteCloser) {
	s.gs.ServeStream(stream, nil)
}

func (s *Server) Initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	s.initialized = true
	return nil
}

func (s *Server) shutdown(ctx *glsp.Context) error {
	_, _ = s.telemetry.Publish(context.Background())
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

func (s *Server) setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}

func (s *Server) WorkspaceFolders() []string {
	return s.workspaceFolders
}

func (s *Server) recomputeDiagnostics() {
	for _, uri := range s.docs.Keys() {
		doc := s.docs.Get(context.Background(), uri)
		if doc != nil {
			s.computeDiagnostics(context.Background(), string(uri))
		}
	}
}

func (s *Server) StartBackgrondProcesses(ctx context.Context) {
	s.publishTelemetry(ctx)
}

func (s *Server) publishTelemetry(ctx context.Context) {
	go func() {
		defer s.handlePanic("publishTelemetry")

		for {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(time.Second * 60)
				_, _ = s.telemetry.Publish(ctx)
			}
		}
	}()
}

func (s *Server) handleRecovered(method string, recovered interface{}) bool {
	if recovered != nil {
		debug.PrintStack()
		properties := map[string]any{
			"type":        telemetry.ServerHeartbeatTypePanic,
			"method":      method,
			"stack_trace": string(debug.Stack()),
		}
		if err, ok := recovered.(error); ok {
			properties["recover"] = err.Error()
			metadata := bugsnag.MetaData{}
			metadata.Add("session", "method", method)
			for property, value := range s.sessionTelemetryProperties {
				metadata.Add("session", property, value)
			}
			metadata.Add("session", "arch", runtime.GOARCH)
			metadata.Add("session", "os", runtime.GOOS)
			_ = bugsnag.Notify(err, metadata)
		}
		s.Enqueue(telemetry.EventServerHeartbeat, properties)
		return true
	}
	return false
}

func (s *Server) handlePanic(method string) bool {
	if r := recover(); r != nil {
		return s.handleRecovered(method, r)
	}
	return false
}

func (s *Server) FetchUnscopedConfiguration() {
	section := "docker.lsp"
	items := []protocol.ConfigurationItem{
		{Section: &section},
	}
	fetchedConfigurations := []configuration.Configuration{}
	s.client.WorkspaceConfiguration(context.Background(), protocol.ConfigurationParams{Items: items}, &fetchedConfigurations)
	if len(fetchedConfigurations) == 1 {
		s.telemetry.UpdateTelemetrySetting(string(fetchedConfigurations[0].Telemetry))
	}
}

func (s *Server) FetchConfigurations(scopes []protocol.DocumentUri) {
	section := "docker.lsp"
	items := []protocol.ConfigurationItem{}
	for _, scope := range scopes {
		items = append(items, protocol.ConfigurationItem{
			ScopeURI: &scope,
			Section:  &section,
		})
	}
	fetchedConfigurations := []configuration.Configuration{}
	s.client.WorkspaceConfiguration(context.Background(), protocol.ConfigurationParams{Items: items}, &fetchedConfigurations)
	if len(scopes) == len(fetchedConfigurations) {
		for i := range scopes {
			configuration.Store(scopes[i], fetchedConfigurations[i])
		}
	}
}

func (s *Server) updateTelemetrySetting(value string) {
	s.telemetry.UpdateTelemetrySetting(value)
}

// registerFormattingCapability sends a client/registerCapability
// request to the client to specifically support the
// textDocument/formatting requests only for dockerbake files. If we
// register it globally for all languages, VS Code will show a warning
// to the user to inform them that multiple formatters have been
// registered. Since we do not actually support formatting Dockerfiles,
// giving the user the impression that we do support it would be
// confusing so it is better to explicitly register formatting support
// rather than doing it globally.
func (s *Server) registerFormattingCapability() {
	dockerbakeLanguage := string(protocol.DockerBakeLanguage)
	dockercomposeLanguage := string(protocol.DockerComposeLanguage)
	capabilities := []protocol.Registration{
		{
			ID:     "docker.lsp.dockerbake.textDocument.formatting",
			Method: "textDocument/formatting",
			RegisterOptions: protocol.TextDocumentRegistrationOptions{
				DocumentSelector: &protocol.DocumentSelector{protocol.DocumentFilter{Language: &dockerbakeLanguage}},
			},
		},
	}
	if s.composeSupport {
		capabilities = append(capabilities, protocol.Registration{
			ID:     "docker.lsp.dockercompose.textDocument.formatting",
			Method: "textDocument/formatting",
			RegisterOptions: protocol.TextDocumentRegistrationOptions{
				DocumentSelector: &protocol.DocumentSelector{protocol.DocumentFilter{Language: &dockercomposeLanguage}},
			},
		})
	}
	s.registerCapability(capabilities)
}

func (s *Server) registerRenameCapability() {
	dockercomposeLanguage := string(protocol.DockerComposeLanguage)
	dockercomposeDocumentSelctor := protocol.DocumentSelector{protocol.DocumentFilter{Language: &dockercomposeLanguage}}
	s.registerCapability(
		[]protocol.Registration{
			{
				ID:     "docker.lsp.dockercompose.textDocument.rename",
				Method: "textDocument/rename",
				RegisterOptions: protocol.RenameRegistrationOptions{
					TextDocumentRegistrationOptions: protocol.TextDocumentRegistrationOptions{
						DocumentSelector: &dockercomposeDocumentSelctor,
					},
					RenameOptions: protocol.RenameOptions{
						PrepareProvider: types.CreateBoolPointer(true),
					},
				},
			},
		},
	)
}

func (s *Server) registerCapability(registrations []protocol.Registration) {
	go func() {
		defer s.handlePanic("registerCapability")

		time.Sleep(registerCapabilityDelay)
		s.client.RegisterCapability(context.Background(), protocol.RegistrationParams{
			Registrations: registrations,
		})
	}()
}

func (s *Server) Enqueue(event string, properties map[string]any) {
	for property, value := range s.sessionTelemetryProperties {
		properties[property] = value
	}
	s.telemetry.Enqueue(event, properties)
}
