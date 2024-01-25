package actions

import (
	"context"
	"sync"

	"github.com/Causely/chaosmania/pkg"
)

type AllocateMemory struct {
	mu             sync.Mutex
	leakedData     [][]byte
	leakedDataSize int
}

type AllocateMemoryConfig struct {
	SizeBytes      int  `json:"sizeBytes"`
	NumAllocations int  `json:"numAllocations"`
	Leak           bool `json:"leak"`
	LeakLimitBytes int  `json:"leakLimitBytes"`
}

func (a *AllocateMemory) leak(config *AllocateMemoryConfig, data [][]byte) {
	if !config.Leak {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range data {
		if config.LeakLimitBytes > 0 && a.leakedDataSize > config.LeakLimitBytes {
			break
		}

		a.leakedDataSize += len(data[i])
		a.leakedData = append(a.leakedData, data[i])
	}
}

func (a *AllocateMemory) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := pkg.ParseConfig[AllocateMemoryConfig](cfg)
	if err != nil {
		return err
	}

	data := make([][]byte, 0)
	for i := 0; i < config.NumAllocations; i++ {
		d := make([]byte, config.SizeBytes)
		for j := 0; j < config.SizeBytes; j++ {
			d[j] = 1
		}

		data = append(data, d)
	}

	a.leak(config, data)

	// Make the compiler happy and use the variables
	var sizeTotal int
	for i := range data {
		sizeTotal += len(data[i])
	}

	return nil
}

func (a *AllocateMemory) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[AllocateMemoryConfig](data)
}

func init() {
	ACTIONS["AllocateMemory"] = &AllocateMemory{}
}
