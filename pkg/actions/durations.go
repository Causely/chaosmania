package actions

import (
	"fmt"
	"time"

	"github.com/Causely/chaosmania/pkg"
)

const (
	// MinPhaseDuration is the minimum allowed duration for any phase
	MinPhaseDuration = 60 * time.Second
	// MaxPhaseDuration is the maximum allowed duration for any phase
	MaxPhaseDuration = 672 * time.Hour // 28 days
)

// PhaseDurations manages duration calculations for phases
type PhaseDurations struct {
	// RuntimeDuration is the total duration for the entire plan if overridden
	RuntimeDuration time.Duration
	// plan is the execution plan
	plan *Plan
	// repeats is the number of repeats per phase
	repeats *PhaseRepeats
	// cachedDurations stores calculated durations for each phase
	cachedDurations map[int]time.Duration
	// AdjustedRuntimeDuration is the actual runtime duration after minimum/maximum validation
	AdjustedRuntimeDuration time.Duration
}

// NewPhaseDurations creates a new PhaseDurations instance
func NewPhaseDurations(runtimeDuration time.Duration, plan *Plan, repeats *PhaseRepeats) *PhaseDurations {
	pd := &PhaseDurations{
		RuntimeDuration: runtimeDuration,
		plan:            plan,
		repeats:         repeats,
		cachedDurations: make(map[int]time.Duration),
	}

	// Calculate total number of phase executions
	totalExecutions := repeats.GetTotalRepeats(len(plan.Phases))

	// Calculate and validate runtime duration if overridden
	if runtimeDuration > 0 {
		// Calculate minimum total runtime (1 minute per phase execution)
		minTotalRuntime := pkg.MinDuration * time.Duration(totalExecutions)
		// Calculate maximum total runtime (28 days)
		maxTotalRuntime := pkg.MaxDuration

		// Adjust runtime if needed
		if runtimeDuration < minTotalRuntime {
			pd.AdjustedRuntimeDuration = minTotalRuntime
		} else if runtimeDuration > maxTotalRuntime {
			pd.AdjustedRuntimeDuration = maxTotalRuntime
		} else {
			pd.AdjustedRuntimeDuration = runtimeDuration
		}
	}

	return pd
}

// GetPhaseDuration returns the duration for a specific phase
func (d *PhaseDurations) GetPhaseDuration(phaseIndex int) time.Duration {
	// Return cached duration if available
	if duration, ok := d.cachedDurations[phaseIndex]; ok {
		return duration
	}

	var duration time.Duration
	if d.RuntimeDuration > 0 {
		// With runtime duration override, divide time equally among all phase executions
		totalExecutions := d.repeats.GetTotalRepeats(len(d.plan.Phases))
		duration = d.AdjustedRuntimeDuration / time.Duration(totalExecutions)
	} else {
		// Without override, use the phase's worker durations
		var maxDuration time.Duration
		for _, w := range d.plan.Phases[phaseIndex].Client.Workers {
			if w.Duration == 0 {
				// Missing duration defaults to minimum
				maxDuration = pkg.MinDuration
			} else if w.Duration > maxDuration {
				maxDuration = w.Duration
			}
		}
		duration = maxDuration
	}

	// Ensure duration is within bounds
	if duration < pkg.MinDuration {
		duration = pkg.MinDuration
	}
	if duration > pkg.MaxDuration {
		duration = pkg.MaxDuration
	}

	// Cache the calculated duration
	d.cachedDurations[phaseIndex] = duration
	return duration
}

// GetTotalDuration calculates the total duration across all phases
func (pd *PhaseDurations) GetTotalDuration() time.Duration {
	if pd.RuntimeDuration > 0 {
		return pd.AdjustedRuntimeDuration
	}

	var total time.Duration
	for i := range pd.plan.Phases {
		total += pd.GetPhaseDuration(i) * time.Duration(pd.repeats.GetRepeat(i))
	}
	return total
}

// GetPhaseTotalDuration returns the total duration for a specific phase including repeats
func (pd *PhaseDurations) GetPhaseTotalDuration(phaseIndex int) time.Duration {
	return pd.GetPhaseDuration(phaseIndex) * time.Duration(pd.repeats.GetRepeat(phaseIndex))
}

// String returns a string representation of the phase durations
func (pd *PhaseDurations) String() string {
	if pd.RuntimeDuration > 0 {
		if pd.AdjustedRuntimeDuration != pd.RuntimeDuration {
			return fmt.Sprintf("Runtime duration: %s (adjusted from %s due to minimum/maximum limits)", pd.AdjustedRuntimeDuration, pd.RuntimeDuration)
		}
		return fmt.Sprintf("Runtime duration: %s", pd.RuntimeDuration)
	}

	var s string
	for i, phase := range pd.plan.Phases {
		duration := pd.GetPhaseDuration(i)
		s += fmt.Sprintf("Phase %d (%s): %s\n", i+1, phase.Name, duration)
	}
	return s
}

// calculatePhaseDuration calculates the total duration for a single phase
func calculatePhaseDuration(phase Phase) time.Duration {
	var duration time.Duration
	for _, w := range phase.Client.Workers {
		duration += w.Duration
	}
	return duration
}
