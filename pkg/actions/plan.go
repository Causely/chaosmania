package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/Causely/chaosmania/pkg/logger"
	"github.com/rotisserie/eris"
	"go.uber.org/zap"
)

type Action interface {
	Execute(context.Context, map[string]any) error
	ParseConfig(map[string]any) (any, error)
}

var ACTIONS map[string]Action = make(map[string]Action)

type ActionConfig struct {
	Name   string         `json:"name"`
	Config map[string]any `json:"config"`
}

type Workers struct {
	Instances uint          `json:"instances" yaml:"instances"`
	Duration  time.Duration `json:"duration" yaml:"duration"`
	Delay     time.Duration `json:"delay" yaml:"delay"`
	Timeout   time.Duration `json:"timeout" yaml:"timeout"`
}

type Phase struct {
	Name     string   `json:"name" yaml:"name"`
	Client   Client   `json:"client" yaml:"client"`
	Setup    Workload `json:"setup" yaml:"setup"`
	Workload Workload `json:"workload" yaml:"workload"`
	Teardown Workload `json:"teardown" yaml:"teardown"`
	Repeat   uint     `json:"repeat" yaml:"repeat"`
}

type Workload struct {
	Actions []ActionConfig `yaml:"actions" json:"actions"`
}

type Client struct {
	Workers []Workers `yaml:"workers" json:"workers"`
}

// Plan defines the structure of a chaos test plan
type Plan struct {
	Pattern PhasePattern `yaml:"pattern"`
	Phases  []Phase      `yaml:"phases"`
}

func (plan *Plan) Verify() error {
	for i, phase := range plan.Phases {
		// Verify worker durations
		for j, worker := range phase.Client.Workers {
			if worker.Duration == 0 {
				return fmt.Errorf("phase %d worker %d: worker duration is required", i+1, j+1)
			}
			if worker.Duration < MinPhaseDuration {
				return fmt.Errorf("phase %d worker %d: worker duration %v is less than minimum allowed duration %v (this will be adjusted at runtime)",
					i+1, j+1, worker.Duration, MinPhaseDuration)
			}
			if worker.Duration > MaxPhaseDuration {
				return fmt.Errorf("phase %d worker %d: worker duration %v exceeds maximum allowed duration %v (this will be adjusted at runtime)",
					i+1, j+1, worker.Duration, MaxPhaseDuration)
			}
		}

		err := phase.Verify()
		if err != nil {
			return err
		}
	}

	return nil
}

func (workload *Workload) Verify() error {
	for _, action := range workload.Actions {
		a := ACTIONS[action.Name]
		if a == nil {
			return eris.New(fmt.Sprintf("Unknown action: %v", action.Name))
		}

		_, err := a.ParseConfig(action.Config)
		if err != nil {
			return err
		}
	}

	return nil
}

func (phase *Phase) Verify() error {
	err := phase.Setup.Verify()
	if err != nil {
		return err
	}

	err = phase.Workload.Verify()
	if err != nil {
		return err
	}

	err = phase.Teardown.Verify()
	if err != nil {
		return err
	}

	return nil
}

func (workload *Workload) Execute(ctx context.Context) error {
	for _, action := range workload.Actions {
		a := ACTIONS[action.Name]
		_, err := a.ParseConfig(action.Config)
		if err != nil {
			logger.FromContext(ctx).Warn("failed to parse config", zap.Error(err), zap.String("action", action.Name))
			return err
		}

		if ctx.Err() != nil {
			logger.FromContext(ctx).Warn("context error", zap.Error(ctx.Err()), zap.String("action", action.Name))
		}

		logger.FromContext(ctx).Info("executing action", zap.String("action", action.Name), zap.Any("config", action.Config))
		err = a.Execute(ctx, action.Config)
		if err != nil {
			logger.FromContext(ctx).Warn("action execution failed", zap.Error(err), zap.String("action", action.Name))
			return err
		}
	}

	return nil
}
