package driftdetect

import (
	"github.com/spf13/cobra"

	"driftdetect/internal/output"
)

var (
	reportScanID string
	reportFormat string
	reportOut    string
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Print a stored scan's drift report",
	RunE:  runReport,
}

func init() {
	reportCmd.Flags().StringVar(&reportScanID, "scan", "", "scan id to display (required)")
	reportCmd.Flags().StringVar(&reportFormat, "format", "table", "output format: table | json")
	reportCmd.Flags().StringVar(&reportOut, "out", "", "write JSON output to this file")
	_ = reportCmd.MarkFlagRequired("scan")
}

func runReport(cmd *cobra.Command, _ []string) error {
	cfg := loadConfig()
	store, err := buildStore(cfg)
	if err != nil {
		return err
	}
	defer store.Close()

	report, err := store.GetReport(reportScanID)
	if err != nil {
		return err
	}

	format := reportFormat
	if reportOut != "" {
		format = "json"
	}
	switch format {
	case "json":
		if reportOut != "" {
			return output.WriteJSON(reportOut, report)
		}
		return output.WriteJSONStdout(report)
	default:
		output.RenderTable(cmd.OutOrStdout(), report)
	}
	return nil
}
