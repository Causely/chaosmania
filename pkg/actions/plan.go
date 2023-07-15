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

type Phase struct {
	Name     string        `json:"name"`
	Duration time.Duration `json:"duration"`
	Client   Client        `json:"client"`
	Setup    Workload      `json:"setup"`
	Workload Workload      `json:"workload"`
	Teardown Workload      `json:"teardown"`
}

type Workload struct {
	Actions []ActionConfig `json:"actions"`
}

type Client struct {
	Delay   time.Duration `json:"delay"`
	Timeout time.Duration `json:"timeout"`
	Workers int           `json:"workers"`
}

type Plan struct {
	Phases []Phase `json:"phases"`
}

func (p *Plan) Verify() error {
	for _, phase := range p.Phases {
		err := phase.Verify()
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Workload) Verify() error {
	for _, action := range p.Actions {
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

func (p *Phase) Verify() error {
	err := p.Setup.Verify()
	if err != nil {
		return err
	}

	err = p.Workload.Verify()
	if err != nil {
		return err
	}

	err = p.Teardown.Verify()
	if err != nil {
		return err
	}

	return nil
}

func (p *Workload) Execute(ctx context.Context) error {
	for _, action := range p.Actions {
		a := ACTIONS[action.Name]
		_, err := a.ParseConfig(action.Config)
		if err != nil {
			logger.NewLogger().Warn("failed to parse config", zap.Error(err), zap.String("action", action.Name))
			return err
		}

		if ctx.Err() != nil {
			logger.NewLogger().Warn("context error", zap.Error(ctx.Err()), zap.String("action", action.Name))
		}

		err = a.Execute(ctx, action.Config)
		if err != nil {
			logger.NewLogger().Warn("action execution failed", zap.Error(err), zap.String("action", action.Name))
			return err
		}
	}

	return nil
}
