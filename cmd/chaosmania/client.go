package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/actions"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"
	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
	"gopkg.in/yaml.v2"
)

const (
	MaxRepeatsPerPhase = 500
)

func loadPlan(logger *zap.Logger, path string) (actions.Plan, map[string]any, error) {
	var plan actions.Plan
	var raw map[string]any
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return plan, raw, err
	}

	err = yaml.Unmarshal(yamlFile, &plan)
	if err != nil {
		return plan, raw, err
	}

	err = yaml.Unmarshal(yamlFile, &raw)
	if err != nil {
		return plan, raw, err
	}

	for i := range plan.Phases {
		if plan.Phases[i].Repeat == 0 {
			plan.Phases[i].Repeat = 1
		}
	}

	return plan, pkg.Convert(raw).(map[string]any), nil
}

func doRequest(req *http.Request, timeout *time.Duration) (*http.Response, error) {
	client := http.Client{}

	if timeout != nil {
		client.Timeout = *timeout
	}

	if pkg.IsDatadogEnabled() {
		client := httptrace.WrapClient(&client)
		return client.Do(req)
	} else if pkg.IsOpenTelemetryEnabled() {
		client.Transport = otelhttp.NewTransport(http.DefaultTransport)
		return client.Do(req)
	} else {
		return client.Do(req)
	}
}

func sendRequest(logger *zap.Logger, payload map[string]any, host string, port int64, headers map[string]string) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d/", host, port), bytes.NewBuffer(payloadBytes))
	if err != nil {
		logger.Error("failed to create request", zap.Error(err))
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := doRequest(req, nil)

	if err != nil {
		return err
	}

	if resp.StatusCode == 400 {
		s, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		logger.Warn("Request failed:")
		logger.Warn(string(s))
	}

	return nil
}

type statisticCounters struct {
	Errors               uint64
	Requests             uint64
	DurationMicroseconds uint64
}

type statistics struct {
	mu             sync.Mutex
	counters       statisticCounters
	allStatusCodes map[int]int
}

func runWorker(logger *zap.Logger, stats *statistics, timeout time.Duration, ctx context.Context, delay time.Duration, host string, port int64, payloadBytes []byte, headers map[string]string) {
	statusCodes := make(map[int]int)

	// Use 10sec client timeout by default, but allow the client to set a custom timeout
	to := time.Duration(10) * time.Second
	if timeout != 0 {
		to = timeout
	}

loop:
	for {
		select {
		case <-ctx.Done():
			switch ctx.Err() {
			case context.Canceled:
				logger.Debug("Worker stopping due to command completion")
			case context.DeadlineExceeded:
				logger.Debug("Worker stopping due to timeout")
			default:
				logger.Debug("Worker stopping due to context error", zap.Error(ctx.Err()))
			}
			break loop
		case <-time.After(delay):
			start := time.Now()

			// Create a new HTTP POST request with the payload
			url := fmt.Sprintf("http://%s:%d/", host, port)
			req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
			if err != nil {
				logger.Error("failed to create request", zap.Error(err))
				return
			}

			// Set the content type header to indicate a JSON payload
			req.Header.Set("Content-Type", "application/json")

			for k, v := range headers {
				if k == "Host" {
					req.Host = v
				} else {
					req.Header.Set(k, v)
				}
			}

			resp, err := doRequest(req, &to)
			took := time.Since(start)

			if err != nil {
				// Check if the error is due to context cancellation
				if ctx.Err() != nil {
					switch ctx.Err() {
					case context.Canceled:
						logger.Debug("Request cancelled due to command completion")
					case context.DeadlineExceeded:
						logger.Debug("Request cancelled due to timeout")
					default:
						logger.Debug("Request cancelled due to context error", zap.Error(ctx.Err()))
					}
					break loop
				}
				atomic.AddUint64(&stats.counters.Errors, 1)
				continue
			}

			if resp.StatusCode == 400 {
				s, err := io.ReadAll(resp.Body)
				if err != nil {
					resp.Body.Close()
					continue
				}

				logger.Warn(string(s))
			} else if resp.StatusCode > 400 {
				atomic.AddUint64(&stats.counters.Errors, 1)
			}

			atomic.AddUint64(&stats.counters.DurationMicroseconds, uint64(took.Microseconds()))
			atomic.AddUint64(&stats.counters.Requests, 1)

			statusCodes[resp.StatusCode] += 1
			resp.Body.Close()
		}
	}

	stats.mu.Lock()
	defer stats.mu.Unlock()

	for k, v := range statusCodes {
		stats.allStatusCodes[k] += v
	}
}

