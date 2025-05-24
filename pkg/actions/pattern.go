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

// PhasePatternExecutor defines the interface for executing phase patterns
type PhasePatternExecutor interface {
	// NextPhase returns the next phase index to execute, or -1 if no more phases should be executed
	NextPhase(currentPhase int, phaseExecutions []int, repeatsPerPhase int) int
	// IsComplete returns true if all phases have completed their required executions
	IsComplete(phaseExecutions []int, repeatsPerPhase int) bool
}

// SequencePattern implements sequential phase execution
type SequencePattern struct {
	numPhases int
}

func NewSequencePattern(numPhases int) *SequencePattern {
	return &SequencePattern{numPhases: numPhases}
}

func (p *SequencePattern) NextPhase(currentPhase int, phaseExecutions []int, repeatsPerPhase int) int {
	if repeatsPerPhase > 0 && phaseExecutions[currentPhase] >= repeatsPerPhase {
		nextPhase := currentPhase + 1
		if nextPhase >= p.numPhases {
			return -1
		}
		return nextPhase
	}
	return currentPhase
}

func (p *SequencePattern) IsComplete(phaseExecutions []int, repeatsPerPhase int) bool {
	if repeatsPerPhase <= 0 {
		return false
	}
	for _, execs := range phaseExecutions {
		if execs < repeatsPerPhase {
			return false
		}
	}
	return true
}

// CyclePattern implements cycling through phases
type CyclePattern struct {
	numPhases int
}

func NewCyclePattern(numPhases int) *CyclePattern {
	return &CyclePattern{numPhases: numPhases}
}

func (p *CyclePattern) NextPhase(currentPhase int, phaseExecutions []int, repeatsPerPhase int) int {
	if repeatsPerPhase > 0 && phaseExecutions[currentPhase] >= repeatsPerPhase {
		nextPhase := (currentPhase + 1) % p.numPhases
		if p.IsComplete(phaseExecutions, repeatsPerPhase) {
			return -1
		}
		return nextPhase
	}
	return (currentPhase + 1) % p.numPhases
}

func (p *CyclePattern) IsComplete(phaseExecutions []int, repeatsPerPhase int) bool {
	if repeatsPerPhase <= 0 {
		return false
	}
	for _, execs := range phaseExecutions {
		if execs < repeatsPerPhase {
			return false
		}
	}
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

func (p *RandomPattern) NextPhase(currentPhase int, phaseExecutions []int, repeatsPerPhase int) int {
	if repeatsPerPhase <= 0 {
		return p.rng.Intn(p.numPhases)
	}

	// Find available phases that haven't completed their repeats
	var availablePhases []int
	for i, execs := range phaseExecutions {
		if execs < repeatsPerPhase {
			availablePhases = append(availablePhases, i)
		}
	}

	if len(availablePhases) == 0 {
		return -1
	}

	return availablePhases[p.rng.Intn(len(availablePhases))]
}

func (p *RandomPattern) IsComplete(phaseExecutions []int, repeatsPerPhase int) bool {
	if repeatsPerPhase <= 0 {
		return false
	}
	for _, execs := range phaseExecutions {
		if execs < repeatsPerPhase {
			return false
		}
	}
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
