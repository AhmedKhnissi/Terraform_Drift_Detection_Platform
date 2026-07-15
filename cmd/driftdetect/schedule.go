package driftdetect

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"driftdetect/internal/schedule"
)

var (
	schedEvery string
	schedSpec  string
)

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Run scans on a recurring schedule (long-running daemon)",
	RunE:  runSchedule,
}

func init() {
	scheduleCmd.Flags().StringVar(&schedEvery, "every", "", "run every duration, e.g. 15m, 1h (converted to @every)")
	scheduleCmd.Flags().StringVar(&schedSpec, "spec", "", "raw cron expression, e.g. \"*/15 * * * *\"")
}

func runSchedule(cmd *cobra.Command, _ []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := loadConfig()

	// Resolve the schedule spec: --every (duration) > --spec (cron) > config.
	spec := schedSpec
	if schedEvery != "" {
		if _, err := time.ParseDuration(schedEvery); err != nil {
			return fmt.Errorf("invalid --every duration %q: %w", schedEvery, err)
		}
		spec = "@every " + schedEvery
	}
	if spec == "" {
		spec = cfg.Schedule.Spec
	}
	if spec == "" {
		return fmt.Errorf("no schedule specified (use --every, --spec, or set schedule.spec in config)")
	}

	store, err := buildStore(cfg)
	if err != nil {
		return err
	}
	defer store.Close()

	awsCfg, err := buildAWSConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("aws config: %w", err)
	}
	runner, err := buildRunner(ctx, cfg, awsCfg)
	if err != nil {
		return err
	}

	sched := schedule.New()
	if err := sched.Add(spec, func(ctx context.Context) {
		rep, err := runner.Run(ctx)
		if err != nil {
			log.Printf("[scheduler] scan failed: %v", err)
			return
		}
		if err := store.SaveScan(rep); err != nil {
			log.Printf("[scheduler] save failed: %v", err)
			return
		}
		log.Printf("[scheduler] scan %s complete: %d resources, %d drift", rep.ScanID, rep.ResourceCount, rep.DriftCount)
	}); err != nil {
		return fmt.Errorf("invalid schedule %q: %w", spec, err)
	}

	sched.Start()
	defer sched.Stop()
	log.Printf("scheduler running with spec %q (Ctrl-C to stop)", spec)

	<-ctx.Done()
	return nil
}
