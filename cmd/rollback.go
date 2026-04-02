package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback <id>",
	Short: "Manually trigger rollback for a running or stuck experiment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		logger := newLogger(cmd)

		st, err := openStore(logger)
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}
		defer st.Close() //nolint:errcheck // best-effort close; error not actionable at defer time

		result, err := st.Get(cmd.Context(), id)
		if err != nil {
			return fmt.Errorf("finding experiment %q: %w", id, err)
		}

		// In v0.1, manual rollback is best-effort — the experiment must still be in-process.
		// Full manual rollback (persisting rollback state) is a v0.2 feature.
		fmt.Printf("Experiment %q (status: %s)\n", result.Name, result.Status)
		fmt.Println("Note: manual rollback of completed experiments is not yet implemented.")
		fmt.Println("If the experiment is still running, send SIGINT to the makakito process.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rollbackCmd)
}
