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
				atomic.AddUint64(&stats.counters.Errors, 1)
				continue
			}

			if resp.StatusCode == 400 {
				s, err := io.ReadAll(resp.Body)
				if err != nil {
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

func executePhase(logger *zap.Logger, phase actions.Phase, raw map[string]any, host string, port int64, header map[string]string) error {
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

		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, w.Duration)
		defer cancel()

		payloadBytes, err := json.Marshal(raw["workload"])
		if err != nil {
			return err
		}

		for i := 0; i < int(w.Instances); i++ {
			go func(stats *statistics) {
				defer wg.Done()
				runWorker(logger, stats, w.Timeout, ctx, w.Delay, host, port, payloadBytes, header)
			}(&stats)
		}

		wg.Add(1)
		go func(stats *statistics) {
			defer wg.Done()

			var last statisticCounters
			interval := 10
			t := time.NewTicker(time.Duration(interval) * time.Second)

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
				case <-ctx.Done():
					return
				}
			}
		}(&stats)

		wg.Wait()
	}

	// Teardown
	if t, ok := raw["teardown"]; ok {
		err := sendRequest(logger, t.(map[string]any), host, port, header)
		if err != nil {
			return err
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

	return nil
}

func command_client(logger *zap.Logger, ctx *cli.Context) error {
	planPath := ctx.String("plan")
	host := ctx.String("host")
	port := ctx.Int64("port")
	header := ctx.StringSlice("header")

	headers := make(map[string]string)
	for _, h := range header {
		parts := strings.Split(h, ":")
		headers[parts[0]] = parts[1]
	}

	plan, raw, err := loadPlan(logger, planPath)
	if err != nil {
		return err
	}

	shutdown := InitOTLPProvider(logger)
	defer shutdown()

	logger.Info("Successfully loaded execution plan")
	logger.Info(fmt.Sprintf("Phases: %d", len(plan.Phases)))

	var totalSeconds int
	for _, phase := range plan.Phases {
		var phaseSeconds int
		for _, w := range phase.Client.Workers {
			phaseSeconds += int(w.Duration.Seconds())
		}

		totalSeconds += phaseSeconds * int(phase.Repeat)
	}

	logger.Info(fmt.Sprintf("Total estimated execution time: %s", time.Duration(totalSeconds*int(time.Second))))

	for i, phase := range plan.Phases {
		for j := 0; j < int(phase.Repeat); j++ {
			err := executePhase(logger, phase, raw["phases"].([]any)[i].(map[string]any), host, port, headers)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