func executePhase(logger *zap.Logger, phase actions.Phase, raw map[string]any, host string, port int64, header map[string]string, ctx context.Context, durations *actions.PhaseDurations, phaseIndex int) error {
	logger.Info("")
	logger.Info(fmt.Sprintf("Starting phase: %s", phase.Name))

	// Setup
	if s, ok := raw["setup"]; ok {
		err := sendRequest(logger, s.(map[string]any), host, port, header)
		if err != nil {
			return err
		}
	}

	phaseStart := time.Now()
	stats := statistics{
		allStatusCodes: make(map[int]int),
	}

	// Get phase duration from PhaseDurations
	phaseDuration := durations.GetPhaseDuration(phaseIndex)
	logger.Info(fmt.Sprintf("Phase duration: %s", phaseDuration))

	for i, w := range phase.Client.Workers {
		logger.Info(fmt.Sprintf("Starting workers: %v", w.Instances))

		// Run workload
		var wg sync.WaitGroup
		wg.Add(int(w.Instances))

		// Create a worker-level context that will be cancelled when either the phase or worker duration is reached
		workerCtx, cancel := context.WithTimeout(ctx, w.Duration)
		defer cancel()

		payloadBytes, err := json.Marshal(raw["workload"])
		if err != nil {
			return err
		}

		for j := 0; j < int(w.Instances); j++ {
			go func(workerNum int, stats *statistics) {
				defer wg.Done()
				runWorker(logger, stats, w.Timeout, workerCtx, w.Delay, host, port, payloadBytes, header)
				if workerCtx.Err() != nil {
					logger.Info(fmt.Sprintf("Worker %d-%d completed due to: %v", i+1, workerNum+1, workerCtx.Err()))
				}
			}(j, &stats)
		}

		wg.Add(1)
		go func(stats *statistics) {
			defer wg.Done()

			var last statisticCounters
			interval := 10
			t := time.NewTicker(time.Duration(interval) * time.Second)
			defer t.Stop()

			for {
				select {
				case <-t.C:
					current := stats.counters

					if last.Requests == 0 {
						last = current
						continue
					}

					duration := current.DurationMicroseconds - last.DurationMicroseconds
					requests := current.Requests - last.Requests
					errors := current.Errors - last.Errors

					var latency time.Duration
					var ok uint64
					if requests > 0 {
						latency = time.Duration(int64(duration/requests)) * time.Microsecond
						ok = requests - errors
					}

					last = current
					logger.Info(fmt.Sprintf("%.1f req/s, avg latency %v, %v errors, %v ok", float64(requests)/float64(interval), latency, errors, ok))
				case <-workerCtx.Done():
					logger.Info(fmt.Sprintf("Worker group %d completed due to: %v", i+1, workerCtx.Err()))
					return
				}
			}
		}(&stats)

		wg.Wait()

		// Check if phase context is done (phase duration reached)
		if ctx.Err() != nil {
			logger.Info(fmt.Sprintf("Phase completed due to: %v", ctx.Err()))
			break
		}
	}

	var a time.Duration
	successfulRequests := stats.counters.Requests - stats.counters.Errors
	if successfulRequests > 0 {
		a = time.Duration(int64(stats.counters.DurationMicroseconds/successfulRequests)) * time.Microsecond
	}

	logger.Info("")
	logger.Info("Phase Summary")
	logger.Info(fmt.Sprintf("Took: %v", time.Since(phaseStart)))
	logger.Info(fmt.Sprintf("Requests: %v", stats.counters.Requests))
	logger.Info(fmt.Sprintf("Errors: %v", stats.counters.Errors))
	logger.Info(fmt.Sprintf("Average Request Duration: %v", a))
	logger.Info("")
	logger.Info("Status codes:")
	for code, count := range stats.allStatusCodes {
		logger.Info(fmt.Sprintf("%v: %v", code, count))
	}

	// Always run teardown, even if context is cancelled
	if t, ok := raw["teardown"]; ok {
		logger.Info("Running teardown...")
		err := sendRequest(logger, t.(map[string]any), host, port, header)
		if err != nil {
			logger.Warn("Teardown failed", zap.Error(err))
			// Don't return the error since we want to ensure the context cancellation propagates
		}
	}

	return ctx.Err()
}

