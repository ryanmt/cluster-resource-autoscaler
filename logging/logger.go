package logging

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(isDevelopment bool) logr.Logger {
	var err error
	var zapConfig zap.Config

	// unless running with special envvar...
	if isDevelopment {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}
	if _, ok := os.LookupEnv("VERBOSE"); ok {
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.Level(-9))
	}
	zapLog, err := zapConfig.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize zap logger: %v", err))
	}

	return zapr.NewLogger(zapLog)
}

// NewContext hydrates the provided context with the provided Logger
func NewContext(ctx context.Context, logger logr.Logger) context.Context {
	return logr.NewContext(context.Background(), logger)
}

// FromContextOrDiscard provides a logger from the context or a logger that discards all messages
func FromContextOrDiscard(ctx context.Context) logr.Logger {
	return logr.FromContextOrDiscard(ctx)
}
