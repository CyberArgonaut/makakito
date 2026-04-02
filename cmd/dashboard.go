// Package cmd implements the cobra CLI commands for the makakito binary.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Launch the embedded web dashboard",
	RunE: func(_ *cobra.Command, _ []string) error {
		// v0.1 placeholder — dashboard implementation is v0.2.
		fmt.Println("Dashboard is coming in v0.2.")
		fmt.Println("In the meantime, use 'makakito history' and 'makakito status' to review experiments.")
		fmt.Println("")
		fmt.Println("The v0.2 dashboard will be available at http://localhost:7070")
		return nil
	},
}

func init() {
	dashboardCmd.Flags().Int("port", 7070, "Port to serve the dashboard on")
	rootCmd.AddCommand(dashboardCmd)
}
