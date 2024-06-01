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

func (sc *ScriptContext) Print(message string) error {
	cfg := PrintConfig{
		Message: message,
	}

	c, err := pkg.ConfigToMap(&cfg)
	if err != nil {
		return err
	}

	return ACTIONS["Print"].Execute(sc.Ctx, c)
}

func (sc *ScriptContext) Get_message() string {
	return sc.Message
}

func (sc *ScriptContext) Allocate_memory(size_bytes int, num_allocations int) error {
	cfg := AllocateMemoryConfig{
		SizeBytes:      size_bytes,
		NumAllocations: num_allocations,
	}

	c, err := pkg.ConfigToMap(&cfg)
	if err != nil {
		return err
	}

	return ACTIONS["AllocateMemory"].Execute(sc.Ctx, c)
}

func (sc *ScriptContext) Get_service(name string) (Service, error) {
	service, err := Manager.Get(ServiceName(name))
	if err != nil {
		return nil, err
	}

	return service, nil
}

func (sc *ScriptContext) Uuid() string {
	return uuid.New().String()
}

func (sc *ScriptContext) Random_string(n int64) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func (sc *ScriptContext) Sleep(duration string) error {
	cfg := SleepConfig{}

	d, err := time.ParseDuration(duration)
	if err != nil {
		return err
	}

	cfg.Duration = pkg.Duration{Duration: d}

	c, err := pkg.ConfigToMap(&cfg)
	if err != nil {
		return err
	}

	return ACTIONS["Sleep"].Execute(sc.Ctx, c)
}

func (sc *ScriptContext) Http_post(ctx context.Context, url string, body string) error {
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

	return ACTIONS["HTTPRequest"].Execute(sc.Ctx, c)
}

func (sc *ScriptContext) Burn(duration string) error {
	cfg := BurnConfig{}

	d, err := time.ParseDuration(duration)
	if err != nil {
		return err
	}

	cfg.Duration = pkg.Duration{Duration: d}

	c, err := pkg.ConfigToMap(&cfg)
	if err != nil {
		return err
	}

	return ACTIONS["Burn"].Execute(sc.Ctx, c)
}

func (sc *ScriptContext) Lock(id string) error {

	cfg := GlobalMutexLockConfig{
		Id: id,
	}
	c, err := pkg.ConfigToMap(&cfg)
	if err != nil {
		return err
	}

	return ACTIONS["GlobalMutexLock"].Execute(sc.Ctx, c)
}

func (sc *ScriptContext) UnLock(id string) error {

	cfg := GlobalMutexLockConfig{
		Id: id,
	}
	c, err := pkg.ConfigToMap(&cfg)
	if err != nil {
		return err
	}

	return ACTIONS["GlobalMutexUnlock"].Execute(sc.Ctx, c)
}

func (s *Script) Execute(ctx context.Context, cfg map[string]any) error {
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
	err = vm.Set("ctx", scriptContext)
	if err != nil {
		return err
	}

	run, ok := goja.AssertFunction(vm.Get("run"))
	if !ok {
		return errors.New("failed to get run function")
	}

	_, err = run(goja.Undefined())
	return err
}

func (s *Script) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[ScriptConfig](data)
}

func init() {
	ACTIONS["Script"] = &Script{}
}
