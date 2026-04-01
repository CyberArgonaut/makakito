package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/CyberArgonaut/makakito/pkg/schema"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show currently running experiments",
	RunE: func(cmd *cobra.Command, _ []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		logger := newLogger(cmd)

		st, err := openStore(logger)
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}
		defer st.Close() //nolint:errcheck

		results, err := st.ListByStatus(cmd.Context(), schema.StatusRunning, 50)
		if err != nil {
			return fmt.Errorf("querying status: %w", err)
		}

		if jsonOutput {
			return json.NewEncoder(os.Stdout).Encode(results)
		}

		if len(results) == 0 {
			fmt.Println("No running experiments.")
			return nil
		}
		fmt.Printf("%-24s %-40s %-10s %s\n", "ID", "Name", "Status", "Started")
		for _, r := range results {
			fmt.Printf("%-24s %-40s %-10s %s\n", r.ID, r.Name, r.Status, r.StartedAt.Format("2006-01-02 15:04:05"))
		}
		return nil
	},
}

func init() {
	statusCmd.Flags().Bool("json", false, "Output as JSON")
	rootCmd.AddCommand(statusCmd)
}
