package buildkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/pkg/lsp/textdocument"
	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
	"github.com/docker/docker-language-server/internal/types"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/moby/buildkit/frontend/subrequests/lint"
	"github.com/moby/buildkit/solver/pb"
)

type BuildOutput struct {
	Warnings   []lint.Warning   `json:"warnings"`
	BuildError *lint.BuildError `json:"buildError,omitempty"`
}

type BuildKitDiagnosticsCollector struct {
}

// RemoveOverlappingIssues can be used to decide if diagnostics that
// overlap with what dockerfile-utils generates should be removed or
// not.
var RemoveOverlappingIssues = false

var unknownFlagRegexp *regexp.Regexp
var commandMajorityFlagRegexp *regexp.Regexp

func init() {
	unknownFlagRegexp, _ = regexp.Compile(`dockerfile parse error on line ([0-9]+): unknown flag: --([A-Za-z0-9]+)( \(did you mean ([A-Za-z]+)\?\))?`)
	commandMajorityFlagRegexp, _ = regexp.Compile(`.*command majority \((uppercase|lowercase)\)`)
}

// shouldReport examines the build error and determines if this is something
// that the language server should surface. If the string contains any of the
// following, then false will be returned.
//
// - network is unreachable
//
// - failed to resolve source metadata
func shouldReport(buildErrorMessage string) bool {
	return !strings.Contains(buildErrorMessage, "network is unreachable") &&
		!strings.Contains(buildErrorMessage, "failed to resolve source metadata")
}

func NewBuildKitDiagnosticsCollector() textdocument.DiagnosticsCollector {
	return &BuildKitDiagnosticsCollector{}
}

func createRange(lines []string, location *pb.Location) protocol.Range {
	if len(location.Ranges) == 0 {
		// there will be no ranges if the Dockerfile is empty
		return protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 0},
		}
	}

	return protocol.Range{
		Start: protocol.Position{
			Line:      uint32(location.Ranges[0].Start.Line - 1),
			Character: uint32(location.Ranges[0].Start.Character),
		},
		End: protocol.Position{
			Line:      uint32(location.Ranges[len(location.Ranges)-1].End.Line - 1),
			Character: uint32(len(lines[location.Ranges[len(location.Ranges)-1].End.Line-1])),
		},
	}
}

func encloseWithQuotes(s string) string {
	if s[0] == 34 {
		if s[len(s)-1] == 34 {
			// surrounded by quotes, reuse it
			return s
		}
		// starts with a quote
		return fmt.Sprintf(`%v"`, s)
	} else if s[len(s)-1] == 34 {
		// ends with a quote
		return fmt.Sprintf(`"%v`, s)
	}
	return fmt.Sprintf(`"%v"`, s)
}

func createResolutionEdit(instruction *parser.Node, warning lint.Warning) *types.NamedEdit {
	if instruction != nil && instruction.StartLine == int(warning.Location.Ranges[0].Start.Line) && instruction.Next != nil {
		if warning.RuleName == "MaintainerDeprecated" {
			return &types.NamedEdit{
				Title: "Convert MAINTAINER to a org.opencontainers.image.authors LABEL",
				Edit:  fmt.Sprintf(`LABEL org.opencontainers.image.authors=%v`, encloseWithQuotes(instruction.Next.Value)),
			}
		} else if warning.RuleName == "StageNameCasing" {
			stageName := instruction.Next.Next.Next.Value
			lowercase := strings.ToLower(stageName)
			words := []string{instruction.Value, instruction.Next.Value, instruction.Next.Next.Value, lowercase}
			return &types.NamedEdit{
				Title: fmt.Sprintf("Convert stage name (%v) to lowercase (%v)", stageName, lowercase),
				Edit:  strings.Join(words, " "),
			}
		} else if warning.RuleName == "RedundantTargetPlatform" {
			words := getWords(instruction)
			for i := range words {
				if words[i] == "--platform=$TARGETPLATFORM" {
					words = slices.Delete(words, i, i+1)
					break
				}
			}
			return &types.NamedEdit{
				Title: "Remove unnecessary --platform flag",
				Edit:  strings.Join(words, " "),
			}
		} else if warning.RuleName == "ConsistentInstructionCasing" {
			words := getWords(instruction)
			suggestion := strings.ToUpper(instruction.Value)
			caseSuggestion := commandMajorityFlagRegexp.FindStringSubmatch(warning.Detail)[1]
			if caseSuggestion == "lowercase" {
				suggestion = strings.ToLower(suggestion)
			}
			words[0] = suggestion
			return &types.NamedEdit{
				Title: fmt.Sprintf("Convert to %v", caseSuggestion),
				Edit:  strings.Join(words, " "),
			}
		}
	}
	return nil
}

