package actions

import (
	"context"
	"math/rand"
	"os"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type RandomIO struct{}

type RandomIOConfig struct {
	Directory      string  `json:"directory"`
	Delete         bool    `json:"delete"`
	FileSize       int64   `json:"file_size"`
	BlockSize      int64   `json:"block_size"`
	IoCount        int     `json:"io_count"`
	ReadPercentage float32 `json:"read_percentage"`
}

func (a *RandomIO) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := pkg.ParseConfig[RandomIOConfig](cfg)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to parse config", zap.Error(err))
		return err
	}

	filename := config.Directory + "/" + uuid.NewString()

	f, err := os.Create(filename)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to create file", zap.Error(err))
		return err
	}

	defer f.Close()

	// Ensure the file is of the specified size
	err = f.Truncate(config.FileSize)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to truncate file", zap.Error(err))
		return err
	}

	buffer := make([]byte, config.BlockSize)
	for i := 0; i < config.IoCount; i++ {
		offset := rand.Int63n(config.FileSize/config.BlockSize) * config.BlockSize

		if rand.Float32() < config.ReadPercentage {
			_, err = f.ReadAt(buffer, offset)
			if err != nil {
				logger.FromContext(ctx).Warn("failed to read from file", zap.Error(err))
			}
		} else {
			_, err := f.WriteAt(buffer, offset)
			if err != nil {
				logger.FromContext(ctx).Warn("failed to write to file", zap.Error(err))
			}
		}
	}

	err = f.Sync()
	if err != nil {
		logger.FromContext(ctx).Warn("failed to sync file", zap.Error(err))
		return err
	}

	if config.Delete {
		err = os.Remove(filename)
		if err != nil {
			logger.FromContext(ctx).Warn("failed to delete file", zap.Error(err))
			return err
		}
	}

	return nil
}

func (a *RandomIO) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[RandomIOConfig](data)
}

func init() {
	ACTIONS["RandomIO"] = &RandomIO{}
}
