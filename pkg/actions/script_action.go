package actions

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"time"

	"github.com/Causely/chaosmania/pkg"
	"github.com/dop251/goja"
	"github.com/google/uuid"
)

type Script struct{}

type ScriptConfig struct {
	Script  string `json:"script"`
	Message string `json:"message"`
}

type ScriptContext struct {
	Ctx     context.Context
	Message string
}

func (s *ScriptContext) Print(message string) error {
	cfg := PrintConfig{
		Message: message,
	}

	c, err := pkg.ConfigToMap(&cfg)
	if err != nil {
		return err
	}

	return ACTIONS["Print"].Execute(s.Ctx, c)
}

func (s *ScriptContext) Get_message() string {
	return s.Message
}

func (s *ScriptContext) Allocate_memory(size_bytes int, num_allocations int) error {
	cfg := AllocateMemoryConfig{
		SizeBytes:      size_bytes,
		NumAllocations: num_allocations,
	}

	c, err := pkg.ConfigToMap(&cfg)
	if err != nil {
		return err
	}

	return ACTIONS["AllocateMemory"].Execute(s.Ctx, c)
}

func (s *ScriptContext) Get_service(name string) (Service, error) {
	service, err := Manager.Get(ServiceName(name))
	if err != nil {
		return nil, err
	}

	return service, nil
}

func (s *ScriptContext) Uuid() string {
	return uuid.New().String()
}

func (s *ScriptContext) Random_string(n int64) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func (s *ScriptContext) Sleep(duration string) error {
	cfg := SleepConfig{}

	d, err := time.ParseDuration(duration)
	if err != nil {
		return err
	}

	cfg.Duration = pkg.Duration{d}

	c, err := pkg.ConfigToMap(&cfg)
	if err != nil {
		return err
	}

	return ACTIONS["Sleep"].Execute(s.Ctx, c)
}

func (s *ScriptContext) Http_post(ctx context.Context, url string, body string) error {
	b := make(map[string]any)

	err := json.Unmarshal([]byte(body), &b)
	if err != nil {
		return err
	}

	cfg := HTTPRequestConfig{
		Body: b,
		Url:  url,
	}

	c, err := pkg.ConfigToMap(&cfg)
	if err != nil {
		return err
	}

	return ACTIONS["HTTPRequest"].Execute(s.Ctx, c)
}

func (s *ScriptContext) Burn(duration string) error {
	cfg := BurnConfig{}

	d, err := time.ParseDuration(duration)
	if err != nil {
		return err
	}

	cfg.Duration = pkg.Duration{d}

	c, err := pkg.ConfigToMap(&cfg)
	if err != nil {
		return err
	}

	return ACTIONS["Burn"].Execute(s.Ctx, c)
}

func (a *Script) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := pkg.ParseConfig[ScriptConfig](cfg)
	if err != nil {
		return err
	}

	vm := goja.New()
	_, err = vm.RunString(config.Script)
	if err != nil {
		return err
	}

	scriptContext := &ScriptContext{
		Ctx:     ctx,
		Message: config.Message,
	}

	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())
	vm.Set("ctx", scriptContext)

	run, ok := goja.AssertFunction(vm.Get("run"))
	if !ok {
		return errors.New("failed to get run function")
	}

	_, err = run(goja.Undefined())
	return err
}

func (a *Script) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[ScriptConfig](data)
}

func init() {
	ACTIONS["Script"] = &Script{}
}
