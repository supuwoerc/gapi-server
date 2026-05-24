package system

import (
	"github.com/supuwoerc/gapi-server/internal/app"

	"github.com/spf13/cobra"
)

type CliFactory func() (*app.Cli, error)

func Register(parent *cobra.Command, cliFactory CliFactory) {
	parent.AddCommand(newVersionCmd())
	parent.AddCommand(newWelcomeCmd(cliFactory))
}
