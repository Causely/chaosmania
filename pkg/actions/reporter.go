package actions

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Reporter manages logging and reporting for phase execution
type Reporter struct {
	plan      *Plan
	repeats   *PhaseRepeats
	durations *PhaseDurations
	logger    *zap.Logger
}

// NewReporter creates a new Reporter instance
func NewReporter(plan *Plan, repeats *PhaseRepeats, durations *PhaseDurations, logger *zap.Logger) *Reporter {
	return &Reporter{
		plan:      plan,
		repeats:   repeats,
		durations: durations,
		logger:    logger,
	}
}

// LogRuntimeOverrides logs any runtime overrides that were applied
func (r *Reporter) LogRuntimeOverrides(runtimeDuration time.Duration, repeatsPerPhase int, phasePattern string) {
	// Log runtime duration override if specified
	if runtimeDuration > 0 {
		if r.durations.AdjustedRuntimeDuration != r.durations.RuntimeDuration {
			r.logger.Info(fmt.Sprintf("Runtime duration override: %s (adjusted from %s due to minimum/maximum limits)",
				r.durations.AdjustedRuntimeDuration, r.durations.RuntimeDuration))
		} else {
			r.logger.Info(fmt.Sprintf("Runtime duration override: %s", r.durations.RuntimeDuration))
		}
	}

	// Log repeats override if specified
	if repeatsPerPhase > 0 {
		r.logger.Info(fmt.Sprintf("Repeats override: %d repeats per phase", repeatsPerPhase))
	} else if repeatsPerPhase == 0 {
		r.logger.Info(fmt.Sprintf("Repeats override: unlimited (capped at maximum %d)", MaxRepeatsPerPhase))
	}

	// Log pattern override if specified
	if phasePattern != "" {
		r.logger.Info(fmt.Sprintf("Phase pattern override: %s", phasePattern))
	}
}

// LogPlanSummary logs a summary of the loaded plan
func (r *Reporter) LogPlanSummary() {
	// Log plan load success
	r.logger.Info("Successfully loaded execution plan")

	// Log total summary in a single line
	totalDuration := r.durations.GetTotalDuration()
	r.logger.Info(fmt.Sprintf("Plan summary: %d phases, %s total runtime, %s pattern",
		len(r.plan.Phases), totalDuration, r.plan.Pattern))

	// Log phase details
	for i := range r.plan.Phases {
		phase := r.plan.Phases[i]
		repeats := r.repeats.GetRepeat(i)
		phaseDuration := r.durations.GetPhaseDuration(i)
		phaseTotalDuration := r.durations.GetPhaseTotalDuration(i)
		r.logger.Info(fmt.Sprintf("Phase %d: %s, %d repeats, %s per phase, %s total",
			i+1, phase.Name, repeats, phaseDuration, phaseTotalDuration))
	}

	// Add blank line before execution starts
	r.logger.Info("")
}

// LogPhaseStart logs the start of a phase
func (r *Reporter) LogPhaseStart(phaseIndex int) {
	phase := r.plan.Phases[phaseIndex]
	phaseDuration := r.durations.GetPhaseDuration(phaseIndex)
	r.logger.Info(fmt.Sprintf("Starting phase: %s (duration: %s)", phase.Name, phaseDuration))
}

// PhaseStats holds statistics for phase execution
type PhaseStats struct {
	Requests        uint64
	Errors          uint64
	AverageDuration time.Duration
	StatusCodes     map[int]int
	PhaseStart      time.Time
	PhaseEnd        time.Time
}

// LogPhaseEnd logs the end of a phase with statistics
func (r *Reporter) LogPhaseEnd(phaseIndex int, stats *PhaseStats) {
	phase := r.plan.Phases[phaseIndex]
	r.logger.Info("")
	r.logger.Info(fmt.Sprintf("Phase complete: %s", phase.Name))
	r.logger.Info(fmt.Sprintf("  Duration: %v", stats.PhaseEnd.Sub(stats.PhaseStart)))
	r.logger.Info(fmt.Sprintf("  Requests: %v (%v errors)", stats.Requests, stats.Errors))
	r.logger.Info(fmt.Sprintf("  Average request duration: %v", stats.AverageDuration))

	if len(stats.StatusCodes) > 0 {
		r.logger.Info("  Status codes:")
		for code, count := range stats.StatusCodes {
			r.logger.Info(fmt.Sprintf("    %v: %v", code, count))
		}
	}
	r.logger.Info("")
}
