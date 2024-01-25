package actions

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

type BackgroundServiceType string
type BackgroundServiceName string

type BackgroundServiceConstructor func(BackgroundServiceName, map[string]any) BackgroundService

var BACKGROUND_SERVICE_TPES map[BackgroundServiceType]BackgroundServiceConstructor = make(map[BackgroundServiceType]BackgroundServiceConstructor)
var BackgroundManager *BackgroundServiceManager = NewBackgroundServiceManager()

type BackgroundService interface {
	Name() BackgroundServiceName
	Type() BackgroundServiceType
	Run(context.Context) error
}

type BackgroundServiceDeclaration struct {
	Name   BackgroundServiceName `yaml:"name"`
	Type   BackgroundServiceType `yaml:"type"`
	Config map[string]any        `yaml:"config"`
}

type BackgroundServices struct {
	Services []BackgroundServiceDeclaration `yaml:"services"`
}

type ManagedBackgroundService struct {
	Service BackgroundService
}

type BackgroundServiceManager struct {
	services map[BackgroundServiceName]*ManagedBackgroundService
	context  context.Context
}

func NewBackgroundServiceManager() *BackgroundServiceManager {
	return &BackgroundServiceManager{
		services: make(map[BackgroundServiceName]*ManagedBackgroundService),
	}
}

func (m *BackgroundServiceManager) Run(ctx context.Context) {
	m.context = ctx

	for _, service := range m.services {
		go func(service *ManagedBackgroundService) {
			err := service.Service.Run(ctx)
			if err != nil {
				fmt.Println("service", service.Service.Name(), "failed to run:", err)
			}
		}(service)
	}
}

func (m *BackgroundServiceManager) Register(s BackgroundService) error {
	if _, ok := m.services[s.Name()]; ok {
		return errors.New("service already registered")
	}

	fmt.Println("Registering background service", s.Name(), "of type", s.Type())
	m.services[s.Name()] = &ManagedBackgroundService{
		Service: s,
	}

	return nil
}

func (m *BackgroundServiceManager) LoadFromFile(path string) error {
	enabledServices := os.Getenv("ENABLED_BACKGROUND_SERVICES")
	if enabledServices == "" {
		return nil
	}

	serices := make(map[string]bool)
	for _, service := range strings.Split(enabledServices, ",") {
		serices[service] = true
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var config BackgroundServices
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return err
	}

	for _, service := range config.Services {
		if !serices[string(service.Name)] {
			continue
		}

		constructor, ok := BACKGROUND_SERVICE_TPES[service.Type]
		if !ok {
			return errors.New("background service type not found: " + string(service.Type))
		}

		s := constructor(service.Name, service.Config)
		err := m.Register(s)
		if err != nil {
			return err
		}
	}

	return nil
}