func convertToDiagnostics(source string, doc document.DockerfileDocument, lines []string, warnings []lint.Warning) []protocol.Diagnostic {
	diagnostics := []protocol.Diagnostic{}
	for _, warning := range warnings {
		message := warning.Description
		if warning.Detail != "" {
			message = fmt.Sprintf("%v (%v)", warning.Description, warning.Detail)
		}

		diagnostic := &protocol.Diagnostic{
			Range:    createRange(lines, warning.Location),
			Code:     &protocol.IntegerOrString{Value: warning.RuleName},
			Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityWarning),
			Source:   types.CreateStringPointer(source),
			Message:  message,
		}
		if warning.URL != "" {
			diagnostic.CodeDescription = &protocol.CodeDescription{
				HRef: warning.URL,
			}
		}
		if warning.RuleName == "MaintainerDeprecated" {
			diagnostic.Tags = []protocol.DiagnosticTag{protocol.DiagnosticTagDeprecated}
		}
		instruction := doc.Instruction(protocol.Position{Line: uint32(warning.Location.Ranges[0].Start.Line) - 1})
		ignoreEdit := createIgnoreEdit(warning.RuleName)
		resolutionEdit := createResolutionEdit(instruction, warning)
		if resolutionEdit == nil {
			if ignoreEdit != nil {
				diagnostic.Data = []types.NamedEdit{*ignoreEdit}
			}
		} else {
			diagnostic.Data = []types.NamedEdit{*resolutionEdit, *ignoreEdit}
		}
		diagnostics = append(diagnostics, *diagnostic)
	}
	return diagnostics
}

func createIgnoreEdit(ruleName string) *types.NamedEdit {
	switch ruleName {
	case "ConsistentInstructionCasing":
		fallthrough
	case "CopyIgnoredFile":
		fallthrough
	case "DuplicateStageName":
		fallthrough
	case "FromAsCasing":
		fallthrough
	case "FromPlatformFlagConstDisallowed":
		fallthrough
	case "InvalidBaseImagePlatform":
		fallthrough
	case "InvalidDefaultArgInFrom":
		fallthrough
	case "InvalidDefinitionDescription":
		fallthrough
	case "JSONArgsRecommended":
		fallthrough
	case "LegacyKeyValueFormat":
		fallthrough
	case "MaintainerDeprecated":
		fallthrough
	case "MultipleInstructionsDisallowed":
		fallthrough
	case "NoEmptyContinuation":
		fallthrough
	case "RedundantTargetPlatform":
		fallthrough
	case "ReservedStageName":
		fallthrough
	case "SecretsUsedInArgOrEnv":
		fallthrough
	case "StageNameCasing":
		fallthrough
	case "UndefinedArgInFrom":
		fallthrough
	case "UndefinedVar":
		fallthrough
	case "WorkdirRelativePath":
		return &types.NamedEdit{
			Title: fmt.Sprintf("Ignore this type of error with check=skip=%v", ruleName),
			Edit:  fmt.Sprintf("# check=skip=%v\n", ruleName),
			Range: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
		}
	}
	return nil
}

