package cleanup

import (
	"context"
	"fmt"
	"sync"

	"github.com/l8s/l8s/pkg/logging"
)

type CleanupFunc func(context.Context) error

type Cleaner struct {
	mu       sync.Mutex
	cleanups []namedCleanup
	logger   Logger
}

type namedCleanup struct {
	name string
	fn   CleanupFunc
}

type Logger interface {
	Error(msg string, args ...any)
	Warn(msg string, args ...any)
}

func New(logger Logger) *Cleaner {
	if logger == nil {
		logger = logging.Default()
	}
	return &Cleaner{
		cleanups: make([]namedCleanup, 0),
		logger:   logger,
	}
}

func (c *Cleaner) Add(name string, fn CleanupFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cleanups = append(c.cleanups, namedCleanup{name: name, fn: fn})
}

func (c *Cleaner) Cleanup(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errs []error
	for i := len(c.cleanups) - 1; i >= 0; i-- {
		cleanup := c.cleanups[i]
		if err := cleanup.fn(ctx); err != nil {
			c.logger.Error("cleanup failed",
				logging.WithField("cleanup", cleanup.name),
				logging.WithError(err))
			errs = append(errs, fmt.Errorf("%s: %w", cleanup.name, err))
		}
	}

	c.cleanups = c.cleanups[:0]

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}
	return nil
}

func (c *Cleaner) CleanupOnError(ctx context.Context, errPtr *error) {
	if *errPtr != nil {
		if cleanupErr := c.Cleanup(ctx); cleanupErr != nil {
			c.logger.Warn("cleanup failed during error recovery",
				logging.WithError(cleanupErr))
		}
	}
}