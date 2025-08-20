package types

import "github.com/docker/docker-language-server/internal/tliron/glsp/protocol"

type NamedEdit struct {
	Title string          `json:"title"`
	Edit  string          `json:"edit"`
	Range *protocol.Range `json:"range,omitempty"`
}

func CreateDiagnosticSeverityPointer(ds protocol.DiagnosticSeverity) *protocol.DiagnosticSeverity {
	return &ds
}

func CreateAnyPointer(a any) *any {
	return &a
}

func CreateBoolPointer(b bool) *bool {
	return &b
}

func CreateInt32Pointer(i int32) *int32 {
	return &i
}

func CreateStringPointer(s string) *string {
	return &s
}

func CreateDocumentHighlightKindPointer(k protocol.DocumentHighlightKind) *protocol.DocumentHighlightKind {
	return &k
}

func CreateCompletionItemKindPointer(k protocol.CompletionItemKind) *protocol.CompletionItemKind {
	return &k
}

func CreateInsertTextFormatPointer(f protocol.InsertTextFormat) *protocol.InsertTextFormat {
	return &f
}

func CreateInsertTextModePointer(m protocol.InsertTextMode) *protocol.InsertTextMode {
	return &m
}
