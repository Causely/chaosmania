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
}

func (a *CreateFile) Execute(ctx context.Context, cfg map[string]any) error {
	config, err := pkg.ParseConfig[CreateFileConfig](cfg)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to parse config", zap.Error(err))
		return err
	}

	filename := config.Directory + "/" + uuid.NewString()
	return os.WriteFile(filename, make([]byte, config.Size), 0644)
}

func (a *CreateFile) ParseConfig(data map[string]any) (any, error) {
	return pkg.ParseConfig[CreateFileConfig](data)
}

func init() {
	ACTIONS["CreateFile"] = &CreateFile{}
}
