package engine

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/CyberArgonaut/makakito/internal/experiment"
	"github.com/CyberArgonaut/makakito/internal/store"
	"github.com/CyberArgonaut/makakito/pkg/schema"
)

// RunnerConfig holds runtime options for an experiment run.
type RunnerConfig struct {
	DryRun      bool
	Force       bool
	Environment string
}

// Runner orchestrates experiment execution.
type Runner struct {
	targets *TargetRegistry
	faults  *FaultRegistry
	probes  *ProbeRegistry
	store   store.Store
	logger  *slog.Logger
}

// NewRunner creates a Runner with the given registries and store.
func NewRunner(
	targets *TargetRegistry,
	faults *FaultRegistry,
	probes *ProbeRegistry,
	st store.Store,
	logger *slog.Logger,
) *Runner {
	return &Runner{
		targets: targets,
		faults:  faults,
		probes:  probes,
		store:   st,
		logger:  logger,
	}
}

// Run executes an experiment and returns its result.
func (r *Runner) Run(ctx context.Context, exp *schema.Experiment, cfg RunnerConfig) (*schema.ExperimentResult, error) {
	if err := experiment.Validate(exp); err != nil {
		return nil, fmt.Errorf("invalid experiment: %w", err)
	}

	// Safety Rule #1: enforce timeout.
	timeout := exp.Controls.Timeout.Duration
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Check environment allowlist.
	if len(exp.Controls.Environments) > 0 && cfg.Environment != "" {
		if !slices.Contains(exp.Controls.Environments, cfg.Environment) {
			return nil, fmt.Errorf("environment %q not in allowlist %v", cfg.Environment, exp.Controls.Environments)
		}
	}

	id := NewExperimentID()
	now := time.Now().UTC()
	result := &schema.ExperimentResult{
		ID:        id,
		Name:      exp.Name,
		Status:    schema.StatusRunning,
		StartedAt: now,
	}

	// Persist running state.
	if err := r.store.Save(ctx, result); err != nil {
		r.logger.Warn("failed to persist initial run state", "error", err)
	}

	// Safety Rule #4: require_approval gate.
	if exp.Controls.RequireApproval && !cfg.Force {
		if !promptApproval(exp.Name) {
			result.Status = schema.StatusAborted
			result.Error = "user declined approval"
			r.finalize(result)
			return result, nil
		}
	}

	// Safety Rule #2: always call rollbacks — even on panic.
	rollbacks := NewRollbackStack(r.logger)
	var panicVal interface{}
	func() {
		defer func() {
			panicVal = recover()
		}()
		r.runExperiment(ctx, exp, cfg, result, rollbacks)
	}()

	rollbacks.Execute()

	if panicVal != nil {
		result.Status = schema.StatusError
		result.Error = fmt.Sprintf("panic: %v", panicVal)
	}

	r.finalize(result)
	return result, nil
}

func (r *Runner) runExperiment(
	ctx context.Context,
	exp *schema.Experiment,
	cfg RunnerConfig,
	result *schema.ExperimentResult,
	rollbacks *RollbackStack,
) {
	// Run BEFORE probes.
	r.logger.Info("checking hypothesis (before)", "experiment", exp.Name)
	if !r.runProbes(ctx, exp.Hypothesis.Probes, schema.ProbePhaseBefore, result) {
		result.Status = schema.StatusAborted
		result.Error = "steady-state hypothesis not met before experiment"
		return
	}

	// Execute each action.
	for _, action := range exp.Method {
		if ctx.Err() != nil {
			result.Status = schema.StatusAborted
			result.Error = "experiment timed out or was cancelled"
			return
		}
		if err := r.runAction(ctx, action, exp.Controls.BlastRadius, cfg, result, rollbacks); err != nil {
			result.Status = schema.StatusError
			result.Error = err.Error()
			return
		}
	}

	// Run AFTER probes.
	r.logger.Info("checking hypothesis (after)", "experiment", exp.Name)
	if !r.runProbes(ctx, exp.Hypothesis.Probes, schema.ProbePhaseAfter, result) {
		result.Status = schema.StatusFailed
		return
	}

	result.Status = schema.StatusPassed
}

