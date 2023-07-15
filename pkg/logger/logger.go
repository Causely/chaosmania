package logger

import (
	"go.uber.org/zap"
)

func NewLogger() *zap.Logger {
	childLogger, _ := zap.NewProduction()

	return childLogger
}
