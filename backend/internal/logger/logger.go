package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a production-ready zap logger
func New(debug bool) (*zap.Logger, error) {
	var config zap.Config

	if debug {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	return config.Build()
}

// MustNew creates logger or panics (only for main())
func MustNew(debug bool) *zap.Logger {
	logger, err := New(debug)
	if err != nil {
		panic(err)
	}
	return logger
}
