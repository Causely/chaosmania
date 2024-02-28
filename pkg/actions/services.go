package actions

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type ServiceType string
type ServiceName string

type ServiceConstructor func(ServiceName, map[string]any) Service

var SERVICE_TYPES map[ServiceType]ServiceConstructor = make(map[ServiceType]ServiceConstructor)
var Manager *ServiceManager = NewServiceManager()

type Service interface {
	Name() ServiceName
	Type() ServiceType
}

type ServiceDeclaration struct {
	Name   ServiceName    `yaml:"name"`
	Type   ServiceType    `yaml:"type"`
	Config map[string]any `yaml:"config"`
}

type Services struct {
	Services []ServiceDeclaration `yaml:"services"`
}

type ServiceManager struct {
	services map[ServiceName]Service
}

func NewServiceManager() *ServiceManager {
	return &ServiceManager{
		services: make(map[ServiceName]Service),
	}
}

func (sm *ServiceManager) Register(s Service) error {
	if _, ok := sm.services[s.Name()]; ok {
		return errors.New("service already registered")
	}

	fmt.Println("Registering service", s.Name(), "of type", s.Type())
	sm.services[s.Name()] = s
	return nil
}

func (sm *ServiceManager) Get(name ServiceName) (Service, error) {
	if _, ok := sm.services[name]; !ok {
		return nil, errors.New("service not found")
	}

	return sm.services[name], nil
}

func (sm *ServiceManager) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var config Services
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return err
	}

	for _, service := range config.Services {
		constructor, ok := SERVICE_TYPES[service.Type]
		if !ok {
			return errors.New("service type not found: " + string(service.Type))
		}

		s := constructor(service.Name, service.Config)
		err := sm.Register(s)
		if err != nil {
			return err
		}
	}

	return nil
}