// calculatePhaseDuration calculates the total duration for a single phase
func calculatePhaseDuration(phase actions.Phase) time.Duration {
	var duration time.Duration
	for _, w := range phase.Client.Workers {
		duration += w.Duration
	}
	return duration
}

// calculatePhaseDurations calculates the duration for each phase based on the plan and optional pattern duration
func calculatePhaseDurations(plan actions.Plan, patternDuration time.Duration, repeatsPerPhase int) []time.Duration {
	if patternDuration == 0 {
		// No duration override, use original phase durations
		durations := make([]time.Duration, len(plan.Phases))
		for i, phase := range plan.Phases {
			durations[i] = calculatePhaseDuration(phase)
		}
		return durations
	}

	// With pattern duration override, divide time equally among phases
	// Each phase gets an equal share of the pattern duration
	perPhaseDuration := patternDuration / time.Duration(len(plan.Phases))
	durations := make([]time.Duration, len(plan.Phases))
	for i := range durations {
		durations[i] = perPhaseDuration
	}
	return durations
}

// selectNextPhase determines the next phase to execute based on the pattern
func selectNextPhase(plan actions.Plan, currentPhase int) int {
	switch plan.Pattern {
	case actions.PatternCycle:
		// Move to next phase, wrap around
		return (currentPhase + 1) % len(plan.Phases)
	case actions.PatternRandom:
		// Randomly select next phase
		return rand.Intn(len(plan.Phases))
	case actions.PatternSequence:
		fallthrough
	default:
		// Default to sequence for backward compatibility
		return currentPhase
	}
}

