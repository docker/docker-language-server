package cli

import (
	"bytes"
	"context"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/docker/docker-language-server/internal/pkg/document"
	"github.com/docker/docker-language-server/internal/pkg/server"
)

type startCmd struct {
	*cobra.Command
	address string
	stdio   bool
}

var exampleTemplate = template.Must(template.New("example").Parse(`
# Launch in stdio mode with extra logging
{{.BaseCommandName}} start --stdio --verbose

# Listen on all interfaces on port 8765
{{.BaseCommandName}} start --address=":8765"`))

type exampleTemplateParams struct {
	BaseCommandName string
}

// creates a new startCmd
// params:
//
//	commandName: what to call the base command in examples (e.g., "docker-language-server")
func newStartCmd(baseCommandName string) *startCmd {
	cmd := startCmd{
		Command: &cobra.Command{
			Use:   "start",
			Short: "Start the Docker LSP server",
			Long: `Start the Docker LSP server.

By default, the server will run in stdio mode: requests should be written to
stdin and responses will be written to stdout. (All logging is _always_ done
to stderr.)

For socket mode, pass the --address option.
`,
		},
	}

	var example bytes.Buffer
	p := exampleTemplateParams{
		BaseCommandName: baseCommandName,
	}
	err := exampleTemplate.Execute(&example, p)
	if err != nil {
		panic(err)
	}
	cmd.Example = example.String()

	cmd.RunE = func(cc *cobra.Command, args []string) error {
		ctx := cc.Context()
		if cmd.address != "" {
			err = runSocketServer(ctx, cmd.address)
		} else {
			err = runStdioServer(ctx)
		}
		if err == context.Canceled {
			err = nil
		}
		return err
	}

	cmd.Flags().StringVar(&cmd.address, "address", "",
		"Address (hostname:port) to listen on")
	cmd.Flags().BoolVar(&cmd.stdio, "stdio", false,
		"Stdio (use stdin and stdout to communicate)")
	cmd.MarkFlagsMutuallyExclusive("address", "stdio")
	cmd.MarkFlagsOneRequired("address", "stdio")

	return &cmd
}

func runStdioServer(_ context.Context) error {
	docManager := document.NewDocumentManager()
	s := server.NewServer(docManager)
	s.StartBackgrondProcesses(context.Background())
	return s.RunStdio()
}

func runSocketServer(_ context.Context, addr string) error {
	docManager := document.NewDocumentManager()
	s := server.NewServer(docManager)
	s.StartBackgrondProcesses(context.Background())
	return s.RunTCP(addr)
}
