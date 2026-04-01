package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/CyberArgonaut/makakito/pkg/schema"
	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "List past experiment results",
	RunE: func(cmd *cobra.Command, _ []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		statusFilter, _ := cmd.Flags().GetString("status")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		logger := newLogger(cmd)

		st, err := openStore(logger)
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}
		defer st.Close() //nolint:errcheck

		var results []schema.ExperimentResult
		if statusFilter != "" {
			results, err = st.ListByStatus(cmd.Context(), schema.ExperimentStatus(statusFilter), limit)
		} else {
			results, err = st.List(cmd.Context(), limit)
		}
		if err != nil {
			return fmt.Errorf("listing history: %w", err)
		}

		if jsonOutput {
			return json.NewEncoder(os.Stdout).Encode(results)
		}

		if len(results) == 0 {
			fmt.Println("No experiment history.")
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
	historyCmd.Flags().Int("limit", 20, "Maximum number of results to show")
	historyCmd.Flags().String("status", "", "Filter by status (passed, failed, aborted, error)")
	historyCmd.Flags().Bool("json", false, "Output as JSON")
	rootCmd.AddCommand(historyCmd)
}
