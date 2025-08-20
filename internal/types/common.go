package types

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/docker-language-server/internal/tliron/glsp/protocol"
)

const BakeBuildCommandId = "dockerLspClient.bake.build"

const CodeActionDiagnosticCommandId = "server.textDocument.codeAction.diagnostics"

const TelemetryCallbackCommandId = "dockerLspServer.telemetry.callback"

func GitRepository(remoteUrl string) string {
	atIndex := strings.Index(remoteUrl, "@")
	colonIndex := strings.Index(remoteUrl, ":")
	if atIndex != -1 && atIndex < colonIndex {
		remoteUrl = remoteUrl[atIndex+1:]
		remoteUrl = strings.Replace(remoteUrl, ":/", "/", 1)
		remoteUrl = strings.Replace(remoteUrl, ":", "/", 1)
		return strings.TrimSuffix(remoteUrl, "/")
	}

	parsed, err := url.Parse(remoteUrl)
	if err != nil {
		return ""
	}

	if strings.Contains(parsed.Scheme, ".") {
		remoteUrl = strings.TrimSuffix(remoteUrl, "/")
		remoteUrl = strings.Replace(remoteUrl, ":/", "/", 1)
		return strings.Replace(remoteUrl, ":", "/", 1)
	}

	repository := fmt.Sprintf("%v%v", parsed.Host, parsed.Path)
	return strings.TrimSuffix(repository, "/")
}

// WorkspaceFolder takes in a URI and the list of workspace folders (on
// the host machine and not what is mounted inside the container) and
// returns the workspace folder that contains the given URI and the path
// relative to the workspace. If no matches can be found, "" may be
// returned for the workspace folder and for the relative path.
func WorkspaceFolder(documentURI protocol.DocumentUri, workspaceFolders []string) (folder string, absolutePath string, relativePath string) {
	parsed, err := url.Parse(documentURI)
	if err != nil {
		return "", documentURI, ""
	}

	length := 0
	candidate := ""
	for _, workspaceFolder := range workspaceFolders {
		if strings.HasPrefix(parsed.Path, workspaceFolder) {
			if length < len(workspaceFolder) {
				length = len(workspaceFolder)
				candidate = workspaceFolder
			}
		}
	}

	if strings.HasSuffix(candidate, "/") {
		return candidate, parsed.Path, parsed.Path[length:]
	}
	return candidate, parsed.Path, parsed.Path[length+1:]
}

func StripLeadingSlash(folder string) string {
	// strip leading slash from URIs with Windows drive letters
	if len(folder) > 2 && folder[0:1] == "/" && folder[2:3] == ":" {
		return folder[1:]
	}
	return folder
}

func AbsoluteFolder(documentURL *url.URL) (string, error) {
	documentPath := documentURL.Path
	if runtime.GOOS == "windows" {
		documentPath = documentURL.Path[1:]
	}
	return filepath.Abs(filepath.Dir(documentPath))
}

func Concatenate(folder, file string, wslDollarSign bool) (uri string, absoluteFilePath string) {
	if wslDollarSign {
		return "file://wsl%24" + path.Join(strings.ReplaceAll(folder, "\\", "/"), file), "\\\\wsl$" + strings.ReplaceAll(path.Join(folder, file), "/", "\\")
	}
	abs := filepath.ToSlash(filepath.Join(folder, file))
	return fmt.Sprintf("file:///%v", strings.TrimPrefix(abs, "/")), filepath.FromSlash(abs)
}

func CreateDefinitionResult(definitionLinkSupport bool, targetRange protocol.Range, originSelectionRange *protocol.Range, linkURI protocol.URI) any {
	if !definitionLinkSupport {
		return []protocol.Location{
			{
				Range: targetRange,
				URI:   linkURI,
			},
		}
	}

	return []protocol.LocationLink{
		{
			OriginSelectionRange: originSelectionRange,
			TargetRange:          targetRange,
			TargetSelectionRange: targetRange,
			TargetURI:            linkURI,
		},
	}
}

func FileStructureCompletionItems(folder string, hideFiles bool) []protocol.CompletionItem {
	if folder != "" {
		items := []protocol.CompletionItem{}
		entries, _ := os.ReadDir(folder)
		for _, entry := range entries {
			if entry.IsDir() {
				item := protocol.CompletionItem{Label: entry.Name()}
				item.Kind = CreateCompletionItemKindPointer(protocol.CompletionItemKindFolder)
				items = append(items, item)
			} else if entry.Type() == os.ModeSymlink {
				item := protocol.CompletionItem{Label: entry.Name()}
				item.Kind = CreateCompletionItemKindPointer(protocol.CompletionItemKindReference)
				items = append(items, item)
			} else if !hideFiles {
				item := protocol.CompletionItem{Label: entry.Name()}
				item.Kind = CreateCompletionItemKindPointer(protocol.CompletionItemKindFile)
				items = append(items, item)
			}
		}
		return items
	}
	return nil
}

func HubRepositoryImage(imageValue string) (repository, image, tag string) {
	// ignore images with a SHA digest
	if strings.Contains(imageValue, "@") {
		return "", "", ""
	}
	// ignore images in another repository
	slashIndex := strings.Index(imageValue, "/")
	if slashIndex != strings.LastIndex(imageValue, "/") {
		return "", "", ""
	}
	// ignore images without an explicit tag
	idx := strings.Index(imageValue, ":")
	if idx == -1 {
		return "", "", ""
	}
	split := strings.Split(imageValue[0:idx], "/")
	if len(split) == 1 {
		return "library", split[0], imageValue[idx+1:]
	}
	return split[0], split[1], imageValue[idx+1:]
}