func (r *Runner) runAction(
	ctx context.Context,
	action schema.Action,
	blastRadius schema.BlastRadius,
	cfg RunnerConfig,
	result *schema.ExperimentResult,
	rollbacks *RollbackStack,
) error {
	log := r.logger.With("action", action.Name, "target", action.Target.Kind, "fault", action.Fault.Kind)

	target, err := r.targets.Get(action.Target.Kind)
	if err != nil {
		return fmt.Errorf("action %q: %w", action.Name, err)
	}

	sel := Selector(action.Target.Selector)
	resources, err := target.Resolve(ctx, sel)
	if err != nil {
		return fmt.Errorf("action %q: resolving targets: %w", action.Name, err)
	}
	if len(resources) == 0 {
		return fmt.Errorf("action %q: no resources matched selector", action.Name)
	}

	// Safety Rule #5: enforce blast radius.
	resources = EnforceBlastRadius(resources, blastRadius)
	log.Info("resolved resources", "count", len(resources))

	fault, err := r.faults.Get(action.Fault.Kind)
	if err != nil {
		return fmt.Errorf("action %q: %w", action.Name, err)
	}

	faultParams := Params(action.Fault.Params)
	if err := fault.Validate(faultParams); err != nil {
		return fmt.Errorf("action %q: invalid fault params: %w", action.Name, err)
	}

	resourceNames := make([]string, len(resources))
	for i, res := range resources {
		resourceNames[i] = res.Name
	}
	actionLog := schema.ActionLog{
		ActionName: action.Name,
		TargetKind: action.Target.Kind,
		FaultKind:  action.Fault.Kind,
		Resources:  resourceNames,
	}

	for _, resource := range resources {
		if ctx.Err() != nil {
			return fmt.Errorf("context cancelled before applying fault to %q", resource.Name)
		}

		// Safety Rule #3: dry-run must never touch real infrastructure.
		if cfg.DryRun {
			log.Info("[dry-run] would apply fault", "resource", resource.Name)
			continue
		}

		log.Info("applying fault", "resource", resource.Name)
		rollback, err := fault.Apply(ctx, resource, faultParams)
		if rollback != nil {
			rollbacks.Push(rollback)
		}
		if err != nil {
			actionLog.Error = err.Error()
			result.ActionLogs = append(result.ActionLogs, actionLog)
			return fmt.Errorf("action %q: applying fault to %q: %w", action.Name, resource.Name, err)
		}
	}

	// Wait for fault duration.
	if action.Duration.Duration > 0 && !cfg.DryRun {
		log.Info("fault active, waiting", "duration", action.Duration.Duration)
		select {
		case <-time.After(action.Duration.Duration):
		case <-ctx.Done():
			actionLog.Error = "context cancelled during fault duration"
			result.ActionLogs = append(result.ActionLogs, actionLog)
			return fmt.Errorf("action %q: context cancelled during fault duration", action.Name)
		}
	}

	actionLog.RolledBack = true
	result.ActionLogs = append(result.ActionLogs, actionLog)
	return nil
}

func (r *Runner) runProbes(
	ctx context.Context,
	probeSpecs []schema.ProbeSpec,
	phase schema.ProbePhase,
	result *schema.ExperimentResult,
) bool {
	allPassed := true
	for _, spec := range probeSpecs {
		probe, err := r.probes.Get(spec.Type)
		if err != nil {
			r.logger.Error("unknown probe type", "type", spec.Type)
			allPassed = false
			continue
		}

		params := probeParams(spec)
		pr, err := probe.Check(ctx, params)
		checkedAt := time.Now().UTC()

		var probeResult schema.ProbeResult
		if err != nil {
			probeResult = schema.ProbeResult{
				ProbeType: spec.Type,
				Success:   false,
				Message:   err.Error(),
				CheckedAt: checkedAt,
				Phase:     phase,
			}
			allPassed = false
		} else {
			probeResult = schema.ProbeResult{
				ProbeType: spec.Type,
				Success:   pr.Success,
				Message:   pr.Message,
				CheckedAt: checkedAt,
				Phase:     phase,
			}
			if !pr.Success {
				allPassed = false
			}
		}

		r.logger.Info("probe result",
			"type", spec.Type,
			"phase", phase,
			"success", probeResult.Success,
			"message", probeResult.Message,
		)
		result.ProbeResults = append(result.ProbeResults, probeResult)
	}
	return allPassed
}

func (r *Runner) finalize(result *schema.ExperimentResult) {
	now := time.Now().UTC()
	result.FinishedAt = &now
	if err := r.store.Save(context.Background(), result); err != nil {
		r.logger.Error("failed to persist final result", "error", err)
	}
}

func probeParams(spec schema.ProbeSpec) Params {
	params := Params{}
	if spec.URL != "" {
		params["url"] = spec.URL
	}
	if spec.ExpectedStatus != 0 {
		params["expected_status"] = fmt.Sprintf("%d", spec.ExpectedStatus)
	}
	if spec.Timeout.Duration > 0 {
		params["timeout"] = spec.Timeout.String()
	}
	if spec.Query != "" {
		params["query"] = spec.Query
	}
	if spec.Endpoint != "" {
		params["endpoint"] = spec.Endpoint
	}
	return params
}

func promptApproval(name string) bool {
	fmt.Printf("Experiment %q requires approval. Proceed? [y/N]: ", name)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.ToLower(strings.TrimSpace(scanner.Text())) == "y"
	}
	return false
}
