package shutdown

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// GracefulShutdown orchestrates ordered cleanup on SIGINT/SIGTERM.
// Registered functions are called sequentially with a shared timeout context.
type GracefulShutdown struct {
	logger  *zap.Logger
	stopFns []func(context.Context) error
	timeout time.Duration
}

// New creates a GracefulShutdown manager with the given logger and timeout.
func New(logger *zap.Logger, timeout time.Duration) *GracefulShutdown {
	return &GracefulShutdown{
		logger:  logger,
		timeout: timeout,
	}
}

// Register adds a cleanup function to be called during shutdown.
func (g *GracefulShutdown) Register(fn func(context.Context) error) {
	g.stopFns = append(g.stopFns, fn)
}

// Wait blocks until SIGINT or SIGTERM is received, then runs all registered
// cleanup functions sequentially within the configured timeout.
func (g *GracefulShutdown) Wait() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	g.logger.Info("shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()

	for i, fn := range g.stopFns {
		if err := fn(ctx); err != nil {
			g.logger.Error("shutdown error",
				zap.Int("step", i),
				zap.Error(err),
			)
		}
	}

	g.logger.Info("shutdown complete")
}
