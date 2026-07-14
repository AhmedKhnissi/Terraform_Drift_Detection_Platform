// Package schedule runs drift scans on a recurring cron schedule.
package schedule

import (
	"context"

	"github.com/robfig/cron/v3"
)

// Scheduler wraps a cron engine that invokes a scan function on a schedule.
type Scheduler struct {
	c *cron.Cron
}

// New creates an idle scheduler.
func New() *Scheduler {
	return &Scheduler{c: cron.New()}
}

// Add registers fn to run on the given cron expression (standard 5-field).
func (s *Scheduler) Add(spec string, fn func(ctx context.Context)) error {
	_, err := s.c.AddFunc(spec, func() {
		fn(context.Background())
	})
	return err
}

// Start begins executing scheduled functions (non-blocking).
func (s *Scheduler) Start() {
	s.c.Start()
}

// Stop halts the scheduler and waits for in-flight jobs to finish.
func (s *Scheduler) Stop() {
	<-s.c.Stop().Done()
}
