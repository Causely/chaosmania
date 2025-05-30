package pkg

import "time"

const (
	// MinPhaseDuration is the minimum allowed duration for any phase
	MinPhaseDuration = 60 * time.Second
	// MaxPhaseDuration is the maximum allowed duration for any phase
	MaxPhaseDuration = 28 * 24 * time.Hour // 28 days
)
