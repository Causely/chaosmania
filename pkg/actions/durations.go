package actions

import (
	"fmt"
	"time"
)

const (
	// MinPhaseDuration is the minimum allowed duration for any phase
	MinPhaseDuration = 30 * time.Second
	// MaxPhaseDuration is the maximum allowed duration for any phase
	MaxPhaseDuration = 672 * time.Hour // 28 days
)

// PhaseDurations manages duration calculations for phases
type PhaseDurations struct {
	// PatternDuration is the total duration for the pattern if overridden
	PatternDuration time.Duration
	// plan is the execution plan
	plan *Plan
	// repeats is the number of repeats per phase
	repeats *PhaseRepeats
	// cachedDurations stores calculated durations for each phase
	cachedDurations map[int]time.Duration
	// AdjustedPatternDuration is the actual pattern duration after minimum/maximum validation
	AdjustedPatternDuration time.Duration
}

// NewPhaseDurations creates a new PhaseDurations instance
func NewPhaseDurations(patternDuration time.Duration, plan *Plan, repeats *PhaseRepeats) *PhaseDurations {
	pd := &PhaseDurations{
		PatternDuration: patternDuration,
		plan:            plan,
		repeats:         repeats,
		cachedDurations: make(map[int]time.Duration),
	}

	// Calculate and validate pattern duration if overridden
	if patternDuration > 0 {
		totalRepeats := repeats.GetTotalRepeats(len(plan.Phases))
		minTotalDuration := MinPhaseDuration * time.Duration(totalRepeats)
		maxTotalDuration := MaxPhaseDuration * time.Duration(totalRepeats)

		if patternDuration < minTotalDuration {
			pd.AdjustedPatternDuration = minTotalDuration
		} else if patternDuration > maxTotalDuration {
			pd.AdjustedPatternDuration = maxTotalDuration
		} else {
			pd.AdjustedPatternDuration = patternDuration
		}
	}

	return pd
}

// GetPhaseDuration calculates the duration for a specific phase
func (pd *PhaseDurations) GetPhaseDuration(phaseIndex int) time.Duration {
	// Return cached duration if available
	if duration, ok := pd.cachedDurations[phaseIndex]; ok {
		return duration
	}

	var duration time.Duration
	if pd.PatternDuration > 0 {
		// With pattern duration override, calculate based on total repeats
		totalRepeats := pd.repeats.GetTotalRepeats(len(pd.plan.Phases))
		// Each phase execution gets an equal share of the total duration
		duration = pd.AdjustedPatternDuration / time.Duration(totalRepeats)
	} else {
		// Without override, use the phase's original duration
		duration = calculatePhaseDuration(pd.plan.Phases[phaseIndex])
		// If duration is 0 (missing) or below minimum, use minimum
		if duration == 0 || duration < MinPhaseDuration {
			duration = MinPhaseDuration
		}
	}

	// Enforce maximum duration limit
	if duration > MaxPhaseDuration {
		duration = MaxPhaseDuration
	}

	// Cache the calculated duration
	pd.cachedDurations[phaseIndex] = duration
	return duration
}

// GetTotalDuration calculates the total duration across all phases
func (pd *PhaseDurations) GetTotalDuration() time.Duration {
	if pd.PatternDuration > 0 {
		return pd.AdjustedPatternDuration
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
	if pd.PatternDuration > 0 {
		if pd.AdjustedPatternDuration != pd.PatternDuration {
			return fmt.Sprintf("Pattern duration: %s (adjusted from %s due to minimum/maximum limits)", pd.AdjustedPatternDuration, pd.PatternDuration)
		}
		return fmt.Sprintf("Pattern duration: %s", pd.PatternDuration)
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