func command_client(logger *zap.Logger, ctx *cli.Context) error {
	planPath := ctx.String("plan")
	host := ctx.String("host")
	port := ctx.Int64("port")
	header := ctx.StringSlice("header")
	patternDurationStr := ctx.String("pattern-duration")
	repeatsPerPhase := ctx.Int("repeats-per-phase")
	phasePattern := ctx.String("phase-pattern")

	// Validate repeats-per-phase
	if repeatsPerPhase < -1 {
		return fmt.Errorf("repeats-per-phase must be -1 (use plan values), 0 (unlimited), or a positive number")
	}
	if repeatsPerPhase > MaxRepeatsPerPhase {
		return fmt.Errorf("repeats-per-phase cannot exceed %d", MaxRepeatsPerPhase)
	}

	// Load and validate plan
	plan, raw, err := loadPlan(logger, planPath)
	if err != nil {
		return err
	}

	// Initialize phase repeats
	phaseRepeats := actions.NewPhaseRepeats(1) // Default to 1 if not specified

	// Handle repeats-per-phase flag
	if repeatsPerPhase == 0 {
		// Convert 0 (unlimited) to MaxRepeatsPerPhase
		repeatsPerPhase = MaxRepeatsPerPhase
		logger.Info(fmt.Sprintf("Converting unlimited repeats (0) to maximum allowed repeats (%d)", MaxRepeatsPerPhase))
		phaseRepeats.DefaultRepeat = MaxRepeatsPerPhase
	} else if repeatsPerPhase > 0 {
		// Override all phase repeats with the flag value
		phaseRepeats.DefaultRepeat = repeatsPerPhase
		logger.Info(fmt.Sprintf("Overriding plan repeats with %d repeats per phase", repeatsPerPhase))
	} else {
		// Use plan values, but validate and normalize them
		for i, phase := range plan.Phases {
			repeat := int(phase.Repeat)
			if repeat <= 0 {
				repeat = 1 // Default to 1 if not specified or invalid
			} else if repeat > MaxRepeatsPerPhase {
				repeat = MaxRepeatsPerPhase // Cap at maximum
				logger.Warn(fmt.Sprintf("Phase %d repeat count (%d) exceeds maximum, capping at %d", i+1, phase.Repeat, MaxRepeatsPerPhase))
			}
			phaseRepeats.SetRepeat(i, repeat)
		}
		logger.Info("Using repeat values from plan")
	}

	// Override pattern if specified
	if phasePattern != "" {
		switch actions.PhasePattern(phasePattern) {
		case actions.PatternSequence, actions.PatternCycle, actions.PatternRandom:
			plan.Pattern = actions.PhasePattern(phasePattern)
		default:
			return fmt.Errorf("invalid phase pattern: %s. Must be one of: sequence, cycle, random", phasePattern)
		}
	}

	// Set default pattern if not specified
	if plan.Pattern == "" {
		plan.Pattern = actions.PatternSequence
	}

	// Create headers map
	headers := make(map[string]string)
	for _, h := range header {
		parts := strings.Split(h, ":")
		headers[parts[0]] = parts[1]
	}

	logger.Info("Successfully loaded execution plan")
	logger.Info(fmt.Sprintf("Phases: %d", len(plan.Phases)))
	logger.Info(fmt.Sprintf("Phase pattern: %s", plan.Pattern))

	shutdown := InitOTLPProvider(logger)
	defer shutdown()

	// Parse pattern duration if specified
	var patternDuration time.Duration
	if patternDurationStr != "" {
		duration, err := time.ParseDuration(patternDurationStr)
		if err != nil {
			return fmt.Errorf("invalid duration format: %w", err)
		}
		if duration < time.Second {
			return fmt.Errorf("duration must be at least 1 second")
		}
		if duration > 28*24*time.Hour {
			return fmt.Errorf("duration cannot exceed 28 days")
		}
		patternDuration = duration
	}

	// Create PhaseDurations instance for all duration calculations
	durations := actions.NewPhaseDurations(patternDuration, &plan, phaseRepeats)

	// Log duration information
	if patternDuration > 0 {
		totalRepeats := phaseRepeats.GetTotalRepeats(len(plan.Phases))
		if durations.PatternDuration != durations.AdjustedPatternDuration {
			logger.Info(fmt.Sprintf("Pattern duration: %s (adjusted from %s due to minimum/maximum limits)", durations.AdjustedPatternDuration, durations.PatternDuration))
		} else {
			logger.Info(fmt.Sprintf("Pattern duration: %s", durations.PatternDuration))
		}
		logger.Info(fmt.Sprintf("Total runtime will be: %s (pattern duration × %d total repeats)", durations.AdjustedPatternDuration, totalRepeats))
		for i := range plan.Phases {
			phaseTotalDuration := durations.GetPhaseTotalDuration(i)
			logger.Info(fmt.Sprintf("Phase %d total runtime: %s (%s × %d repeats)", i+1, phaseTotalDuration, durations.GetPhaseDuration(i), phaseRepeats.GetRepeat(i)))
		}
	} else {
		totalDuration := durations.GetTotalDuration()
		logger.Info(fmt.Sprintf("Total estimated execution time: %s", totalDuration))
	}

	// Create root context for cancellation propagation
	rootCtx, cancel := context.WithCancel(context.Background())
	defer func() {
		logger.Info("Command completed, initiating graceful shutdown...")
		cancel()
	}()

	// Create pattern executor
	patternExecutor := actions.NewPatternExecutor(plan.Pattern, len(plan.Phases))

	// Execute phases based on pattern
	currentPhase := 0
	phaseExecutions := make([]int, len(plan.Phases))

	for {
		// Check if we've hit the repeat limit for current phase
		if phaseExecutions[currentPhase] >= phaseRepeats.GetRepeat(currentPhase) {
			nextPhase := patternExecutor.NextPhase(currentPhase, phaseExecutions, phaseRepeats)
			if nextPhase == -1 {
				logger.Info("All phases completed their repeats, stopping execution")
				return nil
			}
			currentPhase = nextPhase
			continue
		}

		// Get the duration for this phase
		phaseDuration := durations.GetPhaseDuration(currentPhase)

		// Create a phase context that will be cancelled when the phase duration is reached
		phaseCtx, phaseCancel := context.WithTimeout(rootCtx, phaseDuration)

		// Execute current phase
		err := executePhase(logger, plan.Phases[currentPhase], raw["phases"].([]any)[currentPhase].(map[string]any), host, port, headers, phaseCtx, durations, currentPhase)

		// Always cancel the phase context after execution
		phaseCancel()

		// Increment phase execution counter
		phaseExecutions[currentPhase]++

		// Check if we should advance to the next phase
		if err == context.DeadlineExceeded {
			logger.Info(fmt.Sprintf("Phase %d completed after reaching its time limit", currentPhase+1))
		} else if err != nil {
			return err
		}

		// Let the pattern executor decide if we should advance
		if patternExecutor.ShouldAdvancePhase(currentPhase, phaseExecutions, phaseRepeats) {
			nextPhase := patternExecutor.NextPhase(currentPhase, phaseExecutions, phaseRepeats)
			if nextPhase == -1 {
				logger.Info("All phases completed their repeats, stopping execution")
				return nil
			}
			currentPhase = nextPhase
		}
	}
}
