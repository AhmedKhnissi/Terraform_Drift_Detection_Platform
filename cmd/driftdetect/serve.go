package driftdetect

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"driftdetect/internal/model"
	"driftdetect/internal/schedule"
	"driftdetect/internal/web"
)

var (
	serveAddr     string
	serveSchedule string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web dashboard (and optional scheduler)",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().StringVar(&serveAddr, "addr", "", "HTTP listen address (overrides config web.addr)")
	serveCmd.Flags().StringVar(&serveSchedule, "schedule", "", "cron spec or @every <dur> to run scans on a schedule")
}

func runServe(cmd *cobra.Command, _ []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := loadConfig()
	addr := serveAddr
	if addr == "" {
		addr = cfg.Web.Addr
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

	// scanFn runs a scan, persists it, and returns the report (used by both the
	// scheduler and the dashboard's on-demand trigger).
	scanFn := func(ctx context.Context) (model.DriftReport, error) {
		rep, err := runner.Run(ctx)
		if err != nil {
			return rep, err
		}
		if err := store.SaveScan(rep); err != nil {
			return rep, fmt.Errorf("save scan: %w", err)
		}
		return rep, nil
	}

	srv, err := web.NewServer(store, scanFn, addr)
	if err != nil {
		return err
	}

	// Optional recurring scans.
	spec := serveSchedule
	if spec == "" {
		spec = cfg.Schedule.Spec
	}
	if spec != "" {
		sched := schedule.New()
		if err := sched.Add(spec, func(ctx context.Context) {
			if _, err := scanFn(ctx); err != nil {
				log.Printf("[scheduler] scan failed: %v", err)
			} else {
				log.Printf("[scheduler] scan completed on schedule %q", spec)
			}
		}); err != nil {
			return fmt.Errorf("invalid schedule %q: %w", spec, err)
		}
		sched.Start()
		defer sched.Stop()
		log.Printf("scheduler active with spec %q", spec)
	}

	log.Printf("drift dashboard listening on http://%s", addr)
	return srv.Start(ctx)
}
