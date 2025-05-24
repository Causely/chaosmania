package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

func executePhase(logger *zap.Logger, phase actions.Phase, raw map[string]any, host string, port int64, header map[string]string, ctx context.Context) error {
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

	for _, w := range phase.Client.Workers {
		logger.Info(fmt.Sprintf("Starting workers: %v", w.Instances))

		// Run workload
		var wg sync.WaitGroup
		wg.Add(int(w.Instances))

		// Create a child context with timeout that will be cancelled when parent is cancelled
		workerCtx, cancel := context.WithTimeout(ctx, w.Duration)
		defer cancel()

		payloadBytes, err := json.Marshal(raw["workload"])
		if err != nil {
			return err
		}

		for i := 0; i < int(w.Instances); i++ {
			go func(stats *statistics) {
				defer wg.Done()
				runWorker(logger, stats, w.Timeout, workerCtx, w.Delay, host, port, payloadBytes, header)
			}(&stats)
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
					return
				}
			}
		}(&stats)

		wg.Wait()
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

// calculateScaleFactor calculates the scale factor needed to fit all phases within the target duration
func calculateScaleFactor(plan actions.Plan, targetDuration time.Duration) float64 {
	var totalPlannedDuration time.Duration
	for _, phase := range plan.Phases {
		// Only consider the first iteration of each phase
		for _, w := range phase.Client.Workers {
			totalPlannedDuration += w.Duration
		}
	}

	if totalPlannedDuration == 0 {
		return 1.0
	}

	return float64(targetDuration) / float64(totalPlannedDuration)
}

// calculatePhaseDuration calculates the total duration for a single phase
func calculatePhaseDuration(phase actions.Phase) time.Duration {
	var duration time.Duration
	for _, w := range phase.Client.Workers {
		duration += w.Duration
	}
	return duration
}

// calculatePhaseDurations calculates the duration for each phase based on the plan and optional total duration
func calculatePhaseDurations(plan actions.Plan, totalDuration time.Duration) []time.Duration {
	if totalDuration == 0 {
		// No duration override, use original phase durations
		durations := make([]time.Duration, len(plan.Phases))
		for i, phase := range plan.Phases {
			durations[i] = calculatePhaseDuration(phase)
		}
		return durations
	}

	// With duration override, divide time equally among phases
	perPhaseDuration := totalDuration / time.Duration(len(plan.Phases))
	durations := make([]time.Duration, len(plan.Phases))
	for i := range durations {
		durations[i] = perPhaseDuration
	}
	return durations
}

func command_client(logger *zap.Logger, ctx *cli.Context) error {
	planPath := ctx.String("plan")
	host := ctx.String("host")
	port := ctx.Int64("port")
	header := ctx.StringSlice("header")
	durationStr := ctx.String("total-duration")
	maxRepeats := ctx.Int("max-repeats")

	// Validate max-repeats
	if maxRepeats < 0 {
		return fmt.Errorf("max-repeats must be 0 (unlimited) or a positive number")
	}
	if maxRepeats > 500 {
		return fmt.Errorf("max-repeats cannot exceed 500")
	}

	headers := make(map[string]string)
	for _, h := range header {
		parts := strings.Split(h, ":")
		headers[parts[0]] = parts[1]
	}

	plan, raw, err := loadPlan(logger, planPath)
	if err != nil {
		return err
	}

	logger.Info("Successfully loaded execution plan")
	logger.Info(fmt.Sprintf("Phases: %d", len(plan.Phases)))

	shutdown := InitOTLPProvider(logger)
	defer shutdown()

	// Parse duration if specified
	var totalDuration time.Duration
	if durationStr != "" {
		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			return fmt.Errorf("invalid duration format: %w", err)
		}
		if duration < time.Second {
			return fmt.Errorf("duration must be at least 1 second")
		}
		if duration > 28*24*time.Hour {
			return fmt.Errorf("duration cannot exceed 28 days")
		}
		totalDuration = duration
	}

	// Calculate phase durations upfront
	phaseDurations := calculatePhaseDurations(plan, totalDuration)

	// Log duration information
	if totalDuration > 0 {
		// Calculate actual total duration accounting for repeats
		actualDuration := totalDuration
		if maxRepeats > 0 {
			actualDuration = totalDuration * time.Duration(maxRepeats)
		}
		logger.Info(fmt.Sprintf("Test will run for maximum duration: %s", actualDuration))
		for i, duration := range phaseDurations {
			phaseActualDuration := duration
			if maxRepeats > 0 {
				phaseActualDuration = duration * time.Duration(maxRepeats)
			}
			logger.Info(fmt.Sprintf("Phase %d will run for: %s", i+1, phaseActualDuration))
		}
	} else {
		var totalSeconds int
		for _, phase := range plan.Phases {
			var phaseSeconds int
			for _, w := range phase.Client.Workers {
				phaseSeconds += int(w.Duration.Seconds())
			}
			// If maxRepeats is set, use that value for each phase
			phaseRepeats := int(phase.Repeat)
			if maxRepeats > 0 {
				phaseRepeats = maxRepeats
			}
			totalSeconds += phaseSeconds * phaseRepeats
		}
		logger.Info(fmt.Sprintf("Total estimated execution time: %s", time.Duration(totalSeconds*int(time.Second))))
	}

	if maxRepeats > 0 {
		logger.Info(fmt.Sprintf("Each phase will run %d times", maxRepeats))
	}

	// Create root context for cancellation propagation
	rootCtx, cancel := context.WithCancel(context.Background())
	defer func() {
		logger.Info("Command completed, initiating graceful shutdown...")
		cancel()
	}()

	// Execute each phase with its calculated duration
	for i, phase := range plan.Phases {
		// Determine how many times to repeat this phase
		phaseRepeats := int(phase.Repeat)
		if maxRepeats > 0 {
			phaseRepeats = maxRepeats
		}

		for j := 0; j < phaseRepeats; j++ {
			// Create a new context for each phase execution with its specific duration
			phaseCtx, phaseCancel := context.WithTimeout(rootCtx, phaseDurations[i])
			err := executePhase(logger, phase, raw["phases"].([]any)[i].(map[string]any), host, port, headers, phaseCtx)
			phaseCancel()

			if err != nil {
				// Only treat deadline exceeded as success if this is the last phase
				if err == context.DeadlineExceeded && totalDuration > 0 {
					if i == len(plan.Phases)-1 && j == phaseRepeats-1 {
						logger.Info("Test completed successfully after reaching specified duration limit")
						return nil
					}
					// For non-final phases, continue to the next phase
					logger.Info(fmt.Sprintf("Phase %d completed after reaching its time limit, continuing to next phase", i+1))
					continue
				}
				return err
			}
		}
	}

	return nil
}
