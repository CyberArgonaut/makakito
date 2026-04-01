package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/CyberArgonaut/makakito/internal/experiment"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate an experiment YAML file without running it",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		path := args[0]

		exp, err := experiment.Load(path)
		if err != nil {
			return fmt.Errorf("loading experiment: %w", err)
		}

		if err := experiment.Validate(exp); err != nil {
			if jsonOutput {
				out := map[string]interface{}{"valid": false, "error": err.Error()}
				return json.NewEncoder(os.Stdout).Encode(out)
			}
			fmt.Fprintf(os.Stderr, "✗ Validation failed:\n%s\n", err)
			os.Exit(1)
		}

		if jsonOutput {
			out := map[string]interface{}{"valid": true, "name": exp.Name, "version": exp.Version}
			return json.NewEncoder(os.Stdout).Encode(out)
		}
		fmt.Printf("✓ Experiment %q is valid (version: %s)\n", exp.Name, exp.Version)
		return nil
	},
}

func init() {
	validateCmd.Flags().Bool("json", false, "output result as JSON")
	rootCmd.AddCommand(validateCmd)
}
