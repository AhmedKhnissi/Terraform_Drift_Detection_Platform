package driftdetect

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"driftdetect/internal/output"
)

var (
	scanFormat string
	scanOut    string
	scanSave   bool
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Run a drift scan on demand",
	RunE:  runScan,
}

func init() {
	scanCmd.Flags().StringVar(&scanFormat, "format", "table", "output format: table | json")
	scanCmd.Flags().StringVar(&scanOut, "out", "", "write JSON output to this file (implies json format)")
	scanCmd.Flags().BoolVar(&scanSave, "save", false, "persist the scan result to storage")
}

func runScan(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()
	cfg := loadConfig()

	awsCfg, err := buildAWSConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("aws config: %w", err)
	}
	runner, err := buildRunner(ctx, cfg, awsCfg)
	if err != nil {
		return err
	}

	report, err := runner.Run(ctx)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	if scanSave {
		store, err := buildStore(cfg)
		if err != nil {
			return err
		}
		defer store.Close()
		if err := store.SaveScan(report); err != nil {
			return fmt.Errorf("save scan: %w", err)
		}
		cmd.Printf("scan %s saved\n", report.ScanID)
	}

	format := scanFormat
	if scanOut != "" {
		format = "json"
	}
	switch format {
	case "json":
		if scanOut != "" {
			return output.WriteJSON(scanOut, report)
		}
		return output.WriteJSONStdout(report)
	default:
		output.RenderTable(cmd.OutOrStdout(), report)
	}
	return nil
}
