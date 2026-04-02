package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/CyberArgonaut/makakito/internal/engine"
	"github.com/CyberArgonaut/makakito/internal/experiment"
	"github.com/CyberArgonaut/makakito/internal/fault"
	"github.com/CyberArgonaut/makakito/internal/probe"
	"github.com/CyberArgonaut/makakito/internal/store"
	"github.com/CyberArgonaut/makakito/internal/target"
	"github.com/CyberArgonaut/makakito/pkg/schema"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var runCmd = &cobra.Command{
	Use:   "run <file>",
	Short: "Execute a chaos experiment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")
		env, _ := cmd.Flags().GetString("environment")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		logger := newLogger(cmd)
		path := args[0]

		exp, err := experiment.Load(path)
		if err != nil {
			return fmt.Errorf("loading experiment: %w", err)
		}
		if err := experiment.Validate(exp); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		st, err := openStore(logger)
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}
		defer st.Close() //nolint:errcheck // best-effort close; error not actionable at defer time

		runner := buildRunner(st, logger)

		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		cfg := engine.RunnerConfig{
			DryRun:      dryRun,
			Force:       force,
			Environment: env,
		}

		if dryRun {
			fmt.Printf("[dry-run] Simulating experiment: %s\n", exp.Name)
		} else {
			fmt.Printf("Running experiment: %s\n", exp.Name)
		}

		result, err := runner.Run(ctx, exp, cfg)
		if err != nil {
			return fmt.Errorf("experiment failed: %w", err)
		}

		printResult(result, jsonOutput)

		if result.Status == schema.StatusFailed || result.Status == schema.StatusError || result.Status == schema.StatusAborted {
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	runCmd.Flags().Bool("dry-run", false, "Validate and simulate without applying faults")
	runCmd.Flags().Bool("force", false, "Bypass require_approval gate (writes to audit log)")
	runCmd.Flags().String("environment", "", "Environment label for allowlist check")
	runCmd.Flags().Bool("json", false, "Output result as JSON")
	rootCmd.AddCommand(runCmd)
}

func buildRunner(st store.Store, logger *slog.Logger) *engine.Runner {
	targets := engine.NewTargetRegistry()
	faults := engine.NewFaultRegistry()
	probes := engine.NewProbeRegistry()

	target.RegisterAll(targets)
	fault.RegisterAll(faults, nil) // nil docker client — real docker registered per target
	probe.RegisterAll(probes)

	return engine.NewRunner(targets, faults, probes, st, logger)
}

func openStore(logger *slog.Logger) (store.Store, error) {
	dbPath := viper.GetString("store.path")
	if dbPath == "" {
		home, _ := os.UserHomeDir()
		dbPath = home + "/.makakito/history.db"
	}
	if err := os.MkdirAll(dbPath[:len(dbPath)-len("/history.db")], 0o755); err != nil {
		logger.Warn("could not create store directory", "error", err)
	}
	return store.NewSQLiteStore(dbPath)
}

func newLogger(cmd *cobra.Command) *slog.Logger {
	level := slog.LevelInfo
	if l, _ := cmd.Flags().GetString("log-level"); l == "" {
		l = viper.GetString("log_level")
		switch l {
		case "debug":
			level = slog.LevelDebug
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

func printResult(result *schema.ExperimentResult, jsonOutput bool) {
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(result) //nolint:errcheck // stdout encoding errors are not actionable
		return
	}

	icon := "✅"
	switch result.Status {
	case schema.StatusFailed:
		icon = "❌"
	case schema.StatusAborted:
		icon = "⚠️ "
	case schema.StatusError:
		icon = "💥"
	}

	duration := ""
	if result.FinishedAt != nil {
		duration = fmt.Sprintf(" in %s", result.FinishedAt.Sub(result.StartedAt).Round(1000000000))
	}
	fmt.Printf("\n%s Experiment %q %s%s\n", icon, result.Name, result.Status, duration)
	if result.Error != "" {
		fmt.Printf("   Error: %s\n", result.Error)
	}
	fmt.Printf("   ID: %s\n", result.ID)
}