func lintWithBuildKitBinary(contextPath, source string, doc document.DockerfileDocument, content string) ([]protocol.Diagnostic, error) {
	var buf bytes.Buffer
	cmd := exec.Command("docker", "buildx", "build", "--call=check,format=json", "-f-", contextPath)
	cmd.Env = append(cmd.Environ(), "DOCKER_CLI_OTEL_EXPORTER_OTLP_ENDPOINT=dockerlsp://disabled")
	cmd.Stdin = bytes.NewBuffer([]byte(content))
	cmd.Stdout = &buf
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start buildx: %w", err)
	}
	_ = cmd.Wait()

	var output BuildOutput
	err = json.Unmarshal(buf.Bytes(), &output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the buildx output: %w", err)
	}

	lines := strings.Split(content, "\n")
	diagnostics := convertToDiagnostics(source, doc, lines, output.Warnings)
	// ignore unreachable network errors
	if output.BuildError != nil && shouldReport(output.BuildError.Message) {
		diagnostic := protocol.Diagnostic{
			Range:    createRange(lines, &output.BuildError.Location),
			Severity: types.CreateDiagnosticSeverityPointer(protocol.DiagnosticSeverityError),
			Message:  output.BuildError.Message,
			Source:   types.CreateStringPointer(source),
		}
		suggestions := unknownFlagRegexp.FindStringSubmatch(output.BuildError.Message)
		if len(suggestions) == 5 {
			line, _ := strconv.Atoi(suggestions[1])
			node := doc.Instruction(protocol.Position{Line: uint32(line - 1)})
			if node != nil {
				words := getWords(node)
				if suggestions[4] == "" {
					for i := 1; i < len(words); i++ {
						if strings.HasPrefix(words[i], fmt.Sprintf("--%v", suggestions[2])) {
							words = slices.Delete(words, i, i+1)
							diagnostic.Data = []types.NamedEdit{
								{
									Title: "Remove unrecognized flag",
									Edit:  strings.Join(words, " "),
								},
							}
							break
						}
					}
				} else {
					for i := 1; i < len(words); i++ {
						if strings.HasPrefix(words[i], fmt.Sprintf("--%v", suggestions[2])) {
							words[i] = fmt.Sprintf("--%v%v", suggestions[4], words[i][2+len(suggestions[2]):])
							diagnostic.Data = []types.NamedEdit{
								{
									Title: fmt.Sprintf("Change flag name to %v", suggestions[4]),
									Edit:  strings.Join(words, " "),
								},
							}
							break
						}
					}
				}
			}
		}
		diagnostics = append(diagnostics, diagnostic)
	}
	return diagnostics, nil
}

// getWords returns the words in the line represented by the given node.
func getWords(node *parser.Node) []string {
	words := []string{}
	words = append(words, node.Value)
	words = append(words, node.Flags...)
	node = node.Next
	for node != nil {
		words = append(words, node.Value)
		node = node.Next
	}
	return words
}

func parse(contextPath, source string, doc document.DockerfileDocument, content string) ([]protocol.Diagnostic, error) {
	escapedPath, err := url.QueryUnescape(contextPath)
	if err != nil {
		escapedPath = contextPath
	}
	return lintWithBuildKitBinary(escapedPath, source, doc, content)
}

func shouldIgnore(diagnostic protocol.Diagnostic) bool {
	if *diagnostic.Severity == protocol.DiagnosticSeverityError {
		return true
	}
	if diagnostic.Code != nil {
		if value, ok := diagnostic.Code.Value.(string); ok {
			switch value {
			case "ConsistentInstructionCasing":
				return true
			case "DuplicateStageName":
				return true
			case "MaintainerDeprecated":
				return true
			case "MultipleInstructionsDisallowed":
				return true
			case "NoEmptyContinuation":
				return true
			case "WorkdirRelativePath":
				return true
			}
		}
	}
	return false
}

func (c *BuildKitDiagnosticsCollector) CollectDiagnostics(source, workspaceFolder string, doc document.Document, text string) []protocol.Diagnostic {
	diagnostics, _ := parse(workspaceFolder, source, doc.(document.DockerfileDocument), text)
	if RemoveOverlappingIssues {
		filtered := []protocol.Diagnostic{}
		for i := range diagnostics {
			if !shouldIgnore(diagnostics[i]) {
				filtered = append(filtered, diagnostics[i])
			}
		}
		return filtered
	}
	return diagnostics
}

func (c *BuildKitDiagnosticsCollector) SupportsLanguageIdentifier(languageIdentifier protocol.LanguageIdentifier) bool {
	return languageIdentifier == protocol.DockerfileLanguage
}
