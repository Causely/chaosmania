package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Causely/chaosmania/pkg/actions"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
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

	return plan, actions.Convert(raw).(map[string]any), nil
}

func sendRequest(logger *zap.Logger, payload map[string]any, host string, port int64) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d/", host, port), bytes.NewBuffer(payloadBytes))
	if err != nil {
		logger.Error("failed to create request", zap.Error(err))
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)

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

type statistics struct {
	countAllErrors             uint64
	countAllRequests           uint64
	sumAllDurationMicroseconds uint64
}

func executePhase(logger *zap.Logger, phase actions.Phase, raw map[string]any, host string, port int64) error {
	logger.Info("")
	logger.Info(fmt.Sprintf("Starting phase: %s", phase.Name))

	// Setup
	if s, ok := raw["setup"]; ok {
		err := sendRequest(logger, s.(map[string]any), host, port)
		if err != nil {
			return err
		}
	}

	// Run workload
	var wg sync.WaitGroup
	wg.Add(int(phase.Client.Workers))

	deadline := time.Now().Add(phase.Duration)

	var mu sync.Mutex

	var stats statistics
	allStatusCodes := make(map[int]int)

	phaseStart := time.Now()

	payloadBytes, err := json.Marshal(raw["workload"])
	if err != nil {
		return err
	}

	for i := 0; i < int(phase.Client.Workers); i++ {
		go func(stats *statistics) {
			defer wg.Done()

			statusCodes := make(map[int]int)

			// Use 10sec client timeout by default, but allow the client to set a custom timeout
			timeout := time.Duration(10) * time.Second
			if phase.Client.Timeout != 0 {
				timeout = phase.Client.Timeout
			}
			// Send the request to the server
			client := &http.Client{
				Timeout: timeout,
			}

			for {
				time.Sleep(phase.Client.Delay)

				start := time.Now()

				if start.After(deadline) {
					break
				}

				// Create a new HTTP POST request with the payload
				req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d/", host, port), bytes.NewBuffer(payloadBytes))
				if err != nil {
					logger.Error("failed to create request", zap.Error(err))
					return
				}

				// Set the content type header to indicate a JSON payload
				req.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(req)
				took := time.Since(start)

				if err != nil {
					atomic.AddUint64(&stats.countAllErrors, 1)
					continue
				}

				if resp.StatusCode == 400 {
					s, err := io.ReadAll(resp.Body)
					if err != nil {
						continue
					}

					logger.Warn(string(s))
				} else if resp.StatusCode > 400 {
					atomic.AddUint64(&stats.countAllErrors, 1)
				}

				atomic.AddUint64(&stats.sumAllDurationMicroseconds, uint64(took.Microseconds()))
				atomic.AddUint64(&stats.countAllRequests, 1)

				statusCodes[resp.StatusCode] += 1
				resp.Body.Close()
			}

			mu.Lock()
			defer mu.Unlock()

			for k, v := range statusCodes {
				allStatusCodes[k] += v
			}
		}(&stats)
	}

	go func(stats *statistics) {
		var last statistics
		interval := 10
		t := time.NewTicker(time.Duration(interval) * time.Second)

		for {
			<-t.C
			current := *stats

			duration := current.sumAllDurationMicroseconds - last.sumAllDurationMicroseconds
			requests := current.countAllRequests - last.countAllRequests
			errors := current.countAllErrors - last.countAllErrors

			var latency time.Duration
			if requests > 0 {
				latency = time.Duration(int64(duration/requests)) * time.Microsecond
			}

			last = current
			logger.Info(fmt.Sprintf("%v req/s, avg latency %v, %v errors", float64(requests)/float64(interval), latency, errors))
			if time.Now().After(deadline) {
				return
			}
		}
	}(&stats)

	wg.Wait()

	// Teardown
	if t, ok := raw["teardown"]; ok {
		err = sendRequest(logger, t.(map[string]any), host, port)
		if err != nil {
			return err
		}
	}

	var a time.Duration

	successfulRequests := stats.countAllRequests - stats.countAllErrors

	if successfulRequests > 0 {
		a = time.Duration(int64(stats.sumAllDurationMicroseconds/successfulRequests)) * time.Microsecond
	}

	logger.Info("")
	logger.Info("Phase Summary")
	logger.Info(fmt.Sprintf("Delay: %v", phase.Client.Delay))
	logger.Info(fmt.Sprintf("Took: %v", time.Since(phaseStart)))
	logger.Info(fmt.Sprintf("Workers: %v", phase.Client.Workers))
	logger.Info(fmt.Sprintf("Requests: %v", stats.countAllRequests))
	logger.Info(fmt.Sprintf("Errors: %v", stats.countAllErrors))
	logger.Info(fmt.Sprintf("Average Request Duration: %v", a))
	logger.Info("")
	logger.Info("Status codes:")
	for code, count := range allStatusCodes {
		logger.Info(fmt.Sprintf("%v: %v", code, count))
	}

	return nil
}

func command_client(logger *zap.Logger, ctx *cli.Context) error {
	planPath := ctx.String("plan")
	host := ctx.String("host")
	port := ctx.Int64("port")

	plan, raw, err := loadPlan(logger, planPath)
	if err != nil {
		return err
	}

	logger.Info("Successfully loaded execution plan")
	logger.Info(fmt.Sprintf("Phases: %d", len(plan.Phases)))

	var totalSeconds int
	for _, phase := range plan.Phases {
		totalSeconds += int(phase.Duration.Seconds())
	}

	logger.Info(fmt.Sprintf("Total estimated execution time: %s", time.Duration(totalSeconds*int(time.Second))))

	for i, phase := range plan.Phases {
		err := executePhase(logger, phase, raw["phases"].([]any)[i].(map[string]any), host, port)
		if err != nil {
			return err
		}
	}

	return nil
}
