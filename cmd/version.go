package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of kyma",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Kyma %s\n", version)
	},
}
