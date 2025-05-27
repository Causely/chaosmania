package actions

import (
	"math/rand"
	"time"
)

// PhasePattern defines how phases are executed in sequence
type PhasePattern string

const (
	// PatternSequence executes each phase's repeats before moving to the next phase
	PatternSequence PhasePattern = "sequence"
	// PatternCycle cycles through all phases before repeating
	PatternCycle PhasePattern = "cycle"
	// PatternRandom randomly selects the next phase
	PatternRandom PhasePattern = "random"
)

// PhaseRepeats defines how many times each phase should be repeated
type PhaseRepeats struct {
	// DefaultRepeat is used when no specific repeat is set for a phase
	DefaultRepeat int
	// PhaseRepeats maps phase index to its specific repeat count
	PhaseRepeats map[int]int
}

// NewPhaseRepeats creates a new PhaseRepeats with the given default value
func NewPhaseRepeats(defaultRepeat int) *PhaseRepeats {
	return &PhaseRepeats{
		DefaultRepeat: defaultRepeat,
		PhaseRepeats:  make(map[int]int),
	}
}

// GetRepeat returns the repeat count for a specific phase
func (pr *PhaseRepeats) GetRepeat(phaseIndex int) int {
	if repeat, ok := pr.PhaseRepeats[phaseIndex]; ok {
		return repeat
	}
	return pr.DefaultRepeat
}

// GetTotalRepeats returns the total number of repeats across all phases
func (pr *PhaseRepeats) GetTotalRepeats(numPhases int) int {
	if len(pr.PhaseRepeats) == 0 {
		// If no specific phase repeats are set, use default for all phases
		return pr.DefaultRepeat * numPhases
	}

	// Sum up all phase-specific repeats
	total := 0
	for i := 0; i < numPhases; i++ {
		total += pr.GetRepeat(i)
	}
	return total
}

// SetRepeat sets the repeat count for a specific phase
func (pr *PhaseRepeats) SetRepeat(phaseIndex, repeat int) {
	pr.PhaseRepeats[phaseIndex] = repeat
}

// PhasePatternExecutor defines the interface for executing phase patterns
type PhasePatternExecutor interface {
	// NextPhase returns the next phase index to execute, or -1 if no more phases should be executed
	NextPhase(currentPhase int, phaseExecutions []int, repeats *PhaseRepeats) int
	// IsComplete returns true if all phases have completed their required executions
	IsComplete(phaseExecutions []int, repeats *PhaseRepeats) bool
	// ShouldAdvancePhase returns true if the current phase should advance to the next phase
	ShouldAdvancePhase(currentPhase int, phaseExecutions []int, repeats *PhaseRepeats) bool
}

// SequencePattern implements sequential phase execution
type SequencePattern struct {
	numPhases int
}

func NewSequencePattern(numPhases int) *SequencePattern {
	return &SequencePattern{numPhases: numPhases}
}

func (p *SequencePattern) NextPhase(currentPhase int, phaseExecutions []int, repeats *PhaseRepeats) int {
	nextPhase := currentPhase + 1
	if nextPhase >= p.numPhases {
		return -1
	}
	return nextPhase
}

func (p *SequencePattern) IsComplete(phaseExecutions []int, repeats *PhaseRepeats) bool {
	for i, execs := range phaseExecutions {
		if execs < repeats.GetRepeat(i) {
			return false
		}
	}
	return true
}

func (p *SequencePattern) ShouldAdvancePhase(currentPhase int, phaseExecutions []int, repeats *PhaseRepeats) bool {
	// For sequence pattern, always advance after a phase completes
	return true
}

// CyclePattern implements cycling through phases
type CyclePattern struct {
	numPhases int
}

func NewCyclePattern(numPhases int) *CyclePattern {
	return &CyclePattern{numPhases: numPhases}
}

func (p *CyclePattern) NextPhase(currentPhase int, phaseExecutions []int, repeats *PhaseRepeats) int {
	if phaseExecutions[currentPhase] >= repeats.GetRepeat(currentPhase) {
		nextPhase := (currentPhase + 1) % p.numPhases
		if p.IsComplete(phaseExecutions, repeats) {
			return -1
		}
		return nextPhase
	}
	return (currentPhase + 1) % p.numPhases
}

func (p *CyclePattern) IsComplete(phaseExecutions []int, repeats *PhaseRepeats) bool {
	for i, execs := range phaseExecutions {
		if execs < repeats.GetRepeat(i) {
			return false
		}
	}
	return true
}

func (p *CyclePattern) ShouldAdvancePhase(currentPhase int, phaseExecutions []int, repeats *PhaseRepeats) bool {
	// For cycle pattern, always advance after a phase completes
	return true
}

// RandomPattern implements random phase selection
type RandomPattern struct {
	numPhases int
	rng       *rand.Rand
}

func NewRandomPattern(numPhases int) *RandomPattern {
	return &RandomPattern{
		numPhases: numPhases,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (p *RandomPattern) NextPhase(currentPhase int, phaseExecutions []int, repeats *PhaseRepeats) int {
	// Find available phases that haven't completed their repeats
	var availablePhases []int
	for i, execs := range phaseExecutions {
		if execs < repeats.GetRepeat(i) {
			availablePhases = append(availablePhases, i)
		}
	}

	if len(availablePhases) == 0 {
		return -1
	}

	return availablePhases[p.rng.Intn(len(availablePhases))]
}

func (p *RandomPattern) IsComplete(phaseExecutions []int, repeats *PhaseRepeats) bool {
	for i, execs := range phaseExecutions {
		if execs < repeats.GetRepeat(i) {
			return false
		}
	}
	return true
}

func (p *RandomPattern) ShouldAdvancePhase(currentPhase int, phaseExecutions []int, repeats *PhaseRepeats) bool {
	// For random pattern, always advance after a phase completes
	return true
}

// NewPatternExecutor creates a new pattern executor based on the pattern type
func NewPatternExecutor(pattern PhasePattern, numPhases int) PhasePatternExecutor {
	switch pattern {
	case PatternSequence:
		return NewSequencePattern(numPhases)
	case PatternCycle:
		return NewCyclePattern(numPhases)
	case PatternRandom:
		return NewRandomPattern(numPhases)
	default:
		return NewSequencePattern(numPhases)
	}
}
