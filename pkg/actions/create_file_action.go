package actions

import (
	"context"
	"os"

	"github.com/Causely/chaosmania/pkg"
	"github.com/Causely/chaosmania/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type CreateFile struct{}

type CreateFileConfig struct {
	Directory string `json:"directory"`
	Size      int    `json:"size"`
	Delete    bool   `json:"delete"`
}

func (a *CreateFile) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := pkg.ParseConfig[CreateFileConfig](cfg)
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

	_, err = f.Write(make([]byte, config.Size))
	if err != nil {
		logger.FromContext(ctx).Warn("failed to write to file", zap.Error(err))
		return err
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

func (a *CreateFile) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[CreateFileConfig](data)
}

func init() {
	ACTIONS["CreateFile"] = &CreateFile{}
}
